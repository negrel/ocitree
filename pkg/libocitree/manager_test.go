package libocitree

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

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

func TestManagerAdd(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	const repoName = "archlinux"
	addFilepath, err := filepath.Abs("../../main.go")
	require.NoError(t, err, "fail to canonicalize main.go filepath")

	// Clone archlinux repository
	err = manager.Clone(repoName)
	require.NoError(t, err)

	// mount helper function
	mount := func(repoName string) (mountpoint string, umount func() error, stat func(string) (fs.FileInfo, error)) {
		repo, err := manager.Repository(repoName)
		require.NoError(t, err)
		mountpoint, err = repo.Mount()
		require.NoError(t, err)

		return mountpoint, repo.Unmount, func(fname string) (fs.FileInfo, error) {
			dirfs := os.DirFS(mountpoint).(fs.StatFS)
			return dirfs.Stat(filepath.Base(addFilepath))
		}
	}

	_, umount, stat := mount(repoName)
	defer umount()

	// Ensure file is absent
	_, err = stat(addFilepath)
	require.Error(t, err, "file is already present in image")
	require.Contains(t, err.Error(), "no such file or directory")

	for _, test := range []struct {
		chown            string
		chmod            string
		message          string
		expectedUid      uint32
		expectedGid      uint32
		expectedFileMode fs.FileMode
	}{
		{
			chown:            "",
			chmod:            "",
			message:          "simple message",
			expectedUid:      0,
			expectedGid:      0,
			expectedFileMode: 0644,
		},
		{
			chown:            "root:root",
			chmod:            "0777",
			message:          randomCommitMessage(),
			expectedUid:      0,
			expectedGid:      0,
			expectedFileMode: 0777,
		},
		{
			chown:            "root:2000",
			chmod:            "0421",
			message:          randomCommitMessage(),
			expectedUid:      0,
			expectedGid:      2000,
			expectedFileMode: 0421,
		},
		{
			chown:            "2020:2000",
			chmod:            "0321",
			message:          randomCommitMessage(),
			expectedUid:      2020,
			expectedGid:      2000,
			expectedFileMode: 0321,
		},
	} {
		t.Run("", func(t *testing.T) {
			// Add the file
			err = manager.Add(repoName, "/", AddOptions{
				Chown:   test.chown,
				Chmod:   test.chmod,
				Message: test.message,
			}, addFilepath)
			require.NoError(t, err)

			// Mount repository
			_, umount, stat = mount(repoName)
			defer umount()

			// Ensure file is present
			fileInfo, err := stat(addFilepath)
			require.NoError(t, err, "file is not in image")

			// With the right permission
			require.Equal(t, test.expectedFileMode, fileInfo.Mode(), "added file don't have the right permissions")
			// And owner
			fileStat := fileInfo.Sys().(*syscall.Stat_t)
			require.Equal(t, test.expectedUid, fileStat.Uid)
			require.Equal(t, test.expectedGid, fileStat.Gid)

			repo, err := manager.Repository(repoName)
			require.NoError(t, err)

			// Check commit
			commits, err := repo.Commits()
			require.NoError(t, err, "failed to retrieve commits from repository")
			require.Greater(t, len(commits), 0, "commits list is empty")
			// commit message
			require.Contains(t, commits[0].Comment(), test.message + "\n", "commit message doesn't contains expected substring")


			// Check commit CreatedBy
			require.Equal(t,
				commits[0].CreatedBy(),
				fmt.Sprintf("/bin/sh -c #(ocitree) ADD --chown=\"%v\" --chmod=\"%v\" %v /", test.chown, test.chmod, addFilepath),
			)
		})
	}
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

func randomCommitMessage() string {
	return base64.StdEncoding.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
}
