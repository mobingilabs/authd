package main

import (
	goflag "flag"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/golang/glog"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/mobingilabs/authd/v1"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	version = "?"

	rootCmd = &cobra.Command{
		Use:   "authd",
		Short: "authorization and authentication service for Mobingi",
		Long:  "Authorization and authentication for Mobingi.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			goflag.Parse()
		},
		// Run: serve,
		Run: serve2,
	}

	tctx     *tokenctx
	port     string
	region   string
	bucket   string
	noverify bool
)

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().StringVar(&port, "port", "8080", "server port")
	rootCmd.PersistentFlags().StringVar(&region, "aws-region", "ap-northeast-1", "aws region to access aws resources")
	rootCmd.PersistentFlags().StringVar(&bucket, "token-bucket", "authd", "s3 bucket that contains our public/private pem files")
	rootCmd.PersistentFlags().BoolVar(&noverify, "skip-verify", noverify, "skip token verification")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func serve2(cmd *cobra.Command, args []string) {
	_, pemprv, err := downloadTokenFiles()
	if err != nil {
		err = errors.Wrap(err, "download token files failed, fatal")
		glog.Exit(err)
	}

	e := echo.New()

	// middlewares
	e.Use(middleware.CORS())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderServer, "mobingi:authd:"+version)
			return next(c)
		}
	})

	// routes
	v1.NewApiV1(e, pemprv)

	// serve
	e.Server.Addr = ":" + port
	gracehttp.Serve(e.Server)
}

/*
func serve(cmd *cobra.Command, args []string) {
	beego.BConfig.ServerName = "mobingi:authd:" + version
	beego.BConfig.RunMode = beego.DEV

	// needed for http input body in request to be available for non-get and head reqs
	beego.BConfig.CopyRequestBody = true

	pempub, pemprv, err := downloadTokenFiles()
	if err != nil {
		err = errors.Wrap(err, "download token files failed, fatal")
		glog.Exit(err)
	}

	// tctx is global at the moment
	tctx, err = NewCtx(pempub, pemprv)
	if err != nil {
		err = errors.Wrap(err, "token context create failed")
		glog.Exit(err)
	}

	beego.Router("/", &ApiController{}, "get:DispatchRoot")
	beego.Router("/version", &ApiController{}, "get:DispatchVersion")
	beego.Router("/token", &ApiController{}, "post:DispatchToken")

	// enable cors
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
		AllowCredentials: true,
	}))

	glog.Info("start server: ", port)
	beego.Run(":" + port)
}
*/

func downloadTokenFiles() (string, string, error) {
	var pempub, pemprv string
	var err error

	// fnames := []string{"token.pem", "token.pem.pub"}
	fnames := []string{"private.key", "public.key"}
	sess := session.Must(session.NewSession())
	svc := s3.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	// create dir if necessary
	tmpdir := os.TempDir() + "/jwt/rsa/"
	if !private.Exists(tmpdir) {
		err := os.MkdirAll(tmpdir, 0700)
		if err != nil {
			err = errors.Wrap(err, "mkdir failed: "+tmpdir)
			glog.Error(err)
			return pempub, pemprv, err
		}
	}

	downloader := s3manager.NewDownloaderWithClient(svc)
	for _, i := range fnames {
		fl := tmpdir + i
		f, err := os.Create(fl)
		if err != nil {
			err = errors.Wrap(err, "create file failed: "+fl)
			glog.Error(err)
			return pempub, pemprv, err
		}

		// write the contents of S3 Object to the file
		n, err := downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(i),
		})

		if err != nil {
			err = errors.Wrap(err, "s3 download failed: "+fl)
			glog.Error(err)
			return pempub, pemprv, err
		}

		glog.Infof("download s3 file: %s (%v bytes)", i, n)
	}

	pempub = tmpdir + fnames[1]
	pemprv = tmpdir + fnames[0]
	glog.Info(pempub, ", ", pemprv)
	return pempub, pemprv, err
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		glog.Error(err)
	}
}
