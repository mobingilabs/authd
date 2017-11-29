package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/plugins/cors"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "authd",
	Short: "authorization and authentication service for Mobingi",
	Long:  "Authorization and authentication for Mobingi.",
	Run:   serve,
}

func init() {
	rootCmd.Flags().SortFlags = false
	/*
		rootCmd.PersistentFlags().BoolVar(&params.IsDev, "rundev", params.IsDev, "run as dev, otherwise, prod")
		rootCmd.PersistentFlags().BoolVar(&params.UseSyslog, "syslog", false, "set log output to syslog")
		rootCmd.PersistentFlags().StringArray("notify-endpoints", []string{"slack"}, "values: slack")
		params.Region = util.GetRegion()
		params.Ec2Id = util.GetEc2Id()
		rootCmd.PersistentFlags().StringVar(&params.CredProfile, "cred-profile", "sesha3", "aws credenfile profile name")
	*/
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help",
		Short: "help about any command",
		Long: `Help provides help for any command in the application.
Simply type '` + cmdline.Args0() + ` help [path to command]' for full details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	})
}

func serve(cmd *cobra.Command, args []string) {
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

	beego.Run(":8080")
}

func execute() {
	err := rootCmd.Execute()
	if err != nil {
		debug.ErrorTraceExit(err, 1)
	}
}
