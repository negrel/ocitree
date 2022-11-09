package libocitree

import (
	"context"
	"errors"
	"fmt"

	"github.com/containers/common/libimage"
	"github.com/negrel/ocitree/pkg/reference"
)

var (
	ErrRepositoryInvalidNoName = errors.New("invalid repository, no valid name")
)

type imageStore interface {
	addNames(string, []string) error
	lookupImage(reference.LocalRepository) (*libimage.Image, error)
	listImages(filters ...string) ([]*libimage.Image, error)
}

// Repository is an object holding the history of a rootfs (OCI/Docker image).
type Repository struct {
	headRef reference.LocalRepository
	store   imageStore
	head    *libimage.Image
}

func newRepositoryFromImage(store imageStore, head *libimage.Image) (*Repository, error) {
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
		store:   store,
		head:    head,
	}, nil
}

func newRepositoryFromName(store imageStore, name reference.Named) (*Repository, error) {
	ref := reference.LocalHeadFromNamed(name)

	head, err := store.lookupImage(ref)
	if err != nil {
		return nil, err
	}

	return &Repository{
		store:   store,
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
	images, err := r.store.listImages("reference=" + r.Name() + ":*")
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

// Commits returns the commits history of this repository.
func (r *Repository) Commits() ([]Commit, error) {
	history, err := r.head.History(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve history from image: %w", err)
	}

	commits := make([]Commit, len(history))
	for i, h := range history {
		commits[i] = newCommit(h)
	}

	return commits, nil
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
		if name.Tag() == reference.HeadTag {
			return name.Name()
		}
	}

	return ""
}

// ReloadHead reloads underlying HEAD image.
func (r *Repository) ReloadHead() error {
	img, err := r.store.lookupImage(r.headRef)
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
	img, err := r.store.lookupImage(ref)
	if err != nil {
		return fmt.Errorf("local reference not found: %v", err)
	}

	// Tag head
	err = r.store.addNames(img.ID(), []string{reference.LocalHeadFromNamed(ref).String()})
	if err != nil {
		return fmt.Errorf("failed to add HEAD tag: %w", err)
	}

	// Move head
	r.head = img

	return nil
}
