package libocitree

import (
	"os"
	"testing"

	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func TestCommitAdd(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteFromString("alpine")
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

	repo, err := manager.Repository(ref)
	require.NoError(t, err)

	// Add directory
	err = repo.Add("/", AddOptions{
		Chmod:        "",
		Chown:        "",
		Message:      "",
		ReportWriter: os.Stderr,
	}, ".", "https://example.com/index.html")
	require.NoError(t, err, "failed to add files")

	commits, err := repo.Commits()
	require.NoError(t, err)

	require.Equal(t, AddCommitOperation, commits[0].Operation(), "wrong commit operation")
}

func TestCommitExec(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteFromString("alpine")
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

	repo, err := manager.Repository(ref)
	require.NoError(t, err)

	// Add directory
	err = repo.Exec(ExecOptions{
		Stdin:        nil,
		Stdout:       nil,
		Stderr:       nil,
		Message:      "",
		ReportWriter: os.Stderr,
	}, "touch", "/abcdefg")
	require.NoError(t, err, "failed to exec command")

	commits, err := repo.Commits()
	require.NoError(t, err)

	require.Equal(t, ExecCommitOperation, commits[0].Operation(), "wrong commit operation")
}
