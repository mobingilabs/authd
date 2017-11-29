package main

import (
	goflag "flag"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/plugins/cors"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
		Use:   "authd",
		Short: "authorization and authentication service for Mobingi",
		Long:  "Authorization and authentication for Mobingi.",
		Run:   serve,
	}

	port string
)

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.Flags().StringVar(&port, "port", "8080", "server port")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func serve(cmd *cobra.Command, args []string) {
	goflag.Parse()

	beego.BConfig.ServerName = "mobingi:authd:1.0.0"
	beego.BConfig.RunMode = beego.DEV

	// needed for http input body in request to be available for non-get and head reqs
	beego.BConfig.CopyRequestBody = true

	beego.Router("/", &ApiController{}, "get:DispatchRoot")

	// enable cors
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
		AllowCredentials: true,
	}))

	glog.Info("start server:", port)
	beego.Run(":" + port)
}
func main() {
	err := rootCmd.Execute()
	if err != nil {
		glog.Error(err)
	}
}
