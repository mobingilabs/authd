package cmd

import (
	"path/filepath"

	"github.com/docker/libtrust"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/spf13/cobra"
)

func KeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key <filename>",
		Short: "Generate a public/private trust key.",
		Long:  `Generate a public/private trust key.`,
		Run:   key,
	}

	cmd.Flags().SortFlags = false
	return cmd
}

func key(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		debug.ErrorExit("filename not supplied", 1)
	}

	/*
		// generate private key
		issuer := &token.TokenIssuer{}
		issuer.SigningKey, err = libtrust.GenerateECP256PrivateKey()
		glog.Info("keyid: ", issuer.SigningKey.KeyID())
		glog.Info("string: ", issuer.SigningKey.String())
	*/

	// try saving file
	pkfile := args[0]
	debug.Info("pkfile:", pkfile)
	pk, err := libtrust.LoadOrCreateTrustKey(pkfile)
	if err != nil {
		debug.ErrorExit(err, 1)
	}

	debug.Info("key (private):", args[0])
	debug.Info("key (public):", filepath.Join(filepath.Dir(args[0]), "public-"+filepath.Base(args[0])))
	debug.Info("key id:", pk.KeyID())
}
