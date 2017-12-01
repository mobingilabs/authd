package main

import (
	"github.com/astaxie/beego"
	"github.com/golang/glog"
)

type status struct {
	Status      string `json:"status"`
	Description string `json:"description"`
}

type ApiController struct {
	beego.Controller
	noAuth bool
}

func (c *ApiController) Prepare() {
	// do auth by default
	if !c.noAuth {
		glog.Info("(todo) auth")
	}
}

func (c *ApiController) DispatchScratch() {
	type x struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	c.Data["json"] = x{Name: "foo", Value: "bar"}
	c.ServeJSON()
}

func (c *ApiController) DispatchRoot() {
	c.Ctx.ResponseWriter.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
}

func (c *ApiController) DispatchVersion() {
	c.Ctx.ResponseWriter.Write([]byte(version))
}

func (c *ApiController) DispatchToken() {
	m := make(map[string]interface{})
	tokenobj, stoken, err := tctx.GenerateToken(m)
	if err != nil {
		c.Data["json"] = c.serr(err)
		glog.Error(err)
		c.ServeJSON()
		return
	}

	glog.Info("token (obj):", tokenobj)
	glog.Info("token:", stoken)
	reply := make(map[string]string)
	reply["key"] = stoken
	c.Data["json"] = reply
	c.ServeJSON()

	/*
		var creds credentials

		c.info("body:", string(c.Ctx.Input.RequestBody))
		err = json.Unmarshal(c.Ctx.Input.RequestBody, &creds)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			c.err(errors.Wrap(err, "unmarshal body failed"))
			return
		}

		m := make(map[string]interface{})
		m["username"] = creds.Username
		m["password"] = creds.Password
		tokenobj, stoken, err := ctx.GenerateToken(m)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			c.err(errors.Wrap(err, "generate token failed"))
			return
		}

		c.info("token (obj):", tokenobj)
		c.info("token:", stoken)
		reply := make(map[string]string)
		reply["key"] = stoken
		c.Data["json"] = reply
		c.ServeJSON()
	*/
}

func (c *ApiController) serr(err error) status {
	return status{
		Status:      "error",
		Description: err.Error(),
	}
}

func (c *ApiController) sinfo(s string) status {
	return status{
		Status:      "success",
		Description: s,
	}
}
