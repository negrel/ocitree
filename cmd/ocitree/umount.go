package ocitree

import (
	"errors"
	"os"

	"github.com/negrel/ocitree/pkg/libocitree"
	refcomp "github.com/negrel/ocitree/pkg/reference/components"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(umountCmd)
	flagset := umountCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
}

var umountCmd = &cobra.Command{
	Use:   "umount",
	Short: "Unmount a repository.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		if len(args) > 1 {
			return errors.New("too many arguments specified")
		}
		repoName, err := refcomp.NameFromString(args[0])
		if err != nil {
			return err
		}

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
			logrus.Errorf("failed to retrieve repository: %v", err)
			os.Exit(1)
		}

		err = repo.Unmount()
		if err != nil {
			logrus.Errorf("failed to unmount repository: %v", err)
			os.Exit(1)
		}

		return nil
	},
}
