package ocitree

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/negrel/ocitree/pkg/libocitree"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cloneCmd)
	flagset := cloneCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
	flagset.BoolP("idempotent", "i", false, "silence error if repository with already exists")
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
		idempotent, _ := cmd.Flags().GetBool("idempotent")

		repoRef, err := reference.RemoteRefFromString(args[0])
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

		err = manager.Clone(repoRef, libocitree.CloneOptions{
			PullOptions: libocitree.PullOptions{
				MaxRetries:   0,
				RetryDelay:   0,
				ReportWriter: os.Stderr,
			},
		})
		// Repository already exist, ensure reference point to HEAD
		if idempotent && err == libocitree.ErrLocalRepositoryAlreadyExist {
			repo, err := manager.Repository(repoRef.Name())
			if err != nil {
				logrus.Errorf("failed to retrieve local repository %q", repoRef)
				os.Exit(1)
			}

			// ID reference
			if strings.HasPrefix(repoRef.IdOrTag(), reference.IdPrefix) {
				if repo.ID() == repoRef.IdOrTag()[len(reference.IdPrefix):] {
					goto repoCloned
				} else {
					err = fmt.Errorf("HEAD of repository point to another commit: %v", err)
				}
			} else {
				// Tag reference
				otherTags := repo.OtherHeadTags()
				for _, t := range otherTags {
					if t.Tag() == repoRef.IdOrTag()[len(reference.TagPrefix):] {
						goto repoCloned
					}
				}
			}
		}
		if err != nil {
			logrus.Errorf("failed to clone repository %q: %v", repoRef, err)
			os.Exit(1)
		}

	repoCloned:
		fmt.Printf("Repository %q successfully cloned.\n", repoRef.Name())

		return nil
	},
}
