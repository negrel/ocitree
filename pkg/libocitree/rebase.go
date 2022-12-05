package libocitree

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/common/libimage"
	"github.com/containers/storage/pkg/archive"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/sirupsen/logrus"
)

var (
	ErrUnknownRebaseChoice = errors.New("unknown rebase choice")
)

type RebaseChoice uint

const (
	PickRebaseChoice RebaseChoice = iota
)

// String implements fmt.Stringer.
func (rc RebaseChoice) String() string {
	switch rc {
	case PickRebaseChoice:
		return "pick"
	default:
		return "unknown"
	}
}

// RebaseCommit correspond to a commit and a rebase choice.
type RebaseCommit struct {
	Commit
	Choice RebaseChoice
}

// RebaseCommits define a read only wrapper over a slice of RebaseCommit.
type RebaseCommits struct {
	commits []RebaseCommit
}

func newRebaseCommits(commits Commits, newBaseID string) (RebaseCommits, error) {
	rebaseCommits := RebaseCommits{
		commits: make([]RebaseCommit, 0, len(commits)),
	}

	for i, commit := range commits {
		// If commit id is new base or
		// commit wasn't created using ocitree or
		// commit is the first we can't rebase them
		if commit.ID() == newBaseID || !commit.WasCreatedByOcitree() || i == len(commits)-1 {
			break
		}

		rebaseCommits.commits = append(rebaseCommits.commits, RebaseCommit{
			Commit: commit,
			Choice: PickRebaseChoice,
		})
	}

	return rebaseCommits, nil
}

// Get returns the RebaseCommit at the given index.
func (rc RebaseCommits) Get(i int) *RebaseCommit {
	return &rc.commits[i]
}

// Len returns length of underlying RebaseCommit slice.
func (rc RebaseCommits) Len() int {
	return len(rc.commits)
}

// String implements fmt.Stringer.
func (rc RebaseCommits) String() string {
	builder := strings.Builder{}

	for _, c := range rc.commits {
		builder.WriteString(c.Choice.String())
		builder.WriteString(" ")
		builder.WriteString(c.Commit.ID()[:8] + " ")
		builder.WriteString(c.Commit.Comment())
		builder.WriteString("\n")
	}

	return builder.String()
}

func (rc RebaseCommits) done() {
	rc.commits = rc.commits[:0]
}

// RebaseSession define a rebase session of a repository.
type RebaseSession struct {
	baseRef    reference.RemoteRepository
	baseImage  *libimage.Image
	repository *Repository
	commits    RebaseCommits
	runtime    imageRuntime
	builder    *buildah.Builder
}

func newRebaseSession(store imageRuntime, repo *Repository, tagged reference.Tagged) (*RebaseSession, error) {
	baseRef, err := reference.RemoteFromNamedTagged(repo.HeadRef(), tagged)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote repository reference: %w", err)
	}

	baseImage, err := store.lookupImage(reference.LocalFromRemote(baseRef))
	if err != nil {
		return nil, fmt.Errorf("failed to find new base: %w", err)
	}

	commits, err := repo.Commits()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository commits: %w", err)
	}

	rebaseCommits, err := newRebaseCommits(commits, baseImage.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to check rebase commits: %w", err)
	}

	builder, err := store.repoBuilder(baseRef, os.Stderr)
	if err != nil {
		return nil, err
	}

	return &RebaseSession{
		baseRef:    baseRef,
		baseImage:  baseImage,
		repository: repo,
		commits:    rebaseCommits,
		builder:    builder,
		runtime:    store,
	}, nil
}

// Commits returns the RebaseCommits part of this session.
func (rs *RebaseSession) Commits() RebaseCommits {
	return rs.commits
}

func (rs *RebaseSession) Apply() error {
	defer rs.commits.done()

	// Validate commits before executing them
	for i := 0; i < rs.commits.Len(); i++ {
		commit := rs.commits.Get(i)

		if commit.Choice == PickRebaseChoice {
			if commit.Commit.ID() == "" {
				return fmt.Errorf("can't apply commit number %d: can't pick a commit with no associated layer id", i)
			}
		}
	}

	// Nothing to do
	if rs.commits.Len() == 0 {
		return nil
	}

	// Execute rebase
	logrus.Debugf("commits:\n%v", rs.commits)
	for i := rs.commits.Len() - 1; i >= 0; i-- {
		commit := rs.commits.Get(i)

		switch commit.Choice {
		case PickRebaseChoice:
			logrus.Debugf("picking commit %v (%v)", i, commit.Commit.ID())
			err := rs.pick(commit)
			if err != nil {
				return fmt.Errorf("failed to pick commit %v (%v): %w", i, commit.Commit.ID(), err)
			}

		default:
			return ErrUnknownRebaseChoice
		}
	}

	// Move HEAD reference
	err := rs.repository.Checkout(reference.RebaseHeadTag)
	if err != nil {
		return fmt.Errorf("failed to checkout to rebase head: %w", err)
	}

	// Remove REBASE_HEAD reference
	err = rs.repository.removeLocalTag(reference.RebaseHeadTag)
	if err != nil {
		return fmt.Errorf("failed to remove rebase head tag: %w", err)
	}

	return nil
}

func (rs *RebaseSession) pick(commit *RebaseCommit) error {
	// Compute diff
	diff, err := rs.runtime.diff(commit.Parent(), &commit.Commit)
	if err != nil {
		return fmt.Errorf("failed to compute diff between commit %v and %v: %w", commit.Parent().ID(), commit.ID(), err)
	}

	// We must clone as diff holds a lock until close is called.
	diffClone, err := io.ReadAll(diff)
	if err != nil {
		return fmt.Errorf("failed to clone diff: %w", err)
	}
	diff.Close()

	// Mount builder container
	dstMountpoint, err := rs.builder.Mount("")
	if err != nil {
		return fmt.Errorf("failed to mount rebase builder container: %w", err)
	}
	defer rs.builder.Unmount()

	// Apply diff
	_, err = archive.ApplyLayer(dstMountpoint, bytes.NewBuffer(diffClone))
	if err != nil {
		return fmt.Errorf("failed to apply layer: %w", err)
	}

	err = rs.commitRebaseHead()
	if err != nil {
		return fmt.Errorf("failed to commit rebase head: %w", err)
	}

	return nil
}

func (rs *RebaseSession) Delete() error {
	return rs.builder.Delete()
}

func (rs *RebaseSession) RebaseHead() reference.LocalRepository {
	return reference.LocalFromNamedTagged(rs.baseRef, reference.RebaseHeadTag)
}

func (rs *RebaseSession) commitRebaseHead() error {
	sref := rs.runtime.storageReference(rs.RebaseHead())
	_, _, _, err := rs.builder.Commit(context.Background(), sref, buildah.CommitOptions{
		PreferredManifestType: "",
		Compression:           archive.Uncompressed,
		SignaturePolicyPath:   "",
		AdditionalTags:        nil,
		ReportWriter:          os.Stderr,
		HistoryTimestamp:      nil,
		SystemContext:         rs.runtime.systemContext(),
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
		return fmt.Errorf("failed to commit rebase head: %w", err)
	}

	return nil
}
