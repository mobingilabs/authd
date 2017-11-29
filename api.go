package main

import (
	"github.com/astaxie/beego"
	"github.com/golang/glog"
)

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
	glog.Info("controller:", c)
	c.Ctx.ResponseWriter.Write([]byte("root006"))
}

func (c *ApiController) DispatchToken() {
	c.Ctx.ResponseWriter.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
}
