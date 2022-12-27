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
	flagset.BoolP("interactive", "i", false, "List commit to be rebase and let user edit that list before rebasing.")
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
		rebaseRef, err := reference.RelativeFromString(args[0])
		if err != nil {
			return err
		}

		os.Exit(rebase(cmd, args, rebaseRef))
		return nil
	},
}

func rebase(cmd *cobra.Command, args []string, relRebaseRef reference.Relative) int {
	store, err := containersStore()
	if err != nil {
		logrus.Errorf("failed to create containers store: %v", err)
		return 1
	}

	manager, err := libocitree.NewManagerFromStore(store, nil)
	if err != nil {
		logrus.Errorf("failed to create repository manager: %v", err)
		return 1
	}

	rebaseRef, err := manager.ResolveRelativeReference(relRebaseRef)
	if err != nil {
		logrus.Errorf("failed to resolve relative reference: %v", err)
		return 1
	}

	repo, err := manager.Repository(rebaseRef.Name())
	if err != nil {
		logrus.Errorf("repository not found: %v", err)
		return 1
	}

	session, err := repo.RebaseSession(rebaseRef)
	if err != nil {
		logrus.Errorf("failed to start rebase session using reference %q: %v", relRebaseRef, err)
		return 1
	}

	// Interactive session
	if isInteractive, _ := cmd.Flags().GetBool("interactive"); isInteractive {
		err = session.InteractiveEdit()
		if err != nil {
			logrus.Errorf("%v", err)
			return 1
		}
	}

	err = session.Apply()
	if err != nil {
		logrus.Errorf("failed to apply rebase: %v", err)
		return 1
	}

	return 0
}
