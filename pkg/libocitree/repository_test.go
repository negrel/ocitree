package libocitree

import (
	"context"
	"os"
	"testing"

	"github.com/containers/common/libimage"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func TestRepositoryHeadTags(t *testing.T) {
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

	tags := repo.HeadTags()
	require.Equal(t, []string{"latest"}, tags)

	// Add another tag
	testRef, err := reference.RemoteFromString(ref.Name() + ":testtag")
	require.NoError(t, err)
	err = manager.store.AddNames(repo.ID(), []string{testRef.String()})
	require.NoError(t, err, "failed to add tag to repository")

	// Tags remain unchanged until we reload the HEAD
	tags = repo.HeadTags()
	require.Equal(t, []string{"latest"}, tags)

	// Reload head
	err = repo.ReloadHead()
	require.NoError(t, err)

	// Check tag is in the list now
	tags = repo.HeadTags()
	require.Equal(t, []string{"testtag", "latest"}, tags)

	// Remove the latest tag
	err = manager.store.RemoveNames(repo.ID(), []string{testRef.String()})
	require.NoError(t, err)

	// Tags remain unchanged until we reload the HEAD
	tags = repo.HeadTags()
	require.Equal(t, []string{"testtag", "latest"}, tags)

	// Reload head
	err = repo.ReloadHead()
	require.NoError(t, err)

	// Tags are updated now
	tags = repo.HeadTags()
	require.Equal(t, []string{"latest"}, tags)
}

func TestRepositoryOtherTags(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()
	pullOptions := PullOptions{
		MaxRetries:   0,
		RetryDelay:   0,
		ReportWriter: os.Stderr,
	}

	ref, err := reference.RemoteFromString("alpine:latest")
	require.NoError(t, err)

	// Clone alpine repository
	err = manager.Clone(ref, CloneOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	repo, err := manager.Repository(ref)
	require.NoError(t, err)

	// Ensure there is no other tags
	tags, err := repo.OtherTags()
	require.NoError(t, err)
	require.Equal(t, []string{}, tags)

	// Fetch another alpine image
	ref2, err := reference.RemoteFromString("alpine:3.15")
	require.NoError(t, err)
	manager.Fetch(ref2, FetchOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	// 3.15 tag is present now
	tags, err = repo.OtherTags()
	require.NoError(t, err)
	require.Equal(t, []string{"3.15"}, tags)

	_, errs := manager.runtime.RemoveImages(context.Background(), []string{ref2.String()}, &libimage.RemoveImagesOptions{
		Force:   true,
		Ignore:  false,
		NoPrune: true,
	})
	require.Len(t, errs, 0)

	// No other tag anymore
	tags, err = repo.OtherTags()
	require.NoError(t, err)
	require.Equal(t, []string{}, tags)
}

func TestRepositoryAddTag(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// Clone alpine
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

	repo, err := manager.Repository(ref)
	require.NoError(t, err)

	// Add a tag
	tag, err := reference.TagFromString("edge")
	require.NoError(t, err)
	err = repo.AddTag(tag)
	require.NoError(t, err)

	// Check image is tagged
	localRef := reference.LocalFromNamedTagged(ref, tag)
	img, _, err := manager.runtime.LookupImage(localRef.String(), nil)
	require.NoError(t, err)
	require.NotNil(t, img)
	require.Equal(t, repo.ID(), img.ID())

	// Check repository object is up to date
	require.Contains(t, repo.HeadTags(), tag.String(), "repository.HeadTags doesn't contain added tag")
}

func TestRepositoryRemoveTag(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// Clone alpine
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

	repo, err := manager.Repository(ref)
	require.NoError(t, err)

	// Remove latest tag
	err = repo.RemoveTag(ref)
	require.NoError(t, err)

	// Check image can't be found anymore
	_, _, err = manager.runtime.LookupImage(ref.String(), nil)
	require.Error(t, err)

	// Check repository object is up to date
	require.Equal(t, []string{}, repo.HeadTags(), "repository.HeadTags contain removed tag")
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
	tags := repo.HeadTags()
	require.Equal(t, []string{"latest"}, tags)

	// Checkout to 3.15
	err = repo.Checkout(ref2)
	require.NoError(t, err)

	// Check HEAD reference 3.15 now
	tags = repo.HeadTags()
	require.Equal(t, []string{"3.15"}, tags)
}
