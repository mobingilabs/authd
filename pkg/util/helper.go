package util

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ErrV returns an error with stacktrace when glog level is 2, otherwise,
// return the same error with the optional wrap string, if provided.
func ErrV(err error, s ...string) error {
	if glog.V(2) {
		if len(s) > 0 {
			// add wrap text with stacktrace
			return errors.Wrap(err, s[0])
		}

		// add stacktrace
		return errors.WithStack(err)
	}

	if len(s) > 0 {
		return fmt.Errorf("%s: %v", s[0], err)
	}

	return err
}

func GetCliStringFlag(cmd *cobra.Command, f string) string {
	return flag(cmd, f)
}

func flag(cmd *cobra.Command, f string) string {
	s := cmd.Flag(f).DefValue
	if cmd.Flag(f).Changed {
		s = cmd.Flag(f).Value.String()
	}

	return s
}
