package main

import (
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
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
	rootCmd.PersistentFlags().BoolVar(&params.IsDev, "rundev", params.IsDev, "run as dev, otherwise, prod")
	rootCmd.PersistentFlags().BoolVar(&params.UseSyslog, "syslog", false, "set log output to syslog")
	rootCmd.PersistentFlags().StringArray("notify-endpoints", []string{"slack"}, "values: slack")
	params.Region = util.GetRegion()
	params.Ec2Id = util.GetEc2Id()
	rootCmd.PersistentFlags().StringVar(&params.CredProfile, "cred-profile", "sesha3", "aws credenfile profile name")
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
	/*
		if params.UseSyslog {
			logger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "sesha3")
			if err != nil {
				notify.HookPost(errors.Wrap(err, "syslog setup failed, fatal"))
				d.ErrorTraceExit(err, 1)
			}

			log.SetFlags(0)
			log.SetPrefix("[" + util.GetEc2Id() + "] ")
			log.SetOutput(logger)
		}

		metrics.MetricsType.MetricsInit()
		eps, _ := cmd.Flags().GetStringArray("notify-endpoints")
		err := notify.Notifier.Init(eps)
		if err != nil {
			d.Error(err)
		}

		d.Info("--- server start ---")
		d.Info("dns:", util.GetPublicDns()+":"+params.Port)
		d.Info("ec2:", params.Ec2Id)
		d.Info("syslog:", params.UseSyslog)
		d.Info("region:", params.Region)
		d.Info("credprof:", params.CredProfile)

		// try setting up LetsEncrypt certificates locally
		err = cert.SetupLetsEncryptCert(true)
		if err != nil {
			notify.HookPost(err)
			d.Error(err)
		} else {
			certfolder := "/etc/letsencrypt/live/" + util.Domain()
			d.Info("certificate folder:", certfolder)
		}

		startm := "--- server start ---\n"
		startm += "dns: " + util.GetPublicDns() + "\n"
		startm += "ec2: " + params.Ec2Id + "\n"
		startm += "syslog: " + fmt.Sprintf("%v", params.UseSyslog)
		notify.HookPost(startm)

		beego.BConfig.ServerName = "sesha3:1.0.0"
		beego.BConfig.RunMode = beego.PROD
		if params.IsDev {
			beego.BConfig.RunMode = beego.DEV
		}

		// needed for http input body in request to be available for non-get and head reqs
		beego.BConfig.CopyRequestBody = true

		beego.Router("/", &api.ApiController{}, "get:DispatchRoot")
		beego.Router("/scratch", &api.ApiController{}, "get:DispatchScratch")
		beego.Router("/token", &api.ApiController{}, "post:DispatchToken")
		beego.Router("/ttyurl", &api.ApiController{}, "post:DispatchTtyUrl")
		beego.Router("/exec", &api.ApiController{}, "post:DispatchExec")
		beego.Handler("/debug/vars", metrics.MetricsHandler)

		// try enable cors
		beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
			AllowAllOrigins:  true,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Content-Type"},
			ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
			AllowCredentials: true,
		}))

		beego.Run(":" + params.Port)
	*/
}

func execute() {
	err := rootCmd.Execute()
	if err != nil {
		debug.ErrorTraceExit(err, 1)
	}
}
