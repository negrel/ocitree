package ocitree

import (
	"errors"
	"fmt"
	"os"

	"github.com/negrel/ocitree/pkg/libocitree"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mountCmd)
	flagset := mountCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
}

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a repository and print mountpoint.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		if len(args) > 1 {
			return errors.New("too many arguments specified")
		}
		repoName, err := reference.NameFromString(args[0])
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

		mountpoint, err := repo.Mount()
		if err != nil {
			logrus.Errorf("failed to mount repository: %v", err)
			os.Exit(1)
		}
		fmt.Println(mountpoint)

		return nil
	},
}
