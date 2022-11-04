package ocitree

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/docker/go-units"
	"github.com/negrel/ocitree/pkg/libocitree"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logCmd)
	flagset := logCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit logs",
	RunE: func(cobracCmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		if len(args) > 1 {
			return errors.New("too many arguments specified")
		}
		repoName, err := libocitree.ParseRepoName(args[0])
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

		repo, err := manager.RepositoryByNamedRef(repoName)
		if err != nil {
			logrus.Errorf("failed to retrieve repository %q: %v", repoName.Name(), err)
			os.Exit(1)
		}

		commits, err := repo.Commits()
		if err != nil {
			logrus.Errorf("failed to list commits of %q: %v", repoName.Name(), err)
			os.Exit(1)
		}

		fmt.Println(repoName.Name())
		for _, commit := range commits {
			fmt.Printf("commit %v (%v) %v\n", commit.ID(), units.BytesSize(float64(commit.Size())), commit.Tags())
			fmt.Printf("Date %v\n", commit.CreationDate().Format(time.RubyDate))
			if comment := commit.Comment(); comment != "" {
				fmt.Printf("	%v\n", commit.Comment())
			}
			fmt.Printf("	%v\n", commit.CreatedBy())
		}

		return nil
	},
}
