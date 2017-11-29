package main

import (
	goflag "flag"
	"fmt"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/plugins/cors"
	"github.com/golang/glog"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use:   "authd",
	Short: "authorization and authentication service for Mobingi",
	Long:  "Authorization and authentication for Mobingi.",
	Run:   serve,
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help",
		Short: "help about any command",
		Long: `Help provides help for any command in the application.
Simply type '` + cmdline.Args0() + ` help [path to command]' for full details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	})

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func serve(cmd *cobra.Command, args []string) {
	goflag.Parse()

	beego.BConfig.ServerName = "mobingi:authd:1.0.0"
	beego.BConfig.RunMode = beego.PROD

	// needed for http input body in request to be available for non-get and head reqs
	beego.BConfig.CopyRequestBody = true

	beego.Router("/", &ApiController{}, "get:DispatchRoot")

	// try enable cors
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
		AllowCredentials: true,
	}))

	err := fmt.Errorf("test error")
	glog.Infof("%+v", errors.Wrap(err, "info"))
	glog.Warningf("%+v", errors.Wrap(err, "warn"))
	glog.Errorf("%+v", errors.Wrap(err, "err"))

	beego.Run(":8080")
}
func main() {
	err := rootCmd.Execute()
	if err != nil {
		glog.Error(err)
	}
}
