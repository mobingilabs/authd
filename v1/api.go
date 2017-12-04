package v1

import (
	"io/ioutil"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/labstack/echo"
)

type apiv1 struct {
	prv []byte
	e   *echo.Echo
	g   *echo.Group
}

type WrapperClaims struct {
	Data map[string]interface{}
	jwt.StandardClaims
}

func (a *apiv1) token(c echo.Context) error {
	var stoken string
	var claims WrapperClaims

	m := make(map[string]interface{})
	m["username"] = "foo"
	m["password"] = "bar"
	claims.Data = m
	claims.ExpiresAt = time.Now().Add(time.Hour * 24).Unix()
	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), claims)
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

func NewApiV1(e *echo.Echo, kf string) *apiv1 {
	b, err := ioutil.ReadFile(kf)
	if err != nil {
		glog.Error(err)
	}

	g := e.Group("/api/v1")
	api := &apiv1{
		prv: b,
		e:   e,
		g:   g,
	}

	g.POST("/token", api.token)

	return api
}
