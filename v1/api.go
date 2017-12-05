package v1

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	jwt "github.com/dgrijalva/jwt-go"
	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/auth"
	"github.com/golang/glog"
	"github.com/guregu/dynamo"
	"github.com/labstack/echo"
	"github.com/mobingilabs/authd/pkg/token"
	"github.com/pkg/errors"
)

type event struct {
	User   string `dynamo:"username"`
	Pass   string `dynamo:"password"`
	Status string `dynamo:"status"`
}

type root struct {
	ApiToken string `json:"api_token"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

type ApiV1Config struct {
	PublicPemFile  string
	PrivatePemFile string
	AwsRegion      string

	Issuer *token.TokenIssuer
}

type apiv1 struct {
	cnf *ApiV1Config
	prv []byte
	pub []byte
	e   *echo.Echo
	g   *echo.Group
}

type WrapperClaims struct {
	Data map[string]interface{}
	jwt.StandardClaims
}

type creds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *apiv1) token(c echo.Context) error {
	var stoken string
	var claims WrapperClaims
	var crds creds

	err := c.Bind(&crds)
	if err != nil {
		glog.Error(err)
	}

	// kid
	glog.Infof("%x", sha256.Sum256(a.pub))

	md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", crds.Password))))
	valid, err := a.checkdb(crds.Username, md5p)
	if err != nil {
		glog.Error(err)
	}

	glog.Info("valid: ", valid)

	m := make(map[string]interface{})
	m["username"] = crds.Username
	m["password"] = crds.Password
	claims.Data = m
	claims.ExpiresAt = time.Now().Add(time.Hour * 24).Unix()
	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)
	key, err := jwt.ParseRSAPrivateKeyFromPEM(a.prv)
	if err != nil {
		glog.Error(err)
	}

	stoken, err = token.SignedString(key)
	if err != nil {
		glog.Error(err)
	}

	// return token, stoken, nil
	return c.String(http.StatusOK, stoken)
}

func (a *apiv1) verify(c echo.Context) error {
	type token_t struct {
		Key string `json:"key"`
	}

	var tkn token_t

	err := c.Bind(&tkn)
	if err != nil {
		glog.Error(err)
	}

	glog.Info("token received: ", tkn.Key)

	key, err := jwt.ParseRSAPublicKeyFromPEM(a.pub)
	if err != nil {
		glog.Error(err)
	}

	var claims WrapperClaims

	t, err := jwt.ParseWithClaims(tkn.Key, &claims, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", tk.Header["alg"])
		}

		return key, nil
	})

	if err != nil {
		glog.Error(err)
	}

	glog.Info(t.Raw, ", ", t.Valid)
	return c.String(http.StatusOK, t.Raw)
}

func (a *apiv1) checkdb(uname string, pwdmd5 string) (bool, error) {
	sess := session.Must(session.NewSession())
	db := dynamo.New(sess, &aws.Config{
		Region: aws.String(a.cnf.AwsRegion),
	})

	var results []event
	var ret bool

	// look in subusers first
	table := db.Table("MC_IDENTITY")
	err := table.Get("username", uname).All(&results)
	for _, data := range results {
		if pwdmd5 == data.Pass && data.Status != "deleted" {
			glog.Info("valid subuser: ", uname)
			return true, nil
		}
	}

	if err != nil {
		glog.Error("error in table get: ", err)
	}

	// try looking at the root users table
	var queryInput = &dynamodb.QueryInput{
		TableName:              aws.String("MC_USERS"),
		IndexName:              aws.String("email-index"),
		KeyConditionExpression: aws.String("email = :e"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":e": {
				S: aws.String(uname),
			},
		},
	}

	dbsvc := dynamodb.New(sess, &aws.Config{
		Region: aws.String(a.cnf.AwsRegion),
	})

	resp, err := dbsvc.Query(queryInput)
	if err != nil {
		glog.Error(errors.Wrap(err, "query failed"))
	} else {
		ru := []root{}
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &ru)
		if err != nil {
			glog.Error(errors.Wrap(err, "dynamo obj unmarshal failed"))
		}

		glog.Info("raw: ", ru)

		// should be a valid root user
		for _, u := range ru {
			if u.Email == uname && u.Password == pwdmd5 {
				if u.Status == "" || u.Status == "trial" {
					glog.Info("valid root user: ", uname)
					ret = true
					break
				}
			}
		}
	}

	return ret, err
}

var (
	repositoryClassCache = map[string]string{}
	enforceRepoClass     bool
)

type acctSubject struct{}

func (acctSubject) String() string { return "acctSubject" }

type requestedAccess struct{}

func (requestedAccess) String() string { return "requestedAccess" }

type grantedAccess struct{}

func (grantedAccess) String() string { return "grantedAccess" }

func filterAccessList(ctx context.Context, scope string, requestedAccessList []auth.Access) []auth.Access {
	if !strings.HasSuffix(scope, "/") {
		scope = scope + "/"
	}
	grantedAccessList := make([]auth.Access, 0, len(requestedAccessList))
	for _, access := range requestedAccessList {
		if access.Type == "repository" {
			if !strings.HasPrefix(access.Name, scope) {
				dcontext.GetLogger(ctx).Debugf("Resource scope not allowed: %s", access.Name)
				continue
			}
			if enforceRepoClass {
				if class, ok := repositoryClassCache[access.Name]; ok {
					if class != access.Class {
						dcontext.GetLogger(ctx).Debugf("Different repository class: %q, previously %q", access.Class, class)
						continue
					}
				} else if strings.EqualFold(access.Action, "push") {
					repositoryClassCache[access.Name] = access.Class
				}
			}
		} else if access.Type == "registry" {
			if access.Name != "catalog" {
				dcontext.GetLogger(ctx).Debugf("Unknown registry resource: %s", access.Name)
				continue
			}
			// TODO: Limit some actions to "admin" users
		} else {
			dcontext.GetLogger(ctx).Debugf("Skipping unsupported resource type: %s", access.Type)
			continue
		}
		grantedAccessList = append(grantedAccessList, access)
	}
	return grantedAccessList
}

type tokenResponse struct {
	Token        string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

func (a *apiv1) dockerToken(c echo.Context) error {
	ctx := c.Request().Context()
	params := c.Request().URL.Query()
	service := params.Get("service")
	scopeSpecifiers := params["scope"]

	/*
		var offline bool
		if offlineStr := params.Get("offline_token"); offlineStr != "" {
			var err error
			offline, err = strconv.ParseBool(offlineStr)
			if err != nil {
				handleError(ctx, ErrorBadTokenOption.WithDetail(err), w)
				return
			}
		}
	*/

	// realm := "authd-realm"
	// passwdFile := "nil"

	/*
		ac, err := auth.GetAccessController("htpasswd", map[string]interface{}{
			"realm": realm,
			"path":  passwdFile,
		})
	*/

	glog.Info(scopeSpecifiers)

	requestedAccessList := token.ResolveScopeSpecifiers(ctx, scopeSpecifiers)
	glog.Info("requestedAccessList: ", requestedAccessList)

	/*
		authorizedCtx, err := ac.Authorized(ctx, requestedAccessList...)
		if err != nil {
			challenge, ok := err.(auth.Challenge)
			if !ok {
				// handleError(ctx, err, w)
				// return
				glog.Error("challenge not ok")
			}

			// Get response context.
			// ctx, w = dcontext.WithResponseWriter(ctx, w)

			challenge.SetHeaders(c.Response())
			// handleError(ctx, errcode.ErrorCodeUnauthorized.WithDetail(challenge.Error()), w)

			// dcontext.GetResponseLogger(ctx).Info("get token authentication challenge")

			// return
			c.String(http.StatusOK, "hello")
			return nil
		}
	*/

	// ctx = authorizedCtx

	// username := dcontext.GetStringValue(ctx, "auth.user.name")
	username := "subuser01"

	/*
		ctx = context.WithValue(ctx, acctSubject{}, username)
		ctx = dcontext.WithLogger(ctx, dcontext.GetLogger(ctx, acctSubject{}))

		dcontext.GetLogger(ctx).Info("authenticated client")

		ctx = context.WithValue(ctx, requestedAccess{}, requestedAccessList)
		ctx = dcontext.WithLogger(ctx, dcontext.GetLogger(ctx, requestedAccess{}))
	*/

	grantedAccessList := filterAccessList(ctx, username, requestedAccessList)
	glog.Info("grantedAccessList: ", grantedAccessList)

	ctx = context.WithValue(ctx, grantedAccess{}, grantedAccessList)
	ctx = dcontext.WithLogger(ctx, dcontext.GetLogger(ctx, grantedAccess{}))

	// token, err := ts.issuer.CreateJWT(username, service, grantedAccessList)
	if a.cnf.Issuer == nil {
		glog.Info("nil issuer")
	}

	token, err := a.cnf.Issuer.CreateJWT(username, service, grantedAccessList)
	if err != nil {
		// handleError(ctx, err, w)
		glog.Error(err)
		return nil
	}

	glog.Info("context: ", ctx)
	glog.Info("generated token: ", token)
	dcontext.GetLogger(ctx).Info("authorized client")

	response := tokenResponse{
		Token:     token,
		ExpiresIn: int(a.cnf.Issuer.Expiration.Seconds()),
	}

	/*
		if offline {
			response.RefreshToken = newRefreshToken()
			ts.refreshCache[response.RefreshToken] = refreshToken{
				subject: username,
				service: service,
			}
		}

		ctx, w = dcontext.WithResponseWriter(ctx, w)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		dcontext.GetResponseLogger(ctx).Info("get token complete")
	*/

	c.JSON(http.StatusOK, response)
	return nil
}

func NewApiV1(e *echo.Echo, cnf *ApiV1Config) *apiv1 {
	bprv, err := ioutil.ReadFile(cnf.PrivatePemFile)
	if err != nil {
		glog.Error(err)
	}

	bpub, err := ioutil.ReadFile(cnf.PublicPemFile)
	if err != nil {
		glog.Error(err)
	}

	g := e.Group("/api/v1")
	api := &apiv1{
		cnf: cnf,
		prv: bprv,
		pub: bpub,
		e:   e,
		g:   g,
	}

	g.POST("/token", api.token)
	g.POST("/verify", api.verify)
	g.GET("/docker/token", api.dockerToken)

	return api
}
