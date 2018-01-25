package cmd

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/golang/glog"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/oath/api/v1"
	"github.com/mobingilabs/oath/pkg/constants"
	"github.com/mobingilabs/oath/pkg/params"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
)

func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run as an http server.",
		Long:  `Run as an http server.`,
		Run:   serve,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&params.Port, "port", "8080", "server port")
	cmd.Flags().BoolVar(&params.RunK8s, "run-k8s", true, "run inside mochi cluster")
	cmd.Flags().StringVar(&params.Region, "aws-region", "ap-northeast-1", "aws region to access resources")
	cmd.Flags().StringVar(&params.Bucket, "token-bucket", "oath-store", "s3 bucket that contains our key files")
	return cmd
}

func serve(cmd *cobra.Command, args []string) {
	var pempub, pemprv string
	var err error

	if !params.RunK8s {
		pempub, pemprv, err = downloadTokenFiles()
		if err != nil {
			err = errors.Wrap(err, "download token files failed, fatal")
			glog.Exit(err)
		}
	} else {
		// provided from mochi secrets
		glog.V(1).Infof("secrets location: %v", constants.SECRETS)
		pempub = filepath.Join(constants.SECRETS, "public.key")
		pemprv = filepath.Join(constants.SECRETS, "private.key")
	}

	e := echo.New()

	// time in, should be the first middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cid := uuid.NewV4().String()
			c.Set("contextid", cid)
			c.Set("starttime", time.Now())

			// Helper func to print the elapsed time since this middleware. Good to call at end of
			// request handlers, right before/after replying to caller.
			c.Set("fnelapsed", func(ctx echo.Context) {
				start := ctx.Get("starttime").(time.Time)
				glog.Infof("<-- %v, delta: %v", ctx.Get("contextid"), time.Now().Sub(start))
			})

			glog.Infof("--> %v", cid)
			return next(c)
		}
	})

	e.Use(middleware.CORS())

	// some information about request
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			glog.Infof("remoteaddr: %v", c.Request().RemoteAddr)
			return next(c)
		}
	})

	// add server name in response
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderServer, "mobingi:oath:"+version)
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

	// routes
	v1.NewApiV1(e, &v1.ApiV1Config{
		PublicPemFile:  pempub,
		PrivatePemFile: pemprv,
		AwsRegion:      params.Region,
	})

	e.GET("/testalm", func(c echo.Context) error {
		start := time.Now()
		ep := "http://alm-apiv3-internal.default.svc.cluster.local/v3/access_token"
		resp, err := http.Get(ep)
		if err != nil {
			glog.Errorf("get failed: %v", err)
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			glog.Errorf("readall failed: %v", err)
			return err
		}

		defer resp.Body.Close()
		glog.Infof("body: %v", string(body))
		glog.Infof("delta: %v", time.Now().Sub(start))
		return c.NoContent(http.StatusOK)
	})

	// serve
	glog.Infof("serving on :%v", params.Port)
	e.Server.Addr = ":" + params.Port
	gracehttp.Serve(e.Server)
}

func downloadTokenFiles() (string, string, error) {
	var pempub, pemprv string
	var err error

	fnames := []string{"private.key", "public.key"}
	sess := session.Must(session.NewSession())
	svc := s3.New(sess, &aws.Config{
		Region: aws.String(params.Region),
	})

	// create dir if necessary
	if !private.Exists(constants.DATADIR) {
		err := os.MkdirAll(constants.DATADIR, 0700)
		if err != nil {
			glog.Errorf("mkdirall failed: %v", err)
			return pempub, pemprv, err
		}
	}

	downloader := s3manager.NewDownloaderWithClient(svc)
	for _, i := range fnames {
		fl := filepath.Join(constants.DATADIR, i)
		f, err := os.Create(fl)
		if err != nil {
			glog.Errorf("create file failed: %v", err)
			return pempub, pemprv, err
		}

		n, err := downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(params.Bucket),
			Key:    aws.String(i),
		})

		if err != nil {
			glog.Errorf("s3 download failed: %v", err)
			return pempub, pemprv, err
		}

		glog.Infof("download s3 file: %s (%v bytes)", fl, n)
	}

	pempub = filepath.Join(constants.DATADIR, fnames[1])
	pemprv = filepath.Join(constants.DATADIR, fnames[0])
	glog.Infof("pempub: %v, pemprv: %v", pempub, pemprv)

	return pempub, pemprv, err
}
