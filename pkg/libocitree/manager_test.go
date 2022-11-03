package libocitree

import (
	"os"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func TestManagerClone(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// Repository doesn't exist
	_, err := manager.Repository("docker.io/library/alpine")
	require.Error(t, err)

	// Clone the repository using an equivalent name.
	clonedRepo, err := manager.Clone("alpine:latest")
	require.NoError(t, err)


	// Repository exists now, again using a similar name
	repo, err := manager.Repository("docker.io/alpine")
	require.NoError(t, err)
	require.Equal(t, clonedRepo.ID(), repo.ID())
}

func newTestManager(t *testing.T) (manager *Manager, cleanup func()) {
	workdir, err := os.MkdirTemp("", "testStorageRuntime")
	require.NoError(t, err)
	storeOptions := storage.StoreOptions{
		RunRoot:         workdir,
		GraphRoot:       workdir,
		GraphDriverName: "vfs",
	}

	// Make sure that the tests do not use the host's registries.conf.
	systemContext := &types.SystemContext{
		SystemRegistriesConfPath:    "testdata/registries.conf",
		SystemRegistriesConfDirPath: "/dev/null",
	}

	store, err := storage.GetStore(storeOptions)
	require.NoError(t, err)

	manager, err = NewManagerFromStore(store, systemContext)
	require.NoError(t, err)

	cleanup = func() {
		_, _ = manager.store.Shutdown(true)
		_ = os.RemoveAll(workdir)
	}

	return manager, cleanup
}

// tmpdir returns a path to a temporary directory.
func tmpdir() (string, error) {
	var tmpdir string
	defaultContainerConfig, err := config.Default()
	if err == nil {
		tmpdir, err = defaultContainerConfig.ImageCopyTmpDir()
		if err == nil {
			return tmpdir, nil
		}
	}
	return tmpdir, err
}
