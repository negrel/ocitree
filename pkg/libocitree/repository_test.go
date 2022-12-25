package libocitree

import (
	"context"
	"os"
	"testing"

	"github.com/containers/common/libimage"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/stretchr/testify/require"
)

func requireEqualTags(t *testing.T, expected []string, actual []reference.Tag) {
	require.Len(t, actual, len(expected), "unexpected number of tags")
	for i := range expected {
		require.Equal(t, expected[i], actual[i].Tag())
	}
}

func TestRepositoryOtherHeadTags(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	ref, err := reference.RemoteRefFromString("alpine:latest")
	require.NoError(t, err)

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

	otherHeadTags := repo.OtherHeadTags()
	requireEqualTags(t, []string{"latest"}, otherHeadTags)

	// Add another tag
	testRef, err := reference.RemoteRefFromString(ref.Name().String() + ":testtag")
	require.NoError(t, err)
	err = manager.store.AddNames(repo.ID(), []string{testRef.String()})
	require.NoError(t, err, "failed to add ref to repository")

	// Tags remain unchanged until we reload the HEAD
	otherHeadTags = repo.OtherHeadTags()
	requireEqualTags(t, []string{"latest"}, otherHeadTags)

	// Reload head
	err = repo.ReloadHead()
	require.NoError(t, err)

	// Check tag is in the list now
	otherHeadTags = repo.OtherHeadTags()
	requireEqualTags(t, []string{"testtag", "latest"}, otherHeadTags)

	// Remove the latest tag
	err = manager.store.RemoveNames(repo.ID(), []string{testRef.String()})
	require.NoError(t, err)

	// Tags remain unchanged until we reload the HEAD
	otherHeadTags = repo.OtherHeadTags()
	requireEqualTags(t, []string{"testtag", "latest"}, otherHeadTags)

	// Reload head
	err = repo.ReloadHead()
	require.NoError(t, err)

	// Tags are updated now
	otherHeadTags = repo.OtherHeadTags()
	requireEqualTags(t, []string{"latest"}, otherHeadTags)
}

func TestRepositoryOtherTags(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()
	pullOptions := PullOptions{
		MaxRetries:   0,
		RetryDelay:   0,
		ReportWriter: os.Stderr,
	}

	ref, err := reference.RemoteRefFromString("alpine:latest")
	require.NoError(t, err)

	// Clone alpine repository
	err = manager.Clone(ref, CloneOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	repo, err := manager.Repository(ref.Name())
	require.NoError(t, err)

	// Ensure there is no other tags
	tags, err := repo.OtherTags()
	require.NoError(t, err)
	requireEqualTags(t, []string{}, tags)

	// Fetch another alpine image
	ref2, err := reference.RemoteRefFromString("alpine:3.15")
	require.NoError(t, err)
	manager.Fetch(ref2, FetchOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	// 3.15 tag is present now
	tags, err = repo.OtherTags()
	require.NoError(t, err)
	requireEqualTags(t, []string{"3.15"}, tags)

	_, errs := manager.rt.RemoveImages(context.Background(), []string{ref2.String()}, &libimage.RemoveImagesOptions{
		Force:   true,
		Ignore:  false,
		NoPrune: true,
	})
	require.Len(t, errs, 0)

	// No other tag anymore
	tags, err = repo.OtherTags()
	require.NoError(t, err)
	requireEqualTags(t, []string{}, tags)
}

func TestRepositoryAddTag(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// Clone alpine
	ref, err := reference.RemoteRefFromString("alpine:latest")
	require.NoError(t, err)
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

	t.Run("ValidTag", func(t *testing.T) {
		// Add a tag
		tag, err := reference.LocalTagFromString("edge")
		require.NoError(t, err)
		err = repo.AddTag(tag)
		require.NoError(t, err)

		// Check image is tagged
		localRef := reference.NewLocal(ref.Name(), tag)
		img, _, err := manager.rt.LookupImage(localRef.String(), nil)
		require.NoError(t, err)
		require.NotNil(t, img)
		require.Equal(t, repo.ID(), img.ID())

		// Check repository object is up to date
		tags := repo.OtherHeadTags()
		requireEqualTags(t, []string{"latest", "edge"}, tags)
	})

}

func TestRepositoryRemoveTag(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	// Clone alpine
	ref, err := reference.RemoteRefFromString("alpine:latest")
	require.NoError(t, err)
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

	// Remove latest tag
	err = repo.RemoveTag(reference.LatestTag)
	require.NoError(t, err)

	// Check image can't be found anymore
	_, _, err = manager.rt.LookupImage(ref.String(), nil)
	require.Error(t, err)

	// Check repository object is up to date
	require.Len(t, repo.OtherHeadTags(), 0, "repository.OtherHeadTags contains removed tag")
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
	ref, err := reference.RemoteRefFromString("alpine:latest")
	require.NoError(t, err)
	err = manager.Clone(ref, CloneOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	// Fetch another alpine image
	ref2, err := reference.RemoteRefFromString("alpine:3.15")
	require.NoError(t, err)
	manager.Fetch(ref2, FetchOptions{
		PullOptions: pullOptions,
	})
	require.NoError(t, err)

	repo, err := manager.Repository(ref.Name())
	require.NoError(t, err)

	// Check HEAD reference latest
	tags := repo.OtherHeadTags()
	requireEqualTags(t, []string{"latest"}, tags)

	// Checkout to 3.15
	err = repo.Checkout(ref2)
	require.NoError(t, err)

	// Check HEAD reference 3.15 now
	tags = repo.OtherHeadTags()
	requireEqualTags(t, []string{"3.15"}, tags)
}

func TestRepositoryCheckoutRelative(t *testing.T) {

}
