package libocitree

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func setupParseRebaseChoicesTest(t *testing.T) (*Manager, func(), *Repository) {
	manager, cleanup := newTestManager(t)

	ref, err := reference.RemoteRefFromString("alpine")
	require.NoError(t, err)

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

	for i := 0; i < 10; i++ {
		err = repo.Exec(ExecOptions{
			Stdin:        nil,
			Stdout:       nil,
			Stderr:       nil,
			Message:      "",
			ReportWriter: nil,
		}, "/bin/sh", "-c", fmt.Sprintf("echo $(date +%%s) commit %v >> /root/commits", i))
		require.NoError(t, err)
	}

	return manager, cleanup, repo
}

func TestRebaseSession(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteRefFromString("alpine")
	require.NoError(t, err)

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

	// Add one commit per rebase choice
	err = repo.Exec(ExecOptions{
		Stdin:        nil,
		Stdout:       nil,
		Stderr:       nil,
		Message:      "empty commit 1",
		ReportWriter: nil,
	}, "/bin/sh", "-c", "touch /commit1")
	require.NoError(t, err)
	err = repo.Exec(ExecOptions{
		Stdin:        nil,
		Stdout:       nil,
		Stderr:       nil,
		Message:      "empty commit 2",
		ReportWriter: nil,
	}, "/bin/sh", "-c", "touch /commit2")
	require.NoError(t, err)

	// Create a rebase session
	session, err := repo.RebaseSession(ref)
	require.NoError(t, err)

	// Get rebase commits and alter choice
	commits := session.Commits()
	require.Equal(t, 2, commits.Len(), "number of commits part of rebase session")

	// Default choice is pick for every commits
	for i := 0; i < commits.Len(); i++ {
		require.Equal(t, PickRebaseChoice, commits.Get(i).Choice, "default rebase choice")
	}

	// pick commit 2
	commits.Get(0).Choice = PickRebaseChoice
	// drop commit 1
	commits.Get(1).Choice = DropRebaseChoice

	// Apply rebase session
	err = session.Apply()
	require.NoError(t, err)

	// Reload repository
	err = repo.ReloadHead()
	require.NoError(t, err)

	// Mount repository
	mountpoint, err := repo.Mount()
	require.NoError(t, err)

	// Check pick & drop
	require.FileExists(t, filepath.Join(mountpoint, "commit2"))
	require.NoFileExists(t, filepath.Join(mountpoint, "commit1"))

	repo.Unmount()
}
