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
	rootCmd.AddCommand(rebaseCmd)
	flagset := rebaseCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
	setupCommitOptionsFlags(flagset)
}

var rebaseCmd = &cobra.Command{
	Use:   "rebase",
	Short: "Reapply ocitree commit on top of the given reference.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		if len(args) > 1 {
			return errors.New("too many arguments specified")
		}
		rebaseRef, err := reference.RemoteFromString(args[0])
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

		repo, err := manager.Repository(rebaseRef)
		if err != nil {
			logrus.Errorf("repository not found: %v", err)
			os.Exit(1)
		}

		session, err := repo.RebaseSession(rebaseRef)
		if err != nil {
			logrus.Errorf("failed to rebase to reference %q: %v", rebaseRef, err)
			os.Exit(1)
		}

		err = session.Apply()
		if err != nil {
			session.Delete()
			logrus.Errorf("failed to apply rebase: %v", err)
			os.Exit(1)
		}
		session.Delete()

		return nil
	},
}
