package libocitree

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/containers/buildah"
	"github.com/containers/common/libimage"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"github.com/negrel/ocitree/pkg/reference"
	refcomp "github.com/negrel/ocitree/pkg/reference/components"
	"github.com/sirupsen/logrus"
)

var (
	ErrRepositoryInvalidNoName  = errors.New("invalid repository, no valid name")
	ErrImageNotPartOfRepository = errors.New("image is not part of repository")
)

type imageRuntime interface {
	lookupImage(reference.Reference) (*libimage.Image, error)
	listImages(filters ...string) ([]*libimage.Image, error)
	repoBuilder(reference.Named, io.Writer) (*buildah.Builder, error)
	storageReference(reference.Reference) types.ImageReference
	systemContext() *types.SystemContext
	ResolveRelativeReference(reference.Relative) (reference.Reference, error)
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
	name, err := refcomp.NameFromString(repoName)
	if err != nil {
		panic(err)
	}
	headRef := reference.LocalHeadFromNamed(name)

	return &Repository{
		headRef: headRef,
		runtime: store,
		head:    head,
	}, nil
}

func newRepositoryFromName(store imageRuntime, name refcomp.Named) (*Repository, error) {
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

// OtherHeadRefs returns other reference to HEAD.
func (r *Repository) OtherHeadRefs() []reference.Reference {
	names := r.head.Names()
	refs := make([]reference.Reference, 0, len(names))

	for _, name := range names {
		ref, err := reference.RemoteFromString(name)
		if err != nil {
			continue
		}

		if ref.Name() == r.Name() {
			refs = append(refs, ref)
		}
	}

	return refs
}

// OtherRefs returns tags associated to this repository but not pointing to HEAD.
func (r *Repository) OtherRefs() ([]reference.Reference, error) {
	images, err := r.runtime.listImages("reference=" + r.Name() + ":*")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository reference: %w", err)
	}

	refs := make([]reference.Reference, 0, len(images))

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

			refs = append(refs, imgRef)
		}
	}

	return refs, nil
}

// AddTag adds the given tag to HEAD.
func (r *Repository) AddTag(tag refcomp.Tag) error {
	ref, err := reference.RemoteFromString(r.Name() + ":" + tag.Tag())
	if err != nil {
		return err
	}

	return r.head.Tag(ref.String())
}

// RemoveTag returns the given tag from HEAD.
func (r *Repository) RemoveTag(tag refcomp.Tag) error {
	ref, err := reference.RemoteFromString(r.Name() + ":" + tag.Tag())
	if err != nil {
		return err
	}

	return r.head.Untag(ref.String())
}

// removeLocalTag removes the given tag even if it's a local one (e.g. REBASE_HEAD)
func (r *Repository) removeLocalTag(tag refcomp.Tag) error {
	ref := reference.LocalFromNamedTagged(r.HeadRef(), tag)

	return r.head.Untag(ref.AbsoluteReference())
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

func findRepoName(names []dockerref.NamedTagged) string {
	for _, name := range names {
		if name.Tag() == refcomp.Head {
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

// Checkout to commit with the given Identifier.
func (r *Repository) Checkout(id reference.Reference) error {
	img, err := r.runtime.lookupImage(id)
	if err != nil {
		return fmt.Errorf("failed to lookup checkout reference: %w", err)
	}

	names := img.Names()
	names = append(names, img.NamesHistory()...)

	// Ensure image names is same as repository name.
	for _, name := range names {
		ref, err := dockerref.Parse(name)
		if err != nil {
			logrus.Debugf("skipping %v because %v", ref, err)
			continue
		}
		if named, isNamed := ref.(dockerref.Named); isNamed {
			if named.Name() == r.Name() {
				return nil
			}
		} else {
			logrus.Debugf("skipping %v because reference is not named")
		}
	}

	return ErrImageNotPartOfRepository
}

// checkout contains shared code between checkout methods.
// This method shouldn't be called directly, use public methods instead.
func (r *Repository) checkout(ref reference.Reference) error {
	// Get reference
	img, err := r.runtime.lookupImage(ref)
	if err != nil {
		return fmt.Errorf("local reference not found: %v", err)
	}

	// Tag head
	err = img.Tag(reference.LocalHeadFromNamed(r.HeadRef()).AbsoluteReference())
	if err != nil {
		return fmt.Errorf("failed to add HEAD tag: %w", err)
	}

	// Move head
	r.head = img

	return nil
}
