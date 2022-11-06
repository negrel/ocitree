package ocitree

import (
	"errors"
	"os"

	"github.com/negrel/ocitree/pkg/libocitree"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd)
	flagset := addCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
	flagset.String("chown", "", "change owner of source files before adding them")
	flagset.String("chmod", "", "change file mode bits of source files before adding them")
	flagset.StringP("message", "m", "", "commit message")
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add files to a repository.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		repoName, err := libocitree.ParseRepoName(args[0])
		if err != nil {
			return err
		}

		if len(args) == 1 {
			return errors.New("a destination directory must be specified")
		}
		dest := args[1]

		if len(args) == 2 {
			return errors.New("at least one source file must be specified")
		}
		sources := args[2:]

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

		flags := cmd.Flags()
		chmod, _ := flags.GetString("chmod")
		chown, _ := flags.GetString("chown")
		message, _ := flags.GetString("message")

		err = manager.AddByNamedRef(repoName, dest, libocitree.AddOptions{
			Chmod:        chmod,
			Chown:        chown,
			Message:      message + "\n",
			ReportWriter: os.Stderr,
		}, sources...)
		if err != nil {
			logrus.Errorf("failed to add files to repository: %v", err)
			os.Exit(1)
		}

		return nil
	},
}
