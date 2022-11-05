package ocitree

import (
	"errors"
	"fmt"
	"os"

	"github.com/negrel/ocitree/pkg/libocitree"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
	flagset := listCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local repositories.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return errors.New("too many arguments specified")
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

		repositories, err := manager.Repositories()
		if err != nil {
			logrus.Errorf("failed to list repositories: %v", err)
			os.Exit(1)
		}

		fmt.Println("Local repositories:")
		for _, repo := range repositories {
			name, err := repo.Name()
			if err != nil {
				logrus.Errorf("failed to retrieve name of repository %q: %v", repo.ID(), err)
				continue
			}
			fmt.Println(name)
		}

		return nil
	},
}
