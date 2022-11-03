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

// RepositoryByNamedRef returns the repository associated with the given reference name.
func (m *Manager) RepositoryByNamedRef(named reference.Named) (*Repository, error) {
	if err := validRepoName(named); err != nil {
		return nil, err
	}

	img, _, err := m.runtime.LookupImage(named.Name(), &libimage.LookupImageOptions{
		Architecture:   runtime.GOARCH,
		OS:             runtime.GOOS,
		Variant:        "",
		PlatformPolicy: libimage.PlatformPolicyDefault,
		ManifestList:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup image: %w", err)
	}

	return newRepository(img), nil
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
func (m *Manager) Clone(name string) (*Repository, error) {
	named, err := ParseRemoteRepoReference(name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote repository reference: %w", err)
	}

	return m.CloneByNamedRef(named)
}

// CloneByNamedRef clones remote repository with the given remote repository reference to local storage.
func (m *Manager) CloneByNamedRef(named reference.Named) (*Repository, error) {
	// Ensure local repository doesn't exist
	_, err := m.RepositoryByNamedRef(named)
	if err == nil {
		return nil, ErrLocalRepositoryAlreadyExist
	}

	maxRetries := uint(3)
	retryDelay := time.Second
	_, err = m.runtime.Pull(context.Background(), named.Name(), config.PullPolicyNewer, &libimage.PullOptions{
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
	if err != nil {
		return nil, err
	}

	repository, err := m.Repository(named.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository after pulling it %q: %w", named.Name(), err)
	}

	m.store.AddNames(repository.ID(), []string{named.Name() + ":" + HeadTag})

	return repository, nil
}
