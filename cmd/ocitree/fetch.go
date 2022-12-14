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
	rootCmd.AddCommand(fetchCmd)
	flagset := fetchCmd.PersistentFlags()
	setupStoreOptionsFlags(flagset)
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Update each remote repository references.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a repository name must be specified")
		}
		if len(args) > 1 {
			return errors.New("too many arguments specified")
		}
		repoName, err := reference.RemoteRefFromString(args[0])
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

		err = manager.Fetch(repoName, libocitree.FetchOptions{
			PullOptions: libocitree.PullOptions{
				MaxRetries:   0,
				RetryDelay:   0,
				ReportWriter: os.Stderr,
			},
		})
		if err != nil {
			logrus.Errorf("an error occurred while fetching repository: %v", err)
			os.Exit(1)
		}

		return nil
	},
}
