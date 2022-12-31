package ocitree

import (
	"os"
	"strings"

	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	pflag := rootCmd.PersistentFlags()
	pflag.VarP(LogrusLevel{}, "level", "l", `log level, one of "panic", "fatal", "error", "warn", "info", "debug", "trace"`)
}

var rootCmd = &cobra.Command{
	Use:     os.Args[0],
	Version: "v0.0.1",
}

func Execute() {
	if reexec.Init() {
		return
	}

	if isRootless := unshare.IsRootless(); isRootless {
		logrus.Debug("entering modified user namespace...")
		unshare.MaybeReexecUsingUserNamespace(false)
		logrus.Debugf("modified user namespace successfully entered.")
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type LogrusLevel struct{}

func (ll LogrusLevel) Set(v string) error {
	lvl, err := logrus.ParseLevel(strings.ToLower(v))
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)

	return nil
}

func (ll LogrusLevel) String() string {
	return logrus.GetLevel().String()
}

func (ll LogrusLevel) Type() string {
	return "log_level"
}
