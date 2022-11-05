package ocitree

import (
	"errors"
	"fmt"
	"os"

	"github.com/containers/image/v5/docker/reference"
	"github.com/negrel/ocitree/pkg/libocitree"
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
		repoRef, err := libocitree.ParseRepoReference(args[0])
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

		beforeCheckout, err := manager.Repository(repoRef.Name())
		if err != nil {
			logrus.Errorf("failed to find a repository before checkout: %v", err)
			os.Exit(1)
		}

		err = manager.CheckoutByRef(repoRef)
		if err != nil {
			logrus.Errorf("failed to checkout repository %q to reference %q: %v", repoRef.Name(), repoRef.String(), err)
			os.Exit(1)
		}

		afterCheckout, err := manager.Repository(repoRef.Name())
		if err != nil {
			logrus.Errorf("failed to find a repository after checkout: %v", err)
			os.Exit(1)
		}

		beforeIDs := beforeCheckout.ID()[:16]
		if tags, err := beforeCheckout.Tags(); err == nil {
			beforeIDs = fmt.Sprintf("%q (%v)", tags, beforeIDs)
		}
		afterID := afterCheckout.ID()[:16]
		if tagged, isTagged := repoRef.(reference.Tagged); isTagged {
			afterID = fmt.Sprintf("%q (%v)", tagged.Tag(), afterID)
		}

		fmt.Printf("Previous HEAD position was %v\n", beforeIDs)
		fmt.Printf("Switched to %v\n", afterID)

		return nil
	},
}
