package libocitree

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func setupParseRebaseChoicesTest(t *testing.T, ref reference.RemoteRef) (*Manager, func(), *Repository) {
	manager, cleanup := newTestManager(t)

	// Clone alpine image
	err := manager.Clone(ref, CloneOptions{
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
			Message:      fmt.Sprintf("commit %d", i),
			ReportWriter: nil,
		}, "/bin/sh", "-c", fmt.Sprintf("echo $(date +%%s) commit %v >> /root/commits", i))
		require.NoError(t, err)
	}

	return manager, cleanup, repo
}

func TestParseRebaseChoices(t *testing.T) {
	ref, err := reference.RemoteRefFromString("alpine:3.15")
	require.NoError(t, err)

	manager, cleanup, repo := setupParseRebaseChoicesTest(t, ref)
	defer cleanup()

	rebaseRef := reference.NewRemote(ref.Name(), reference.LatestTag)
	err = manager.Fetch(rebaseRef, FetchOptions{
		PullOptions: PullOptions{
			MaxRetries:   0,
			RetryDelay:   0,
			ReportWriter: os.Stderr,
		},
	})
	require.NoError(t, err)

	t.Run("MissingCommitsAreDropped", func(t *testing.T) {
		session, err := repo.RebaseSession(rebaseRef)
		require.NoError(t, err)

		commits := session.Commits()
		err = commits.ParseChoices("")
		require.NoError(t, err)

		for i := 0; i < commits.Len(); i++ {
			require.Equal(t, DropRebaseChoice.String(), commits.Get(i).Choice.String())
		}
	})

	t.Run("Default", func(t *testing.T) {
		session, err := repo.RebaseSession(rebaseRef)
		require.NoError(t, err)

		commits := session.Commits()

		err = commits.ParseChoices(commits.String())
		require.NoError(t, err)

		for i := 0; i < commits.Len(); i++ {
			require.Equalf(t, PickRebaseChoice.String(), commits.Get(i).Choice.String(), "unexpected rebase choice for commit %d", i)
		}
	})

	t.Run("ReorderedLines", func(t *testing.T) {
		session, err := repo.RebaseSession(rebaseRef)
		require.NoError(t, err)

		commits := session.Commits()
		choices := commits.String()

		// Swap two lines
		splitted := strings.Split(choices, "\n")
		splitted[0], splitted[3] = splitted[3], splitted[0]

		// Retrieve associated commits
		commit0 := commits.Get(0)
		commit3 := commits.Get(3)

		// Parse
		err = commits.ParseChoices(strings.Join(splitted, "\n"))
		require.NoError(t, err)

		// Check
		for i := 0; i < commits.Len(); i++ {
			commit := commits.Get(i)
			if i == 0 {
				require.Equal(t, commit3.ID(), commit.ID(), "commits weren't reordered properly")
			} else if i == 3 {
				require.Equal(t, commit0.ID(), commit.ID(), "commits weren't reordered properly")
			}

			require.Equalf(t, PickRebaseChoice.String(), commit.Choice.String(), "unexpected rebase choice for commit %d", i)
		}
	})

	t.Run("ChangedChoices", func(t *testing.T) {
		session, err := repo.RebaseSession(rebaseRef)
		require.NoError(t, err)

		commits := session.Commits()
		choices := commits.String()

		// Change rebase choice of 2 random commit
		splitted := strings.Split(choices, "\n")
		line1 := rand.Intn(len(splitted) - 1)
		splitted[line1] = setRebaseCommitChoice(splitted[line1], DropRebaseChoice)
		line2 := rand.Intn(len(splitted) - 1)
		splitted[line2] = setRebaseCommitChoice(splitted[line2], DropRebaseChoice)

		// Parse
		err = commits.ParseChoices(strings.Join(splitted, "\n"))
		require.NoError(t, err)

		// Check
		for i := 0; i < commits.Len(); i++ {
			commit := commits.Get(i)
			if i == line1 {
				require.Equal(t, DropRebaseChoice.String(), commit.Choice.String(), "commit choice wasn't changed")
			} else if i == line2 {
				require.Equal(t, DropRebaseChoice.String(), commit.Choice.String(), "commit choice wasn't changed")
			} else {
				require.Equal(t, PickRebaseChoice.String(), commit.Choice.String(), "wrong commit choice")
			}
		}
	})

	t.Run("InexistentCommit", func(t *testing.T) {
		session, err := repo.RebaseSession(rebaseRef)
		require.NoError(t, err)

		commits := session.Commits()
		choices := commits.String()

		// Change one commit ID
		splitted := strings.Split(choices, "\n")
		line := rand.Intn(len(splitted) - 1)
		splitted[line] = setRebaseCommitID(splitted[line], "aaa")

		// Parse
		err = commits.ParseChoices(strings.Join(splitted, "\n"))
		require.Error(t, err)
		require.Truef(t,
			regexp.MustCompile(`^failed to parse line "[^"]+": invalid rebase commit id$`).
				MatchString(err.Error()),
			"error %q doesn't match expected format",
			err.Error(),
		)
	})

	t.Run("DuplicateRebaseCommit", func(t *testing.T) {
		session, err := repo.RebaseSession(rebaseRef)
		require.NoError(t, err)

		commits := session.Commits()
		choices := commits.String()

		// Change one commit ID
		splitted := strings.Split(choices, "\n")
		line := rand.Intn(len(splitted) - 1)
		splitted = append(splitted, splitted[line])

		err = commits.ParseChoices(strings.Join(splitted, "\n"))
		require.Error(t, err)
		require.Truef(t,
			regexp.MustCompile(`^failed to parse line "[^"]+": rebase commit line already parsed$`).
				MatchString(err.Error()),
			"error %q doesn't match expected format",
			err.Error(),
		)
	})
}

func setRebaseCommitChoice(line string, choice RebaseChoice) string {
	words := strings.Split(line, " ")
	words[0] = choice.String()

	return strings.Join(words, " ")
}

func setRebaseCommitID(line string, id string) string {
	words := strings.Split(line, " ")
	words[1] = id

	return strings.Join(words, " ")
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

	// pick commit 1
	commits.Get(0).Choice = PickRebaseChoice
	// drop commit 2
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
	require.FileExists(t, filepath.Join(mountpoint, "commit1"))
	require.NoFileExists(t, filepath.Join(mountpoint, "commit2"))

	repo.Unmount()
}
