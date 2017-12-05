package cmd

import (
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/docker/libtrust"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/golang/glog"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/mobingilabs/authd/pkg/token"
	"github.com/mobingilabs/authd/v1"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	port   string
	region string
	bucket string
)

func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run as an http server.",
		Long:  `Run as an http server.`,
		Run:   serve,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&port, "port", "8080", "server port")
	cmd.Flags().StringVar(&region, "aws-region", "ap-northeast-1", "aws region to access region")
	cmd.Flags().StringVar(&bucket, "token-bucket", "authd", "s3 bucket that contains our key files")
	return cmd
}

func serve(cmd *cobra.Command, args []string) {
	pempub, pemprv, err := downloadTokenFiles()
	if err != nil {
		err = errors.Wrap(err, "download token files failed, fatal")
		glog.Exit(err)
	}

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderServer, "mobingi:authd:"+version)
			return next(c)
		}
	})

	e.GET("/", func(c echo.Context) error {
		c.String(http.StatusOK, "Copyright (c) Mobingi, 2015-2017. All rights reserved.")
		return nil
	})

	e.GET("/version", func(c echo.Context) error {
		c.String(http.StatusOK, version)
		return nil
	})

	issuer := &token.TokenIssuer{
		Expiration: 24 * time.Hour,
	}
	issuer.SigningKey, err = libtrust.LoadKeyFile("./testkey")

	// routes
	v1.NewApiV1(e, &v1.ApiV1Config{
		PublicPemFile:  pempub,
		PrivatePemFile: pemprv,
		AwsRegion:      region,
		Issuer:         issuer,
	})

	// serve
	e.Server.Addr = ":" + port
	gracehttp.Serve(e.Server)
}

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
