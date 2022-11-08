package libocitree

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/negrel/ocitree/pkg/reference"
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

	const repoName = "docker.io/library/alpine"
	repoHeadRef, err := reference.LocalFromString(repoName)
	require.NoError(t, err)
	remoteRef, err := reference.RemoteFromString(repoName)
	require.NoError(t, err)

	runtime := manager.runtime

	t.Run("ImageMissing", func(t *testing.T) {
		// Ensure repository doesn't exist
		imageExist, err := runtime.Exists(repoHeadRef.String())
		require.NoError(t, err)
		require.False(t, imageExist, "repository image already exist")

		// Clone reference
		reportWriter := &bytes.Buffer{}
		err = manager.Clone(remoteRef, CloneOptions{
			PullOptions: PullOptions{
				MaxRetries:   0,
				RetryDelay:   0,
				ReportWriter: reportWriter,
			},
		})
		require.NoError(t, err)
		require.Greater(t, len(reportWriter.String()), 0, "report writer is empty")

		// Ensure local repository exists now
		imageExist, err = runtime.Exists(repoHeadRef.String())
		require.NoError(t, err)
		require.True(t, imageExist, "repository image doesn't exist after clone")
	})

	t.Run("ImageHeadTagMissing", func(t *testing.T) {
		// Remove repository:HEAD reference
		runtime.RemoveImages(context.Background(), []string{repoHeadRef.String()}, &libimage.RemoveImagesOptions{
			Force:   true,
			Ignore:  false,
			NoPrune: true,
		})

		// Ensure repository:HEAD doesn't exist
		imageExist, err := runtime.Exists(repoHeadRef.String())
		require.NoError(t, err)
		require.False(t, imageExist, "repository already exist")

		// Ensure repository:latest exists
		imageExist, err = runtime.Exists(remoteRef.String())
		require.NoError(t, err)
		require.True(t, imageExist, "repository image doesn't exist")

		// Clone reference
		reportWriter := &bytes.Buffer{}
		err = manager.Clone(remoteRef, CloneOptions{
			PullOptions: PullOptions{
				MaxRetries:   0,
				RetryDelay:   0,
				ReportWriter: reportWriter,
			},
		})
		require.NoError(t, err)
		require.Len(t, reportWriter.String(), 0, "report writer is not empty")

		// Ensure local repository exists now
		imageExist, err = runtime.Exists(repoHeadRef.String())
		require.NoError(t, err)
		require.True(t, imageExist, "repository still doesn't exist")
	})

	t.Run("RepositoryExists", func(t *testing.T) {
		// Ensure local repository exists
		imageExist, err := runtime.Exists(repoHeadRef.String())
		require.NoError(t, err)
		require.True(t, imageExist, "repository doesn't exist")

		reportWriter := &bytes.Buffer{}
		err = manager.Clone(remoteRef, CloneOptions{
			PullOptions: PullOptions{
				MaxRetries:   0,
				RetryDelay:   0,
				ReportWriter: reportWriter,
			},
		})
		require.Error(t, err)
		require.Equal(t, ErrLocalRepositoryAlreadyExist, err)
		require.Len(t, reportWriter.String(), 0, "report writer is not empty")
	})
}

func TestManagerRepository(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	runtime := manager.runtime

	repoName, err := reference.NameFromString("alpine")
	require.NoError(t, err)

	t.Run("ImageMissing", func(t *testing.T) {
		imageExist, err := runtime.Exists(repoName.Name())
		require.NoError(t, err)
		require.False(t, imageExist, "image is not missing")

		// Repository image is absent
		_, err = manager.Repository(repoName)
		require.Error(t, err)
	})

	t.Run("RepositoryExist", func(t *testing.T) {
		err = manager.Clone(reference.RemoteLatestFromNamed(repoName), CloneOptions{})
		require.NoError(t, err)

		// Get repository
		repo, err := manager.Repository(repoName)
		require.NoError(t, err)
		require.Equal(t, repoName.Name(), repo.Name())

		// Compare ID of image and returned repository
		image, _, err := runtime.LookupImage(reference.LocalHeadFromNamed(repoName).String(), nil)
		require.NoError(t, err)
		require.Equal(t, image.ID(), repo.ID())
	})
}

func TestManagerRepositories(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// List of repository
	repositoriesName := []string{"docker.io/library/alpine", "docker.io/library/archlinux", "docker.io/library/ubuntu"}
	repositoriesRef := make([]reference.RemoteRepository, 3)
	for i, name := range repositoriesName {
		var err error
		repositoriesRef[i], err = reference.RemoteFromString(name)
		require.NoError(t, err)
	}

	// Clone some repositories
	for _, repo := range repositoriesRef {
		err := manager.Clone(repo, CloneOptions{})
		require.NoError(t, err)
	}

	repositories, err := manager.Repositories()
	require.NoError(t, err)
	require.Len(t, repositories, len(repositoriesRef), "length of repository list doesn't match number of cloned repositories")

	for _, repo := range repositories {
		require.Contains(t, repositoriesName, repo.Name())
	}
}

func newTestManager(t *testing.T) (manager *Manager, cleanup func()) {
	store, systemContext, workdir := newStoreAndSystemContext(t)

	manager, err := NewManagerFromStore(store, systemContext)
	require.NoError(t, err)

	cleanup = func() {
		_, _ = manager.store.Shutdown(true)
		_ = os.RemoveAll(workdir)
	}

	return manager, cleanup
}

func newStoreAndSystemContext(t *testing.T) (storage.Store, *types.SystemContext, string) {
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

	return store, systemContext, workdir
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

func randomCommitMessage() string {
	return base64.StdEncoding.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
}
