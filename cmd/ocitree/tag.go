package ocitree

import (
	"errors"
	"fmt"
	"os"

	"github.com/negrel/ocitree/pkg/libocitree"
	refcomp "github.com/negrel/ocitree/pkg/reference/components"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(tagCmd)
	flagset := tagCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)

	flagset.BoolP("delete", "d", false, "delete tags instead of adding them")
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Add a tag to HEAD of repository.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		repoName, err := refcomp.NameFromString(args[0])
		if err != nil {
			return err
		}
		tags := make([]refcomp.Tag, len(args)-1)
		for i, tag := range args[1:] {
			tags[i], err = refcomp.TagFromString(tag)
			if err != nil {
				return fmt.Errorf("tag %q invalid: %v", tag, err)
			}
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
			logrus.Errorf("failed to retrieve repository %q: %v", repoName.Name(), err)
			os.Exit(1)
		}

		action := repo.AddTag
		actionStr := "add"
		if deleteInsteadOfAdd, _ := cmd.Flags().GetBool("delete"); deleteInsteadOfAdd {
			action = repo.RemoveTag
			actionStr = "remove"
		}

		exitCode := 0
		for _, tag := range tags {
			err = action(tag)
			if err != nil {
				logrus.Errorf("failed to %v tag %q: %v", actionStr, tag, err)
				exitCode++
			}
		}

		os.Exit(exitCode)

		return nil
	},
}
