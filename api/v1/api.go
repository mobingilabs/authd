package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/guregu/dynamo"
	"github.com/labstack/echo"
	sdkjwt "github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/oath/pkg/creds"
	"github.com/mobingilabs/oath/pkg/params"
	"github.com/mobingilabs/oath/pkg/util"
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
}

type apiv1 struct {
	prv []byte
	pub []byte
	e   *echo.Echo
	g   *echo.Group
}

type WrapperClaims struct {
	Data map[string]interface{}
	jwt.StandardClaims
}

type tokenPayload struct {
	Key string `json:"key"`
}

func (a *apiv1) simpleResponse(c echo.Context, code int, m string) error {
	resp := map[string]string{}
	resp["message"] = m
	return c.JSON(code, resp)
}

func (a *apiv1) elapsed(c echo.Context) {
	fn := c.Get("fnelapsed").(func(echo.Context))
	fn(c)
}

func (a *apiv1) token(c echo.Context) error {
	defer a.elapsed(c)

	var stoken string
	// var claims WrapperClaims

	ctx, err := sdkjwt.NewCtxWithConfig(&sdkjwt.Config{
		PublicTokenFile:  params.PublicPemFile,
		PrivateTokenFile: params.PrivatePemFile,
	})

	if err != nil {
		a.simpleResponse(c, http.StatusUnauthorized, err.Error())
		glog.Exitf("jwt ctx failed: %+v", util.ErrV(err))
	}

	var credentials creds.Credentials

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		glog.Errorf("readall body failed: %+v", util.ErrV(err))
		return err
	}

	defer c.Request().Body.Close()
	glog.Infof("body (raw): %v", string(body))
	err = json.Unmarshal(body, &credentials)
	if err != nil {
		glog.Errorf("unmarshal failed: %+v", util.ErrV(err))
		return err
	}

	glog.Infof("body: %+v", credentials)

	ok, err := credentials.Validate()
	if !ok {
		m := "credentials validation failed"
		a.simpleResponse(c, http.StatusInternalServerError, m)
		return errors.New(m)
	}

	if err != nil {
		a.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("credentials validation failed: %+v", util.ErrV(err))
		return err
	}

	m := make(map[string]interface{})
	m["username"] = credentials.Username
	m["password"] = credentials.Password
	tokenobj, stoken, err := ctx.GenerateToken(m)
	if err != nil {
		glog.Errorf("generate token failed: %+v", util.ErrV(err))
		return err
	}

	glog.Infof("token (obj): %v", tokenobj)
	glog.Infof("token: %v", stoken)

	reply := make(map[string]string)
	reply["key"] = stoken
	return c.JSON(http.StatusOK, reply)
}

func (a *apiv1) verify(c echo.Context) error {
	var tkn tokenPayload

	err := c.Bind(&tkn)
	if err != nil {
		glog.Errorf("bind failed: %v", err)
		return err
	}

	glog.Infof("body received: %+v", tkn)

	key, err := jwt.ParseRSAPublicKeyFromPEM(a.pub)
	if err != nil {
		glog.Errorf("parse public key from pem failed: %v", err)
		return err
	}

	var claims WrapperClaims

	t, err := jwt.ParseWithClaims(tkn.Key, &claims, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", tk.Header["alg"])
		}

		return key, nil
	})

	if err != nil {
		glog.Errorf("parse with claims failed: %v", err)
		return err
	}

	glog.Infof("token raw: %v, valid: %v", t.Raw, t.Valid)

	code := http.StatusOK
	msg := "valid token"
	if !t.Valid {
		code = http.StatusUnauthorized
		msg = "invalid token"
	}

	return a.simpleResponse(c, code, msg)
}

func (a *apiv1) checkdb(uname string, pwdmd5 string) (bool, error) {
	sess := session.Must(session.NewSession())
	db := dynamo.New(sess, &aws.Config{
		Region: aws.String(params.Region),
	})

	var results []event
	var ret bool

	// look in subusers first
	table := db.Table("MC_IDENTITY")
	err := table.Get("username", uname).All(&results)
	for _, data := range results {
		if pwdmd5 == data.Pass && data.Status != "deleted" {
			glog.Infof("valid subuser: %v", uname)
			return true, nil
		}
	}

	if err != nil {
		glog.Errorf("get table failed: %v", err)
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
		Region: aws.String(params.Region),
	})

	resp, err := dbsvc.Query(queryInput)
	if err != nil {
		glog.Errorf("query failed: %v", err)
		return ret, err
	}

	ru := []root{}
	err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &ru)
	if err != nil {
		glog.Errorf("dynamo unmarshal failed: %v", err)
		return ret, err
	}

	// should be a valid root user
	for _, u := range ru {
		if u.Email == uname && u.Password == pwdmd5 {
			if u.Status == "" || u.Status == "trial" {
				glog.Infof("valid root user: %v", uname)
				ret = true
				break
			}
		}
	}

	if !ret {
		return ret, errors.New("invalid user")
	}

	return ret, err
}

func NewApiV1(e *echo.Echo) *apiv1 {
	bprv, err := ioutil.ReadFile(params.PrivatePemFile)
	if err != nil {
		glog.Errorf("readfile (private) failed: %v", err)
	}

	bpub, err := ioutil.ReadFile(params.PublicPemFile)
	if err != nil {
		glog.Errorf("readfile (public) failed: %v", err)
	}

	g := e.Group("/api/v1")
	api := &apiv1{
		prv: bprv,
		pub: bpub,
		e:   e,
		g:   g,
	}

	g.POST("/token", api.token)
	g.POST("/verify", api.verify)

	return api
}
