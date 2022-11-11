package ocitree

import (
	"errors"
	"os"

	"github.com/negrel/ocitree/pkg/libocitree"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(execCmd)
	flagset := execCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
	setupCommitOptionsFlags(flagset)
}

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Exec a command in a repository rootfs and commit changes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		if len(args) == 1 {
			return errors.New("a command must be specified")
		}
		repoName, err := reference.NameFromString(args[0])
		if err != nil {
			return err
		}
		exec := args[1:]

		store, err := containersStore()
		if err != nil {
			logrus.Errorf("failed to create containers store: %v", err)
			os.Exit(1)
		}

		manager, err := libocitree.NewManagerFromStore(store, nil)
		if err != nil {
			logrus.Errorf("failed to create repository manager: %v", err)
			os.Exit(1)
		}

		repo, err := manager.Repository(repoName)
		if err != nil {
			logrus.Errorf("repository not found: %v", err)
			os.Exit(1)
		}

		flags := cmd.Flags()
		message, _ := flags.GetString("message")

		err = repo.Exec(libocitree.ExecOptions{
			Stdin:        nil,
			Stdout:       os.Stdout,
			Stderr:       os.Stderr,
			Message:      message + "\n",
			ReportWriter: os.Stderr,
		}, exec[0], exec[1:]...)
		if err != nil {
			logrus.Errorf("failed to exec command and commit: %v", err)
			os.Exit(1)
		}

		return nil
	},
}
