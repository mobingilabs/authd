package v1

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/guregu/dynamo"
	"github.com/labstack/echo"
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
		glog.Errorf("bind failed: %v", err)
	}

	md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", crds.Password))))
	valid, err := a.checkdb(crds.Username, md5p)
	if err != nil {
		glog.Errorf("checkdb failed: %v", err)
	}

	glog.Infof("valid: %v", valid)

	m := make(map[string]interface{})
	m["username"] = crds.Username
	m["password"] = crds.Password
	claims.Data = m
	claims.ExpiresAt = time.Now().Add(time.Hour * 24).Unix()
	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)
	key, err := jwt.ParseRSAPrivateKeyFromPEM(a.prv)
	if err != nil {
		glog.Errorf("parse private key from pem failed: %v", err)
	}

	stoken, err = token.SignedString(key)
	if err != nil {
		glog.Errorf("signed string failed: %v", err)
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
		glog.Errorf("bind failed: %v", err)
	}

	glog.Infof("token received: %v", tkn.Key)

	key, err := jwt.ParseRSAPublicKeyFromPEM(a.pub)
	if err != nil {
		glog.Errorf("parse public key from pem failed: %v", err)
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
	}

	glog.Infof("token raw: %v, valid: %v", t.Raw, t.Valid)
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
		Region: aws.String(a.cnf.AwsRegion),
	})

	resp, err := dbsvc.Query(queryInput)
	if err != nil {
		glog.Errorf("query failed: %v", err)
	} else {
		ru := []root{}
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &ru)
		if err != nil {
			glog.Errorf("dynamo unmarshal failed: %v", err)
		}

		glog.Infof("root (raw): %v", ru)

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
	}

	return ret, err
}

func NewApiV1(e *echo.Echo, cnf *ApiV1Config) *apiv1 {
	bprv, err := ioutil.ReadFile(cnf.PrivatePemFile)
	if err != nil {
		glog.Errorf("readfile (private) failed: %v", err)
	}

	bpub, err := ioutil.ReadFile(cnf.PublicPemFile)
	if err != nil {
		glog.Errorf("readfile (public) failed: %v", err)
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

	return api
}