package libocitree

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/containers/common/libimage"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func TestRepositoryAdd(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteFromString("alpine")
	require.NoError(t, err)
	headRef := reference.LocalHeadFromNamed(ref)

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

	t.Run("Valid", func(t *testing.T) {
		history := getImageHistory(t, manager.runtime, headRef.String())
		historySize := len(history)

		// Add directory
		commitMsg := randomCommitMessage()
		err = repo.Add("/", AddOptions{
			Chmod:        "",
			Chown:        "",
			Message:      commitMsg,
			ReportWriter: os.Stderr,
		}, ".", "https://example.com/index.html")
		require.NoError(t, err, "failed to add files")

		// Let's check image history now
		history = getImageHistory(t, manager.runtime, headRef.String())

		// Ensure commit was added to history
		require.Equal(t, historySize+1, len(history), "add commit is missing")

		// Check commit message
		// We're splitting history comment as buildah append text to comment
		require.Equal(t, commitMsg, strings.Split(history[0].Comment, "\nFROM")[0], "wrong commit message")

		// Check created by
		wd, err := os.Getwd()
		require.NoError(t, err, "failed to get working directory")
		expectedCreatedBy := fmt.Sprintf("/bin/sh -c #(ocitree) ADD --chown=\"\" --chmod=\"\" [%q %q] /",
			wd, "https://example.com/index.html")
		require.Equal(t, expectedCreatedBy, history[0].CreatedBy, "wrong CreatedBy field")

		// Check HEAD of repository is up to date
		require.Equal(t, repo.ID(), history[0].ID, "repository id and commit id differ")
	})

	t.Run("Failed", func(t *testing.T) {
		history := getImageHistory(t, manager.runtime, headRef.String())

		// Add directory
		commitMsg := randomCommitMessage()
		err = repo.Add("/", AddOptions{
			Chmod:        "",
			Chown:        "",
			Message:      commitMsg,
			ReportWriter: os.Stderr,
		}, "https://inexistent-subdomain.negrel.dev/index.html")
		require.Error(t, err, "add should fail")

		// Let's check image history now
		history2 := getImageHistory(t, manager.runtime, headRef.String())

		// Ensure no commit was added to history
		require.Equal(t, history, history2, "history was modified but add failed")
	})
}

func TestRepositoryExec(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteFromString("alpine")
	require.NoError(t, err)
	headRef := reference.LocalHeadFromNamed(ref)

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

	history := getImageHistory(t, manager.runtime, headRef.String())
	historySize := len(history)

	// Setup stdin and stdout.
	stdin := &bytes.Buffer{}
	expectedStdout := randomString(128)
	stdin.WriteString(expectedStdout)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Stdin differs from stdout and stderr
	require.NotEqual(t, stdin.Len(), stdout.Len(), "stdout buffer isn't empty before exec")
	require.NotEqual(t, stdin.Len(), stderr.Len(), "stderr buffer isn't empty befor exec")
	require.Greater(t, stdin.Len(), 0, "stdin is empty")

	commitMsg := randomCommitMessage()
	expectedStderr := randomString(64)
	cmd := fmt.Sprintf("cat; printf %q >&2", expectedStderr)
	err = repo.Exec(ExecOptions{
		Stdin:        stdin,
		Stdout:       stdout,
		Stderr:       stderr,
		Message:      commitMsg,
		ReportWriter: os.Stderr,
	}, "/bin/sh", "-c", cmd)
	require.NoError(t, err)

	// Now stdin, stdout and stderr are equals
	require.Equal(t, stdin.Len(), 0, "stdin isn't empty")
	require.Equal(t, expectedStdout, stdout.String(), "stdout doesn't contains expected string")
	require.Equal(t, expectedStderr, stderr.String(), "stderr doesn't contains expected string")

	// Let's check image history now
	history = getImageHistory(t, manager.runtime, headRef.String())

	// Ensure commit was added to history
	require.Equal(t, historySize+1, len(history), "add commit is missing")

	// Check commit message
	// We're splitting history comment as buildah append text to comment
	require.Equal(t, commitMsg, strings.Split(history[0].Comment, "\nFROM")[0], "wrong commit message")

	// Check created by
	expectedCreatedBy := fmt.Sprintf(`/bin/sh -c #(ocitree) EXEC [%q %q %q]`,
		"/bin/sh", "-c", cmd)
	require.Equal(t, expectedCreatedBy, history[0].CreatedBy, "wrong CreatedBy field")

	// Check HEAD of repository is up to date
	require.Equal(t, repo.ID(), history[0].ID, "repository id and commit id differ")
}

func getImageHistory(t *testing.T, runtime *libimage.Runtime, ref string) []libimage.ImageHistory {
	img, _, err := runtime.LookupImage(ref, nil)
	require.NoError(t, err)

	history, err := img.History(context.Background())
	require.NoError(t, err)

	return history
}
