package main

import (
	"fmt"
	"io/ioutil"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type WrapperClaims struct {
	Data map[string]interface{}
	jwt.StandardClaims
}

type tokenctx struct {
	Pub    []byte
	Prv    []byte
	PemPub string
	PemPrv string
	init   bool
}

func (j *tokenctx) GenerateToken(data map[string]interface{}) (*jwt.Token, string, error) {
	var stoken string
	var claims WrapperClaims

	claims.Data = data
	claims.ExpiresAt = time.Now().Add(time.Hour * 24).Unix()
	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), claims)
	key, err := jwt.ParseRSAPrivateKeyFromPEM(j.Prv)
	if err != nil {
		return token, stoken, errors.Wrap(err, "parse priv key from pem failed")
	}

	stoken, err = token.SignedString(key)
	if err != nil {
		return token, stoken, errors.Wrap(err, "signed string failed")
	}

	return token, stoken, nil
}

func (j *tokenctx) ParseToken(token string) (*jwt.Token, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(j.Pub)
	if err != nil {
		return nil, errors.Wrap(err, "ParseRSAPublicKeyFromPEM failed")
	}

	var claims WrapperClaims

	return jwt.ParseWithClaims(token, &claims, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", tk.Header["alg"])
		}

		return key, nil
	})
}

func NewCtx(pempub, pemprv string) (*tokenctx, error) {
	pubcache, err := ioutil.ReadFile(pempub)
	if err != nil {
		err = errors.Wrap(err, "pub readfile failed")
		glog.Error(err)
		return nil, err
	}

	prvcache, err := ioutil.ReadFile(pemprv)
	if err != nil {
		err = errors.Wrap(err, "prv readfile failed")
		glog.Error(err)
		return nil, err
	}

	ctx := tokenctx{
		PemPub: pempub,
		PemPrv: pemprv,
		Pub:    pubcache,
		Prv:    prvcache,
	}

	return &ctx, nil
}
