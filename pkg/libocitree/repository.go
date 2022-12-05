package libocitree

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/containers/buildah"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	"github.com/negrel/ocitree/pkg/reference"
)

var (
	ErrRepositoryInvalidNoName = errors.New("invalid repository, no valid name")
)

type imageRuntime interface {
	lookupImage(reference.LocalRepository) (*libimage.Image, error)
	listImages(filters ...string) ([]*libimage.Image, error)
	repoBuilder(reference.Named, io.Writer) (*buildah.Builder, error)
	storageReference(reference.LocalRepository) types.ImageReference
	systemContext() *types.SystemContext
	diff(from, to *Commit) (io.ReadCloser, error)
}

// Repository is an object holding the history of a rootfs (OCI/Docker image).
type Repository struct {
	headRef reference.LocalRepository
	runtime imageRuntime
	head    *libimage.Image
}

func newRepositoryFromImage(store imageRuntime, head *libimage.Image) (*Repository, error) {
	names, err := head.NamedTaggedRepoTags()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve image references of repository: %w", err)
	}

	repoName := findRepoName(names)
	if repoName == "" {
		return nil, ErrRepositoryInvalidNoName
	}
	nameRef, err := reference.NameFromString(repoName)
	if err != nil {
		panic(err)
	}
	headRef := reference.LocalHeadFromNamed(nameRef)

	return &Repository{
		headRef: headRef,
		runtime: store,
		head:    head,
	}, nil
}

func newRepositoryFromName(store imageRuntime, name reference.Named) (*Repository, error) {
	ref := reference.LocalHeadFromNamed(name)

	head, err := store.lookupImage(ref)
	if err != nil {
		return nil, err
	}

	return &Repository{
		runtime: store,
		head:    head,
		headRef: ref,
	}, nil
}

// ID returns the ID of the image.
func (r *Repository) ID() string {
	return r.head.ID()
}

// Name returns the name of the repository.
func (r *Repository) Name() string {
	return r.headRef.Name()
}

// NameRef returns the underlying HEAD reference.
func (r *Repository) HeadRef() reference.LocalRepository {
	return r.headRef
}

// HeadTags returns other tags pointing to the same commit as HEAD.
func (r *Repository) HeadTags() []string {
	names := r.head.Names()
	tags := make([]string, 0, len(names))

	for _, name := range names {
		ref, err := reference.RemoteFromString(name)
		if err != nil {
			continue
		}

		if ref.Name() == r.Name() {
			tags = append(tags, ref.Tag())
		}
	}

	return tags
}

// OtherTags returns tags associated to this repository without tags associated to
// HEAD.
func (r *Repository) OtherTags() ([]string, error) {
	images, err := r.runtime.listImages("reference=" + r.Name() + ":*")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository reference: %w", err)
	}

	tags := make([]string, 0, len(images))

	for _, img := range images {
		if img.ID() == r.ID() {
			continue
		}

		imgNames := img.Names()
		for _, name := range imgNames {
			imgRef, err := reference.RemoteFromString(name)
			if err != nil {
				continue
			}

			tags = append(tags, imgRef.Tag())
		}
	}

	return tags, nil
}

// AddTag adds the given tag to HEAD.
func (r *Repository) AddTag(tag reference.Tagged) error {
	ref, err := reference.RemoteFromString(r.Name() + ":" + tag.Tag())
	if err != nil {
		return err
	}

	return r.head.Tag(ref.String())
}

// RemoveTag returns the given tag from HEAD.
func (r *Repository) RemoveTag(tag reference.Tagged) error {
	ref, err := reference.RemoteFromString(r.Name() + ":" + tag.Tag())
	if err != nil {
		return err
	}

	return r.head.Untag(ref.String())
}

// removeLocalTag removes the given tag even if it's a local one (e.g. REBASE_HEAD)
func (r *Repository) removeLocalTag(tag reference.Tagged) error {
	ref := reference.LocalFromNamedTagged(r.HeadRef(), tag)

	return r.head.Untag(ref.String())
}

// Commits returns the commits history of this repository.
// Commits are ordered from newer to older commits.
func (r *Repository) Commits() (Commits, error) {
	history, err := r.head.History(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve history from image: %w", err)
	}

	return newCommits(history), nil
}

// Mount mounts the repository and returns the mountpoint.
func (r *Repository) Mount() (string, error) {
	return r.head.Mount(context.Background(), []string{}, "")
}

// Unmount unmount the repository.
func (r *Repository) Unmount() error {
	return r.head.Unmount(true)
}

func findRepoName(names []reference.NamedTagged) string {
	for _, name := range names {
		if name.Tag() == reference.Head {
			return name.Name()
		}
	}

	return ""
}

// ReloadHead reloads underlying HEAD image.
func (r *Repository) ReloadHead() error {
	img, err := r.runtime.lookupImage(r.headRef)
	if err != nil {
		return err
	}

	r.head = img

	return nil
}

// Checkout moves repository's HEAD to the given reference tag.
func (r *Repository) Checkout(tag reference.Tagged) error {
	ref, err := reference.LocalFromString(r.Name() + ":" + tag.Tag())
	// Should never occur as tag and reposity name are valid
	if err != nil {
		panic(err)
	}

	// Get reference
	img, err := r.runtime.lookupImage(ref)
	if err != nil {
		return fmt.Errorf("local reference not found: %v", err)
	}

	// Tag head
	err = img.Tag(reference.LocalHeadFromNamed(ref).String())
	if err != nil {
		return fmt.Errorf("failed to add HEAD tag: %w", err)
	}

	// Move head
	r.head = img

	return nil
}
