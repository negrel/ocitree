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
	repo, err := manager.Repository("docker.io/library/alpine")
	require.Error(t, err)
	require.Nil(t, repo)

	// Clone the repository using an equivalent name
	err = manager.Clone("alpine:3.15")
	require.NoError(t, err)

	// Get cloned repository
	repo3_15, err := manager.Repository("alpine")
	require.NoError(t, err)

	// Repository exists now, again using a similar name
	img3_15, err := manager.lookupImage("docker.io/alpine:HEAD")
	require.NoError(t, err)
	require.Equal(t, repo3_15.ID(), img3_15.ID())

	t.Run("RepositoryAlreadyExist", func(t *testing.T) {
		// Cloning another reference to the same repo
		err := manager.Clone("alpine:3.16")
		require.NoError(t, err)

		img3_16, err := manager.lookupImage("docker.io/library/alpine:3.16")
		require.NoError(t, err)

		// Ensure id differs
		require.NotEqual(t, img3_16.ID(), repo3_15.ID())

		// Ensure HEAD hasn't moved
		repo, err := manager.Repository("alpine")
		require.NoError(t, err)
		require.NotEqual(t, img3_16.ID(), repo.ID())
		require.Equal(t, img3_15.ID(), repo.ID())
	})
}

func TestManagerListRepository(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	repos, err := manager.Repositories()
	require.NoError(t, err)
	require.Len(t, repos, 0)

	// Clone the repository
	err = manager.Clone("alpine:latest")
	require.NoError(t, err)

	// Get cloned repository
	clonedRepo, err := manager.Repository("alpine")
	require.NoError(t, err)

	repos, err = manager.Repositories()
	require.NoError(t, err)
	require.Len(t, repos, 1)

	require.Equal(t, clonedRepo.ID(), repos[0].ID())
}

func TestManagerCheckout(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// Clone the repository
	err := manager.Clone("alpine:latest")
	require.NoError(t, err)

	// Get cloned repository
	clonedRepo, err := manager.Repository("alpine")
	require.NoError(t, err)

	// Checkout to a remote reference not in local storage should fail
	err = manager.Checkout("alpine:3.15")
	require.Error(t, err)
	// Clone remote reference
	err = manager.Clone("alpine:3.15")
	require.NoError(t, err)

	// Checkout local reference
	err = manager.Checkout("alpine:3.15")
	require.NoError(t, err)

	_ = clonedRepo
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
