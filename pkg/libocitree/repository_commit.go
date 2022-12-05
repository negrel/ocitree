package libocitree

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/storage/pkg/archive"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/sirupsen/logrus"
)

const CommitPrefix = "/bin/sh -c #(ocitree) "

var (
	ErrRebaseNothingToRebase    = errors.New("nothing to rebase")
	ErrRebaseUnknownInstruction = errors.New("unknown instruction")
)

// CommitOptions contains options to add a commit to repository.
type CommitOptions struct {
	CreatedBy string
	Message   string

	ReportWriter io.Writer
}

func (r *Repository) commit(builder *buildah.Builder, options CommitOptions) error {
	sref := r.runtime.storageReference(r.headRef)

	builder.SetHistoryComment(options.Message + "\n")
	builder.SetCreatedBy(CommitPrefix + options.CreatedBy)

	_, _, _, err := builder.Commit(context.Background(), sref, buildah.CommitOptions{
		PreferredManifestType: "",
		Compression:           archive.Gzip,
		SignaturePolicyPath:   "",
		AdditionalTags:        nil,
		ReportWriter:          options.ReportWriter,
		HistoryTimestamp:      nil,
		SystemContext:         r.runtime.systemContext(),
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
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	err = r.ReloadHead()
	if err != nil {
		return fmt.Errorf("failed to reload repository's HEAD after commit: %w", err)
	}

	return nil
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

// Add commits the given source files to HEAD.
func (r *Repository) Add(dest string, options AddOptions, sources ...string) error {
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

	builder, err := r.runtime.repoBuilder(r.headRef, options.ReportWriter)
	if err != nil {
		return err
	}
	defer builder.Delete()

	err = builder.Add(dest, false, options.toAddAndCopyOptions(), sources...)
	if err != nil {
		return fmt.Errorf("failed to add files to image: %w", err)
	}

	createdBy := fmt.Sprintf("%v --chown=%q --chmod=%q %v %v", AddCommitOperation,
		options.Chown, options.Chmod, stringList(sources), dest)

	return r.commit(builder, CommitOptions{
		CreatedBy:    createdBy,
		Message:      options.Message,
		ReportWriter: options.ReportWriter,
	})
}

type stringList []string

// String implements fmt.Stringer
func (fl stringList) String() string {
	builder := strings.Builder{}
	builder.WriteRune('[')

	for i, f := range fl {
		builder.WriteRune('"')
		builder.WriteString(strings.ReplaceAll(f, `"`, `\"`))
		builder.WriteRune('"')
		if i < len(fl)-1 {
			builder.WriteRune(' ')
		}
	}

	builder.WriteRune(']')

	return builder.String()
}

// ExecOptions holds options for Manager.Exec method.
type ExecOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Message      string
	ReportWriter io.Writer
}

func (r *Repository) Exec(options ExecOptions, cmd string, args ...string) error {
	builder, err := r.runtime.repoBuilder(r.headRef, nil)
	if err != nil {
		return err
	}
	defer builder.Delete()

	command := make([]string, 0, len(args)+1)
	command = append(command, cmd)
	command = append(command, args...)
	err = builder.Run(command, buildah.RunOptions{
		Logger:           logrus.StandardLogger(),
		Hostname:         "",
		Isolation:        define.IsolationChroot,
		Runtime:          "",
		Args:             nil,
		NoHosts:          false,
		NoPivot:          false,
		Mounts:           nil,
		Env:              nil,
		User:             "root",
		WorkingDir:       "",
		ContextDir:       "",
		Shell:            "",
		Cmd:              []string{},
		Entrypoint:       []string{},
		NamespaceOptions: nil,
		ConfigureNetwork: 0,
		CNIPluginPath:    "",
		CNIConfigDir:     "",
		Terminal:         0,
		TerminalSize:     nil,
		Stdin:            options.Stdin,
		Stdout:           options.Stdout,
		Stderr:           options.Stderr,
		Quiet:            true,
		AddCapabilities: []string{
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_FOWNER",
			"CAP_FSETID",
			"CAP_KILL",
			"CAP_NET_BIND_SERVICE",
			"CAP_SETFCAP",
			"CAP_SETGID",
			"CAP_SETPCAP",
			"CAP_SETUID",
			"CAP_SYS_CHROOT",
		},
		DropCapabilities:    nil,
		Devices:             []define.BuildahDevice{},
		Secrets:             nil,
		SSHSources:          nil,
		RunMounts:           nil,
		StageMountPoints:    nil,
		ExternalImageMounts: nil,
		SystemContext:       r.runtime.systemContext(),
		CgroupManager:       "",
	})
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return r.commit(builder, CommitOptions{
		CreatedBy:    ExecCommitOperation.String() + stringList(command).String(),
		Message:      options.Message,
		ReportWriter: options.ReportWriter,
	})
}

// RebaseSession starts and returns a new RebaseSession with the given tag as base reference.
func (r *Repository) RebaseSession(tagged reference.Tagged) (*RebaseSession, error) {
	ref, err := reference.RemoteFromNamedTagged(r.headRef, tagged)
	if err != nil {
		return nil, err
	}

	return newRebaseSession(r.runtime, r, ref)
}
