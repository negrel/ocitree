package libocitree

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	storageTransport "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
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

// Clone clones remote repository with the given name to local storage.
func (m *Manager) Clone(remoteRef reference.RemoteRepository) error {
	headRef := reference.LocalHeadFromNamed(remoteRef)

	// Ensure repository doesn't exist
	if m.LocalRepositoryExist(reference.NameFromNamed(remoteRef)) {
		return ErrLocalRepositoryAlreadyExist
	}

	// Pull image
	images, err := m.pullRef(remoteRef)
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

func (m *Manager) pullRef(ref reference.RemoteRepository) ([]*libimage.Image, error) {
	maxRetries := uint(3)
	retryDelay := time.Second
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
}

// Checkout moves repository's HEAD associated to the given reference to another reference.
// Name of the repository is extracted from the given reference.
func (m *Manager) Checkout(ref reference.LocalRepository) error {
	img, err := m.lookupImage(ref)
	if err != nil {
		return fmt.Errorf("local reference not found: %v", err)
	}

	err = m.store.AddNames(img.ID(), []string{reference.LocalHeadFromNamed(ref).String()})
	if err != nil {
		return err
	}

	return nil
}

// Fetch updates every repository reference.
func (m *Manager) Fetch(remoteRef reference.RemoteRepository) error {
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
			_, err = m.pullRef(remoteRef)
			if err != nil {
				multierror.Append(pullErrs, err)
			}
		}
	}

	return pullErrs
}

// Add commit the given source files to the HEAD of the given repository name.
func (m *Manager) Add(repoName reference.Named, dest string, options AddOptions, sources ...string) error {
	for i, src := range sources {
		srcURL, err := url.Parse(src)
		if err != nil {
			return fmt.Errorf("failed to parse sources URL: %w", err)
		}

		// if filepath
		if srcURL.Scheme == "" {
			// get absolute path
			absSrc, err := filepath.Abs(src)
			if err != nil {
				return fmt.Errorf("failed to find absolute path to source: %v", err)
			}
			sources[i] = absSrc
		}
	}

	builder, err := m.repoBuilder(repoName, options.ReportWriter)
	if err != nil {
		return err
	}
	defer builder.Delete()

	err = builder.Add(dest, false, options.toAddAndCopyOptions(), sources...)
	if err != nil {
		return fmt.Errorf("failed to add files to image: %w", err)
	}

	createdBy := fmt.Sprintf("ADD --chown=%q --chmod=%q %v %v",
		options.Chown, options.Chmod, strings.Join(sources, ", "), dest)

	return m.commit(builder, repoName, CommitOptions{
		CreatedBy:    createdBy,
		Message:      options.Message,
		ReportWriter: options.ReportWriter,
	})
}

func (m *Manager) repoBuilder(repoName reference.Named, reportWriter io.Writer) (*buildah.Builder, error) {
	repoHeadRef := reference.LocalHeadFromNamed(repoName)

	builder, err := buildah.NewBuilder(context.Background(), m.store, buildah.BuilderOptions{
		Args:                  nil,
		FromImage:             repoHeadRef.String(),
		ContainerSuffix:       "ocitree",
		Container:             repoName.Name(),
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

func (m *Manager) commit(builder *buildah.Builder, repoName reference.Named, options CommitOptions) error {
	imgRef, err := storageTransport.Transport.ParseStoreReference(
		m.store, reference.LocalHeadFromNamed(repoName).String())
	if err != nil {
		return fmt.Errorf("failed to retrieve storage reference of HEAD of repository: %w", err)
	}

	builder.SetHistoryComment(options.Message + "\n")
	builder.SetCreatedBy("/bin/sh -c #(ocitree) " + options.CreatedBy)

	_, _, _, err = builder.Commit(context.Background(), imgRef, buildah.CommitOptions{
		PreferredManifestType: "",
		Compression:           archive.Gzip,
		SignaturePolicyPath:   "",
		AdditionalTags:        nil,
		ReportWriter:          options.ReportWriter,
		HistoryTimestamp:      nil,
		SystemContext:         m.runtime.SystemContext(),
		IIDFile:               "",
		Squash:                false,
		OmitHistory:           false,
		BlobDirectory:         "",
		EmptyLayer:            false,
		OmitTimestamp:         false,
		SignBy:                "",
		Manifest:              "",
		MaxRetries:            0,
		RetryDelay:            0,
		OciEncryptConfig:      nil,
		OciEncryptLayers:      nil,
		UnsetEnvs:             nil,
	})
	if err != nil {
		return fmt.Errorf("failed to commit repository: %w", err)
	}

	return nil
}

// CommitOptions contains options to add a commit to repository.
type CommitOptions struct {
	CreatedBy string
	Message   string

	ReportWriter io.Writer
}

// AddOptions holds option to Manager.Add method.
type AddOptions struct {
	//Chmod sets the access permissions of the destination content.
	Chmod string
	// Chown is a spec for the user who should be given ownership over the
	// newly-added content, potentially overriding permissions which would
	// otherwise be set to 0:0.
	Chown string

	Message string

	ReportWriter io.Writer
}

func (ao *AddOptions) toAddAndCopyOptions() buildah.AddAndCopyOptions {
	return buildah.AddAndCopyOptions{
		Chmod:             ao.Chmod,
		Chown:             ao.Chown,
		PreserveOwnership: false,
		Hasher:            nil,
		Excludes:          nil,
		IgnoreFile:        "",
		ContextDir:        "/",
		IDMappingOptions:  nil,
		DryRun:            false,
		StripSetuidBit:    false,
		StripSetgidBit:    false,
		StripStickyBit:    false,
	}
}

func (m *Manager) Exec(repoName reference.Named, options ExecOptions, args ...string) error {
	builder, err := m.repoBuilder(repoName, nil)
	if err != nil {
		return err
	}
	defer builder.Delete()

	err = builder.Run(args, buildah.RunOptions{
		Logger:              logrus.StandardLogger(),
		Hostname:            "",
		Isolation:           define.IsolationChroot,
		Runtime:             "",
		Args:                nil,
		NoHosts:             false,
		NoPivot:             false,
		Mounts:              nil,
		Env:                 nil,
		User:                "",
		WorkingDir:          "",
		ContextDir:          "",
		Shell:               "",
		Cmd:                 nil,
		Entrypoint:          nil,
		NamespaceOptions:    nil,
		ConfigureNetwork:    0,
		CNIPluginPath:       "",
		CNIConfigDir:        "",
		Terminal:            0,
		TerminalSize:        nil,
		Stdin:               options.Stdin,
		Stdout:              options.Stdout,
		Stderr:              options.Stderr,
		Quiet:               true,
		AddCapabilities:     nil,
		DropCapabilities:    nil,
		Devices:             []define.BuildahDevice{},
		Secrets:             nil,
		SSHSources:          nil,
		RunMounts:           nil,
		StageMountPoints:    nil,
		ExternalImageMounts: nil,
		SystemContext:       m.runtime.SystemContext(),
		CgroupManager:       "",
	})
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return m.commit(builder, repoName, CommitOptions{
		CreatedBy:    "EXEC " + strings.Join(args, " "),
		Message:      options.Message,
		ReportWriter: options.ReportWriter,
	})
}

// ExecOptions holds options for Manager.Exec method.
type ExecOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Message      string
	ReportWriter io.Writer
}
