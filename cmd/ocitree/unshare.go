package ocitree

import (
	"errors"
	"os"
	"os/exec"

	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(unshareCmd)
}

var unshareCmd = &cobra.Command{
	Use:   "unshare",
	Short: "Run a command in a modified user namespace.",
	RunE:  runUnshare,
}

func runUnshare(cobraCmd *cobra.Command, args []string) error {
	if isRootless := unshare.IsRootless(); !isRootless {
		logrus.Error("please use unshare with rootless")
		os.Exit(1)
	}

	// exec the specified command, if there is one
	if len(args) < 1 {
		logrus.Debug("no cmd specified, detecting $SHELL...")

		// try to exec the shell, if one's set
		shell, shellSet := os.LookupEnv("SHELL")
		if !shellSet {
			return errors.New("no command specified and no $SHELL specified")
		}

		logrus.Debug("$SHELL detected: ", shell)
		args = []string{shell}
	}

	logrus.Debug("entering modified user namespace...")
	unshare.MaybeReexecUsingUserNamespace(false)
	logrus.Debugf("modified user namespace successfully entered, executing %v in a modified user namespace...", args)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		logrus.Debugf("failed to run command in modified user namespace: %v", err)
		os.Exit(1)
	}

	logrus.Debugf("%v successfully executed.", args)
	return nil
}

