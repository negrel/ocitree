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
	rootCmd.AddCommand(cloneCmd)
	flagset := cloneCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
}

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone a remote repository to local storage.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		if len(args) > 1 {
			return errors.New("too many arguments specified")
		}
		repoRef, err := reference.RemoteFromString(args[0])
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

		err = manager.Clone(repoRef)
		if err != nil {
			logrus.Errorf("failed to clone repository %q: %v", repoRef, err)
			os.Exit(1)
		}

		fmt.Printf("Repository %q successfully cloned.\n", repoRef.Name())

		return nil
	},
}
