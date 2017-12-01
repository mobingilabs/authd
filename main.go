package main

import (
	goflag "flag"
	"os"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/plugins/cors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/pkg/errors"
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

	port   string
	region string
	bucket string
)

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().StringVar(&port, "port", "8080", "server port")
	rootCmd.PersistentFlags().StringVar(&region, "aws-region", "ap-northeast-1", "aws region to access aws resources")
	rootCmd.PersistentFlags().StringVar(&bucket, "token-bucket", "authd", "s3 bucket that contains our public/private pem files")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func serve(cmd *cobra.Command, args []string) {
	goflag.Parse()

	beego.BConfig.ServerName = "mobingi:authd:1.0.0"
	beego.BConfig.RunMode = beego.DEV

	// needed for http input body in request to be available for non-get and head reqs
	beego.BConfig.CopyRequestBody = true

	err := downloadTokenFiles()
	if err != nil {
		err = errors.Wrap(err, "download token files failed, fatal")
		glog.Exit(err)
	}

	beego.Router("/", &ApiController{}, "get:DispatchRoot")

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

func downloadTokenFiles() error {
	fnames := []string{"token.pem", "token.pem.pub"}
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
			return err
		}
	}

	downloader := s3manager.NewDownloaderWithClient(svc)
	for _, i := range fnames {
		fl := tmpdir + i
		f, err := os.Create(fl)
		if err != nil {
			err = errors.Wrap(err, "create file failed: "+fl)
			glog.Error(err)
			return err
		}

		// write the contents of S3 Object to the file
		n, err := downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(i),
		})

		if err != nil {
			err = errors.Wrap(err, "s3 download failed: "+fl)
			glog.Error(err)
			return err
		}

		glog.Infof("download s3 file: %s (%v bytes)", i, n)
	}

	return nil
}
func main() {
	err := rootCmd.Execute()
	if err != nil {
		glog.Error(err)
	}
}
