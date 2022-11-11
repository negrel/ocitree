package ocitree

import (
	"github.com/containers/storage"
	"github.com/containers/storage/types"
	"github.com/spf13/pflag"
)

var storeOptions = types.StoreOptions{}

func setupStoreOptionsFlags(flagset *pflag.FlagSet) {
	flagset.StringVarP(&storeOptions.RunRoot, "run", "R", storeOptions.RunRoot, "root of the runtime state tree")
	flagset.StringVarP(&storeOptions.GraphRoot, "graph", "g", storeOptions.GraphRoot, "root of the storage tree")
	flagset.StringVarP(&storeOptions.GraphDriverName, "storage-driver", "s", storeOptions.GraphDriverName, "storage driver to use")
}

func containersStore() (storage.Store, error) {
	var err error
	if storeOptions.GraphRoot == "" && storeOptions.RunRoot == "" &&
		storeOptions.GraphDriverName == "" && len(storeOptions.GraphDriverOptions) == 0 {
		storeOptions, err = types.DefaultStoreOptionsAutoDetectUID()
		if err != nil {
			return nil, err
		}
	}

	return storage.GetStore(storeOptions)
}

type commitOptions struct {
	message string
}

var commitOpts = commitOptions{}

func setupCommitOptionsFlags(flagset *pflag.FlagSet) {
	flagset.StringVarP(&commitOpts.message, "message", "m", "", "commit message")
}
