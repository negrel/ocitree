package libocitree

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	storageTransport "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/hashicorp/go-multierror"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/sirupsen/logrus"
)

var (
	ErrLocalRepositoryAlreadyExist = errors.New("local repository with the same name already exist")
	ErrLocalRepositoryUnknown      = errors.New("unknown local repository")
)

// Manager defines a repositories manager.
type Manager struct {
	store   storage.Store
	runtime *libimage.Runtime
}

// systemContext implements imageStore
func (m *Manager) systemContext() *types.SystemContext {
	return m.runtime.SystemContext()
}

// storageReference implements imageStore
func (m *Manager) storageReference(ref reference.LocalRepository) types.ImageReference {
	r, err := storageTransport.Transport.NewStoreReference(m.store, ref, "")
	if err != nil {
		panic(err)
	}

	return r
}

// listImages implements imageStore
func (m *Manager) listImages(filters ...string) ([]*libimage.Image, error) {
	return m.runtime.ListImages(context.Background(), nil, &libimage.ListImagesOptions{
		Filters: filters,
	})
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
// An error is returned if local repository is missing or corrupted.
func (m *Manager) Repository(name reference.Named) (*Repository, error) {
	return newRepositoryFromName(m, name)
}

// lookupImage returns the image associated to the given ref.
// This function expect a fully qualified reference and will use default values
// ("latest" for tag, "docker.io" for registry) if not.
func (m *Manager) lookupImage(ref reference.LocalRepository) (*libimage.Image, error) {
	img, _, err := m.runtime.LookupImage(ref.String(), &libimage.LookupImageOptions{
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

// LocalRepositoryExist returns true if a local repository with the given name
// exist.
func (m *Manager) LocalRepositoryExist(name reference.Named) bool {
	img, err := m.lookupImage(reference.LocalHeadFromNamed(name))
	return img != nil && err == nil
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
		result[i], err = newRepositoryFromImage(m, image)
		if err != nil {
			logrus.Debugf("image %q was listed with HEAD reference but repository can't be created from it: %v", image.Names(), err)
			continue
		}
	}

	return result, nil
}

// CloneOptions holds clone options.
type CloneOptions struct {
	PullOptions
}

// Clone clones remote repository with the given name to local storage.
func (m *Manager) Clone(remoteRef reference.RemoteRepository, options CloneOptions) error {
	headRef := reference.LocalHeadFromNamed(remoteRef)

	// Ensure repository doesn't exist
	if m.LocalRepositoryExist(reference.NameFromNamed(remoteRef)) {
		return ErrLocalRepositoryAlreadyExist
	}

	// Pull image
	images, err := m.pullRef(remoteRef, &options.PullOptions)
	if err != nil {
		return err
	}

	// Assign HEAD reference
	img := images[0]
	err = m.store.AddNames(img.ID(), []string{headRef.String()})
	if err != nil {
		return fmt.Errorf("failed to add HEAD reference to image: %w", err)
	}

	return nil
}

// PullOptions holds configuration options for pulling operations.
type PullOptions struct {
	MaxRetries   uint
	RetryDelay   time.Duration
	ReportWriter io.Writer
}

func (m *Manager) pullRef(ref reference.RemoteRepository, options *PullOptions) ([]*libimage.Image, error) {
	return m.runtime.Pull(context.Background(), ref.String(), config.PullPolicyNewer, &libimage.PullOptions{
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
			MaxRetries:                       &options.MaxRetries,
			RetryDelay:                       &options.RetryDelay,
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
			Writer:                           options.ReportWriter,
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
}

// FetchOptions holds fetch options.
type FetchOptions struct {
	PullOptions
}

// Fetch fetches multiple version of the given repository reference.
// It starts by updating every HEAD tags and then finally, it downloads
// the given remote reference.
func (m *Manager) Fetch(remoteRef reference.RemoteRepository, options FetchOptions) error {
	if !m.LocalRepositoryExist(reference.NameFromNamed(remoteRef)) {
		return ErrLocalRepositoryUnknown
	}

	// List images with same name as repository
	images, err := m.runtime.ListImages(context.Background(), []string{}, &libimage.ListImagesOptions{
		Filters: []string{"reference=" + remoteRef.Name() + ":*"},
	})
	if err != nil {
		return fmt.Errorf("failed to list references to repository: %w", err)
	}

	// Updates every reference
	// For every images matching the repository name
	var pullErrs *multierror.Error
	for _, img := range images {
		// Iterate over every name of this image
		for _, name := range img.Names() {
			imgRemoteRef, err := reference.RemoteFromString(name)
			// Filter HEAD reference
			if err != nil {
				logrus.Debugf("skipping %q because of error: %v", name, err)
				continue
			}

			// Filter image name that don't match repository name
			if imgRemoteRef.Name() != remoteRef.Name() {
				continue
			}

			// Pull image
			_, err = m.pullRef(imgRemoteRef, &options.PullOptions)
			if err != nil {
				multierror.Append(pullErrs, err)
			}
		}
	}

	// Pull the given reference now
	_, err = m.pullRef(remoteRef, &options.PullOptions)
	if err != nil {
		multierror.Append(pullErrs, err)
	}

	return pullErrs.ErrorOrNil()
}

func (m *Manager) repoBuilder(ref reference.Named, reportWriter io.Writer) (*buildah.Builder, error) {
	builder, err := buildah.NewBuilder(context.Background(), m.store, buildah.BuilderOptions{
		Args:                  nil,
		FromImage:             ref.String(),
		ContainerSuffix:       "ocitree",
		Container:             ref.Name(),
		PullPolicy:            buildah.PullNever,
		Registry:              "",
		BlobDirectory:         "",
		Logger:                logrus.StandardLogger(),
		Mount:                 false,
		SignaturePolicyPath:   "",
		ReportWriter:          reportWriter,
		SystemContext:         m.runtime.SystemContext(),
		DefaultMountsFilePath: "",
		Isolation:             0,
		NamespaceOptions:      nil,
		ConfigureNetwork:      0,
		CNIPluginPath:         "",
		CNIConfigDir:          "",
		NetworkInterface:      nil,
		IDMappingOptions:      nil,
		Capabilities:          nil,
		CommonBuildOpts:       nil,
		Format:                "",
		Devices:               nil,
		DefaultEnv:            nil,
		MaxPullRetries:        0,
		PullRetryDelay:        0,
		OciDecryptConfig:      nil,
		ProcessLabel:          "",
		MountLabel:            "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	return builder, nil
}
