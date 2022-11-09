package libocitree

import (
	"os"
	"testing"

	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func TestRepositoryTags(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteFromString("alpine:latest")
	require.NoError(t, err)

	err = manager.Clone(ref, CloneOptions{
		PullOptions: PullOptions{
			MaxRetries:   0,
			RetryDelay:   0,
			ReportWriter: os.Stderr,
		},
	})
	require.NoError(t, err)

	repo, err := manager.Repository(reference.NameFromNamed(ref))
	require.NoError(t, err)

	tags, err := repo.Tags()
	require.NoError(t, err)

	require.Equal(t, []string{"latest"}, tags)

	// Add another tag
	testRef, err := reference.RemoteFromString(ref.Name() + ":testtag")
	require.NoError(t, err)
	err = manager.store.AddNames(repo.ID(), []string{testRef.String()})
	require.NoError(t, err, "failed to add tag to repository")

	// Tags remain unchanged until we reload the HEAD
	tags, err = repo.Tags()
	require.NoError(t, err)
	require.Equal(t, []string{"latest"}, tags)

	// Reload head
	err = repo.ReloadHead()
	require.NoError(t, err)

	// Check tag is in the list now
	tags, err = repo.Tags()
	require.NoError(t, err)
	require.Equal(t, []string{"testtag", "latest"}, tags)

	// Remove the latest tag
	err = manager.store.RemoveNames(repo.ID(), []string{testRef.String()})
	require.NoError(t, err)

	// Tags remain unchanged until we reload the HEAD
	tags, err = repo.Tags()
	require.NoError(t, err)
	require.Equal(t, []string{"testtag", "latest"}, tags)

	// Reload head
	err = repo.ReloadHead()
	require.NoError(t, err)

	// Tags are updated now
	tags, err = repo.Tags()
	require.NoError(t, err)
	require.Equal(t, []string{"latest"}, tags)
}

func TestRepositoryCheckout(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()
	pullOptions := PullOptions{
		MaxRetries:   0,
		RetryDelay:   0,
		ReportWriter: os.Stderr,
	}

	// Clone alpine
	ref, err := reference.RemoteFromString("alpine:latest")
	require.NoError(t, err)
	err = manager.Clone(ref, CloneOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	// Fetch another alpine image
	ref2, err := reference.RemoteFromString("alpine:3.15")
	require.NoError(t, err)
	manager.Fetch(ref2, FetchOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	repo, err := manager.Repository(ref)
	require.NoError(t, err)

	// Check HEAD reference latest
	tags, err := repo.Tags()
	require.NoError(t, err)
	require.Equal(t, []string{"latest"}, tags)

	// Checkout to 3.15
	err = repo.Checkout(ref2)
	require.NoError(t, err)

	// Check HEAD reference 3.15 now
	tags, err = repo.Tags()
	require.NoError(t, err)
	require.Equal(t, []string{"3.15"}, tags)
}
