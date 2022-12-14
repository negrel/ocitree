package libocitree

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/unshare"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	if reexec.Init() {
		return
	}

	unshare.MaybeReexecUsingUserNamespace(false)

	os.Exit(m.Run())
}

func TestManagerClone(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	const repoName = "docker.io/library/alpine"
	repoHeadRef, err := reference.LocalRefFromString(repoName)
	require.NoError(t, err)
	remoteRef, err := reference.RemoteRefFromString(repoName)
	require.NoError(t, err)

	runtime := manager.rt

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

	runtime := manager.rt

	repoName, err := reference.NameFromString("alpine")
	require.NoError(t, err)

	t.Run("ImageMissing", func(t *testing.T) {
		imageExist, err := runtime.Exists(repoName.String())
		require.NoError(t, err)
		require.False(t, imageExist, "image is not missing")

		// Repository image is absent
		_, err = manager.Repository(repoName)
		require.Error(t, err)
	})

	t.Run("RepositoryExist", func(t *testing.T) {
		err = manager.Clone(reference.RemoteFromName(repoName), CloneOptions{})
		require.NoError(t, err)

		// Get repository
		repo, err := manager.Repository(repoName)
		require.NoError(t, err)
		require.Equal(t, repoName.String(), repo.Name().String())

		// Compare ID of image and returned repository
		image, _, err := runtime.LookupImage(reference.LocalFromName(repoName).String(), nil)
		require.NoError(t, err)
		require.Equal(t, image.ID(), repo.ID())
	})
}

func TestManagerRepositories(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// List of repository
	repositoriesName := []string{"docker.io/library/alpine", "docker.io/library/archlinux", "docker.io/library/ubuntu"}
	repositoriesRef := make([]reference.RemoteRef, 3)
	for i, name := range repositoriesName {
		var err error
		repositoriesRef[i], err = reference.RemoteRefFromString(name)
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
		require.Contains(t, repositoriesName, repo.Name().String())
	}
}

func TestManagerFetch(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	pullOptions := PullOptions{
		MaxRetries:   0,
		RetryDelay:   0,
		ReportWriter: os.Stderr,
	}

	ref, err := reference.RemoteRefFromString("alpine:3.15")
	require.NoError(t, err)
	headRef := reference.LocalFromName(ref.Name())

	err = manager.Clone(ref, CloneOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	img, _, err := manager.rt.LookupImage(ref.String(), nil)
	require.NoError(t, err)

	// Add a latest tag
	latestRef := reference.RemoteFromName(ref.Name())
	err = img.Tag(latestRef.String())
	require.NoError(t, err)
	require.Equal(t, []string{headRef.String(), ref.String(), latestRef.String()}, img.Names())

	// Fetch all HEAD tags + the given one (e.g 3.15, 3.14 and latest)
	ref2, err := reference.RemoteRefFromString("alpine:3.14")
	require.NoError(t, err)
	err = manager.Fetch(ref2, FetchOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	// We should have 3 images now, 3.15, 3.14 and latest
	// let's test 3.15
	img, _, err = manager.rt.LookupImage(ref.String(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{reference.LocalFromName(ref.Name()).String(), ref.String()}, img.Names())

	// latest now
	img, _, err = manager.rt.LookupImage(latestRef.String(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{latestRef.String()}, img.Names())

	// And 3.14
	img, _, err = manager.rt.LookupImage(ref2.String(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{ref2.String()}, img.Names())
}

func TestManagerResolveRelativeReference(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteRefFromString("alpine")
	require.NoError(t, err)
	headRef := reference.LocalFromName(ref.Name())

	// Clone alpine image
	err = manager.Clone(ref, CloneOptions{
		PullOptions: PullOptions{
			MaxRetries:   0,
			RetryDelay:   0,
			ReportWriter: os.Stderr,
		},
	})
	require.NoError(t, err)

	repo, err := manager.Repository(ref.Name())
	require.NoError(t, err)

	// Create one commit
	err = repo.Exec(ExecOptions{
		Stdin:        nil,
		Stdout:       nil,
		Stderr:       nil,
		Message:      randomCommitMessage(),
		ReportWriter: nil,
	}, "/bin/true")
	require.NoError(t, err)

	t.Run("ExistingReference", func(t *testing.T) {
		t.Run("Offset0", func(t *testing.T) {
			// No offset
			relRef := reference.RelativeFromReferenceAndOffset(headRef, 0)
			absRef, err := manager.ResolveRelativeReference(relRef)
			require.NoError(t, err)

			commits, err := repo.Commits()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(commits), 1)

			require.Equal(t, relRef.Base().Name().String()+reference.IdPrefix+commits[0].ID(), absRef.String())
		})
		t.Run("Offset1/HeadBase", func(t *testing.T) {
			relRef := reference.RelativeFromReferenceAndOffset(headRef, 1)
			absRef, err := manager.ResolveRelativeReference(relRef)
			require.NoError(t, err)

			commits, err := repo.Commits()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(commits), 2)

			require.Equal(t, relRef.Base().Name().String()+reference.IdPrefix+commits[1].ID(), absRef.String())
		})
	})
	t.Run("MissingReference", func(t *testing.T) {
		t.Run("Offset1/LatestBase", func(t *testing.T) {
			relRef := reference.RelativeFromReferenceAndOffset(ref, 1)
			absRef, err := manager.ResolveRelativeReference(relRef)
			require.Error(t, ErrCommitHasNoImageAssociated, err)
			require.Nil(t, absRef)
		})
	})
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

func randomString(length uint) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	result := strings.Builder{}
	result.Grow(int(length))

	for i := uint(0); i < length; i++ {
		chari := rand.Intn(len(charset))
		result.WriteByte(charset[chari])
	}

	return result.String()
}

func randomCommitMessage() string {
	return randomString(128)
}
