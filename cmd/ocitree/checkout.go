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
	rootCmd.AddCommand(checkoutCmd)
	flagset := checkoutCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
}

var checkoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Moves HEAD to another reference.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository reference must be specified")
		}
		if len(args) > 1 {
			return errors.New("too many arguments specified")
		}

		repoRef, err := reference.LocalFromString(args[0])
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

		repo, err := manager.Repository(repoRef)
		if err != nil {
			logrus.Errorf("failed to find a repository: %v", err)
			os.Exit(1)
		}
		beforeIDs := repo.ID()[:16]
		if tags := repo.HeadTags(); len(tags) > 0 {
			beforeIDs = fmt.Sprintf("%q (%v)", tags, beforeIDs)
		}

		err = repo.Checkout(repoRef)
		if err != nil {
			logrus.Errorf("failed to checkout repository %q to tag %q: %v", repoRef.Name(), repoRef.Tag(), err)
			os.Exit(1)
		}

		afterID := fmt.Sprintf("%q (%v)", repoRef.Tag(), repo.ID()[:16])
		fmt.Printf("Previous HEAD position was %v\n", beforeIDs)
		fmt.Printf("Switched to %v\n", afterID)

		return nil
	},
}
