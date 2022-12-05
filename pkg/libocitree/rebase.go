package libocitree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/common/libimage"
	"github.com/containers/storage/pkg/archive"
	"github.com/negrel/ocitree/pkg/reference"
	"github.com/sirupsen/logrus"
)

var (
	ErrUnknownRebaseChoice  = errors.New("unknown rebase choice")
	interactiveEditHelpText = `#
# Commands:
# p, pick <commit> = use commit
# d, drop <commit> = remove commit
#
# These lines can't be re-ordered; they are executed from top to bottom.
#
`
)

type RebaseChoice uint

const (
	PickRebaseChoice RebaseChoice = iota
	DropRebaseChoice
	UnknownRebaseChoice
)

var validRebaseChoice = map[RebaseChoice]struct{}{
	PickRebaseChoice: {},
	DropRebaseChoice: {},
}

// String implements fmt.Stringer.
func (rc RebaseChoice) String() string {
	switch rc {
	case PickRebaseChoice:
		return "pick"
	case DropRebaseChoice:
		return "drop"
	default:
		return "unknown"
	}
}

func choiceFromString(str string) RebaseChoice {
	switch strings.ToLower(str) {
	case "pick", "p":
		return PickRebaseChoice

	case "drop", "d":
		return DropRebaseChoice

	default:
		return UnknownRebaseChoice
	}
}

// RebaseCommit correspond to a commit and a rebase choice.
type RebaseCommit struct {
	Commit
	Choice RebaseChoice
}

// RebaseCommits define a read only wrapper over a slice of RebaseCommit.
// Commits is ordered from newer to older.
type RebaseCommits struct {
	commits []RebaseCommit
}

func newRebaseCommits(commits Commits, newBaseID string) (RebaseCommits, error) {
	rebaseCommits := RebaseCommits{
		commits: make([]RebaseCommit, 0, len(commits)),
	}

	for i, commit := range commits {
		// If commit id has no associated image or
		// commit is new base or
		// commit wasn't created using ocitree or
		// commit is the first we can't rebase them
		if commit.ID() == "" || commit.ID() == newBaseID ||
			!commit.WasCreatedByOcitree() || i == len(commits)-1 {
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

	for i, c := range rc.commits {
		builder.WriteString(c.Choice.String())
		builder.WriteString(" ")
		builder.WriteString(c.Commit.ID()[:8] + " ")
		builder.WriteString(c.Commit.Comment())
		if i != rc.Len()-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func (rc RebaseCommits) parseAndSetChoices(choices string) error {
	length := rc.Len()

	// For each line
	commitIndex := 0
	for _, line := range strings.Split(choices, "\n") {
		if line == "" || (len(line) > 0 && line[0] == '#') {
			continue
		}

		// Split on space to parse choice
		splitted := strings.SplitN(line, " ", 2)
		if len(splitted) < 2 {
			continue
		}

		// parse commit choice
		rawChoice := splitted[0]
		choice := choiceFromString(rawChoice)
		if choice == UnknownRebaseChoice {
			return ErrUnknownRebaseChoice
		}

		// Set choice
		commit := rc.Get((length - 1) - commitIndex)
		commit.Choice = choice
		commitIndex++
	}

	return nil
}

// RebaseSession define a rebase session of a repository.
type RebaseSession struct {
	baseRef    reference.RemoteRepository
	baseImage  *libimage.Image
	repository *Repository
	commits    RebaseCommits
	runtime    imageRuntime
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
	err = baseImage.Tag(reference.LocalRebaseFromNamed(baseRef).String())
	if err != nil {
		return nil, fmt.Errorf("failed add REBASE_HEAD tag to new base: %w", err)
	}

	commits, err := repo.Commits()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository commits: %w", err)
	}

	rebaseCommits, err := newRebaseCommits(commits, baseImage.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to check rebase commits: %w", err)
	}

	return &RebaseSession{
		baseRef:    baseRef,
		baseImage:  baseImage,
		repository: repo,
		commits:    rebaseCommits,
		runtime:    store,
	}, nil
}

// BaseImage returns the rebase target image.
func (rs *RebaseSession) BaseImage() *libimage.Image {
	return rs.baseImage
}

// Commits returns the RebaseCommits part of this session.
func (rs *RebaseSession) Commits() RebaseCommits {
	return rs.commits
}

// Apply applies rebase choice. RebaseSession must no be used
// after this method has been called.
func (rs *RebaseSession) Apply() error {
	// Validate commits before executing them
	for i := 0; i < rs.commits.Len(); i++ {
		commit := rs.commits.Get(i)

		if _, ok := validRebaseChoice[commit.Choice]; !ok {
			return ErrUnknownRebaseChoice
		}

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

	// Apply rebase choice
	err := rs.apply()
	if err != nil {
		return err
	}

	// Move HEAD reference
	err = rs.repository.Checkout(reference.RebaseHeadTag)
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

func (rs *RebaseSession) apply() error {
	// Execute rebase
	logrus.Debugf("commits:\n%v", rs.commits)
	for i := rs.commits.Len() - 1; i >= 0; i-- {
		commit := rs.commits.Get(i)
		// drop commit
		if commit.Choice == DropRebaseChoice {
			continue
		}

		// Create builder
		builder, err := rs.builder()
		if err != nil {
			return fmt.Errorf("failed to create builder for commit %v (%v): %w", i, commit.ID(), err)
		}

		switch commit.Choice {
		case PickRebaseChoice:
			logrus.Debugf("picking commit %v (%v)", i, commit.Commit.ID())
			err := rs.pick(builder, commit)
			if err != nil {
				return fmt.Errorf("failed to pick commit %v (%v): %w", i, commit.Commit.ID(), err)
			}

		default:
			return ErrUnknownRebaseChoice
		}

		// Commit rebase head
		err = rs.commitRebaseHead(builder, CommitOptions{
			CreatedBy:    commit.CreatedBy()[len(CommitPrefix):],
			Message:      commit.Comment(),
			ReportWriter: os.Stderr,
		})
		if err != nil {
			return fmt.Errorf("failed to commit rebase head: %w", err)
		}

		// Delete builder
		err = builder.Delete()
		if err != nil {
			return fmt.Errorf("failed to delete rebase container: %w", err)
		}
	}

	return nil
}

func (rs *RebaseSession) pick(builder *buildah.Builder, commit *RebaseCommit) error {
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
	dstMountpoint, err := builder.Mount("")
	if err != nil {
		return fmt.Errorf("failed to mount rebase builder container: %w", err)
	}
	defer builder.Unmount()

	// Apply diff
	_, err = archive.ApplyLayer(dstMountpoint, bytes.NewBuffer(diffClone))
	if err != nil {
		return fmt.Errorf("failed to apply layer: %w", err)
	}

	builder.SetCreatedBy(commit.CreatedBy())

	return nil
}

// RebaseHead returns reference to rebase head.
func (rs *RebaseSession) RebaseHead() reference.LocalRepository {
	return reference.LocalFromNamedTagged(rs.baseRef, reference.RebaseHeadTag)
}

// create builder from REBASE_HEAD
func (rs *RebaseSession) builder() (*buildah.Builder, error) {
	return rs.repository.runtime.repoBuilder(rs.RebaseHead(), os.Stderr)
}

func (rs *RebaseSession) commitRebaseHead(builder *buildah.Builder, options CommitOptions) error {
	sref := rs.runtime.storageReference(rs.RebaseHead())
	err := commit(builder, options, sref, rs.runtime.systemContext())
	if err != nil {
		return fmt.Errorf("failed to commit rebase head: %w", err)
	}

	return nil
}

// InteractiveEdit starts an interactive session
func (rs *RebaseSession) InteractiveEdit() error {
	// Create temporary file
	f, err := os.CreateTemp(os.TempDir(), "ocitree-rebase-*")
	if err != nil {
		return fmt.Errorf("failed to create interactive rebase file: %w", err)
	}
	// Delete temporary file
	defer os.Remove(f.Name())

	// No commits, nothing to rebase
	if rs.commits.Len() == 0 {
		f.WriteString("noop")
	} else {
		// Reverse lines so commits are ordered from older to newer
		f.WriteString(reverseLines(rs.commits.String()))
		f.WriteString("\n\n")
		fmt.Fprintf(f, `# Rebase %v..%v onto %v (%v command(s))`,
			rs.repository.ID()[:8], rs.commits.Get(0).ID()[:8],
			rs.baseImage.ID()[:8], rs.commits.Len())
	}
	f.WriteString(interactiveEditHelpText)

	// Start editor process
	err = edit(f.Name())
	if err != nil {
		logrus.Errorf("failed to exec interactive rebase file editor: %v", err)
	}

	// Read choices
	f.Seek(0, io.SeekStart)
	b, _ := io.ReadAll(f)
	rawChoices := string(b)

	err = rs.commits.parseAndSetChoices(rawChoices)
	if err != nil {
		return fmt.Errorf("failed to parse choices: %w", err)
	}

	return nil
}

func edit(file string) error {
	// Try to execute $EDITOR editor
	editor := os.Getenv("EDITOR")
	// fallback to nano
	if editor == "" {
		editor = "nano"
	}
	cmd := exec.Command(editor, file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func reverseLines(str string) string {
	splitted := strings.Split(str, "\n")
	lastIndex := len(splitted) - 1

	for i := 0; i < len(splitted)/2; i++ {
		tmp := splitted[i]
		splitted[i] = splitted[lastIndex-i]
		splitted[lastIndex-i] = tmp
	}

	return strings.Join(splitted, "\n")
}
