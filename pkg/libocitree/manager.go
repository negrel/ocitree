package libocitree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
)

var (
	ErrLocalRepositoryAlreadyExist = errors.New("local repository with the same name already exist")
)

// Manager defines a repositories manager.
type Manager struct {
	store   storage.Store
	runtime *libimage.Runtime
}

// NewManagerFromStore returns a new Manager using the given store.
// An error is returned if libimage.Runtime can't be created using the given
// store and system context.
// Call Destroy() once you're done with the manager.
func NewManagerFromStore(store storage.Store, sysctx *types.SystemContext) (*Manager, error) {
	rt, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{
		SystemContext: sysctx,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create manager runtime: %w", err)
	}

	return &Manager{
		store:   store,
		runtime: rt,
	}, nil
}

// Repository returns the repository associated with the given name.
func (m *Manager) Repository(name string) (*Repository, error) {
	named, err := ParseRepoName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository name: %w", err)
	}

	return m.RepositoryByNamedRef(named)
}

// RepositoryByNamedRef returns the repository associated with the given
// reference name.
func (m *Manager) RepositoryByNamedRef(named reference.Named) (*Repository, error) {
	if err := validRepoName(named); err != nil {
		return nil, err
	}

	named, err := reference.WithTag(named, HeadTag)
	if err != nil {
		return nil, err
	}

	img, err := m.lookupImage(named.String())
	if err != nil {
		return nil, err
	}

	return newRepository(img), nil
}

func (m *Manager) imageExist(name string) (bool, error) {
	images, err := m.runtime.ListImages(context.Background(), []string{}, &libimage.ListImagesOptions{
		Filters: []string{"reference=" + name},
	})
	if err != nil {
		return false, err
	}

	return len(images) > 0, err
}

// lookupImage returns the image associated to the given ref.
// This function expect a fully qualified reference and will use default values
// ("latest" for tag, "docker.io" for registry) if not.
func (m *Manager) lookupImage(ref string) (*libimage.Image, error) {
	img, _, err := m.runtime.LookupImage(ref, &libimage.LookupImageOptions{
		Architecture:   runtime.GOARCH,
		OS:             runtime.GOOS,
		Variant:        "",
		PlatformPolicy: libimage.PlatformPolicyDefault,
		ManifestList:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup image: %w", err)
	}

	return img, nil
}

// Repositories returns the list of repositories
func (m *Manager) Repositories() ([]*Repository, error) {
	images, err := m.runtime.ListImages(context.Background(), nil, &libimage.ListImagesOptions{
		Filters: []string{"reference=*:HEAD"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	result := make([]*Repository, len(images))
	for i, image := range images {
		result[i] = newRepository(image)
	}

	return result, nil
}

// Clone clones remote repository with the given name to local storage.
func (m *Manager) Clone(name string) error {
	named, err := ParseRemoteRepoReference(name)
	if err != nil {
		return fmt.Errorf("failed to parse remote repository reference: %w", err)
	}

	return m.CloneByNamedRef(named)
}

// CloneByNamedRef clones remote repository with the given remote repository reference to local storage.
func (m *Manager) CloneByNamedRef(named reference.Named) error {
	// Ensure local repository doesn't exist
	alreadyExist := false
	if repo, _ := m.Repository(named.Name()); repo != nil {
		alreadyExist = true
	}

	err := m.pullRef(named)
	if err != nil {
		return err
	}

	img, err := m.lookupImage(named.String())
	if err != nil {
		return fmt.Errorf("failed to retrieve repository after pulling image %q: %w", named.Name(), err)
	}

	if !alreadyExist {
		err = m.store.AddNames(img.ID(), []string{named.Name() + ":" + HeadTag})
		if err != nil {
			return fmt.Errorf("failed to create repository from image %q: %v", named.String(), err)
		}
	}

	return nil
}

func (m *Manager) pullRef(repoRef reference.Reference) error {
	maxRetries := uint(3)
	retryDelay := time.Second
	_, err := m.runtime.Pull(context.Background(), repoRef.String(), config.PullPolicyNewer, &libimage.PullOptions{
		CopyOptions: libimage.CopyOptions{
			SystemContext:                    m.runtime.SystemContext(),
			SourceLookupReferenceFunc:        nil,
			DestinationLookupReferenceFunc:   nil,
			CompressionFormat:                nil,
			CompressionLevel:                 nil,
			AuthFilePath:                     "",
			BlobInfoCacheDirPath:             "",
			CertDirPath:                      "",
			DirForceCompress:                 false,
			InsecureSkipTLSVerify:            0,
			MaxRetries:                       &maxRetries,
			RetryDelay:                       &retryDelay,
			ManifestMIMEType:                 "",
			OciAcceptUncompressedLayers:      true,
			OciEncryptConfig:                 nil,
			OciEncryptLayers:                 nil,
			OciDecryptConfig:                 nil,
			Progress:                         nil,
			PolicyAllowStorage:               false,
			SignaturePolicyPath:              "",
			SignBy:                           "",
			SignPassphrase:                   "",
			SignBySigstorePrivateKeyFile:     "",
			SignSigstorePrivateKeyPassphrase: nil,
			RemoveSignatures:                 false,
			Writer:                           os.Stderr,
			Architecture:                     "",
			OS:                               "",
			Variant:                          "",
			Username:                         "",
			Password:                         "",
			Credentials:                      "",
			IdentityToken:                    "",
		},
		AllTags: false,
	})
	return err
}

// Checkout moves repository's HEAD to the given reference.
func (m *Manager) Checkout(refStr string) error {
	ref, err := ParseRemoteRepoReference(refStr)
	if err != nil {
		return err
	}

	return m.CheckoutByRef(ref)
}

// CheckoutByRef moves repository's HEAD associated to the given reference to another reference.
// Name of the repository is extracted from the given reference.
func (m *Manager) CheckoutByRef(ref reference.Named) error {
	err := validRemoteRepoReference(ref)
	if err != nil {
		return err
	}

	img, err := m.lookupImage(ref.String())
	if err != nil {
		return fmt.Errorf("local reference to %q not found: %v", ref.String(), err)
	}

	err = m.store.AddNames(img.ID(), []string{ref.Name() + ":" + HeadTag})
	if err != nil {
		return err
	}

	return nil
}

// Fetch updates every repository reference.
func (m *Manager) Fetch(refStr string) error {
	ref, err := ParseRemoteRepoReference(refStr)
	if err != nil {
		return fmt.Errorf("failed to parse repository name: %w", err)
	}

	return m.FetchByNamedRef(ref)
}

// FetchByNamedRef updates every repository reference.
func (m *Manager) FetchByNamedRef(named reference.Named) error {
	repo, _ := m.Repository(named.Name())
	if repo == nil {
		return fmt.Errorf("repository not found")
	}

	// List images with same name as repository
	images, err := m.runtime.ListImages(context.Background(), []string{}, &libimage.ListImagesOptions{
		Filters: []string{"reference=" + named.Name() + ":*"},
	})
	if err != nil {
		return fmt.Errorf("failed to list references to repository: %w", err)
	}

	// Updates every reference
	// For every images matching the repository name
	for _, img := range images {
		// Iterate over every name of this image
		for _, name := range img.Names() {
			ref, err := ParseRemoteRepoReference(name)
			// Filter HEAD reference
			if err == ErrRemoteRepoReferenceContainsHeadTag {
				continue
			}
			if err != nil {
				return err
			}

			// Filter name that don't match repository name
			if ref.Name() != named.Name() {
				continue
			}

			// Pull reference
			err = m.pullRef(ref)
			if err != nil {
				return err
			}
		}
	}

	// Now pull the given reference
	return m.pullRef(named)
}
