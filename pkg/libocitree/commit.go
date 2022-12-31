package libocitree

import (
	"errors"
	"strings"
	"time"

	"github.com/containers/common/libimage"
)

var (
	ErrUnknownCommitOperation = errors.New("unknown commit operation")
)

type CommitOperation uint

const (
	UnknownCommitOperation CommitOperation = iota
	ExecCommitOperation
	AddCommitOperation
)

func commitOperationFromString(str string) CommitOperation {
	switch str {
	case "EXEC":
		return ExecCommitOperation
	case "ADD":
		return AddCommitOperation
	default:
		return UnknownCommitOperation
	}
}

// String implements fmt.Stringer.
func (co CommitOperation) String() string {
	switch co {
	case ExecCommitOperation:
		return "EXEC"
	case AddCommitOperation:
		return "ADD"
	default:
		return "UNKNOWN"
	}
}

type Commits []Commit

func newCommits(history []libimage.ImageHistory) Commits {
	commits := make(Commits, len(history))

	for i, h := range history {
		commits[i] = Commit{
			history: h,
			parent:  nil,
		}

		// If not first commit, set parent field
		if i < len(history)-1 {
			commits[i].parent = &commits[i+1]
		}
	}

	return commits
}

// Commit define the history of a single layer.
type Commit struct {
	history libimage.ImageHistory
	parent  *Commit
}

func newCommit(history libimage.ImageHistory) Commit {
	return Commit{
		history: history,
	}
}

// ID returns the ID associated to this commit.
func (c *Commit) ID() string {
	return c.history.ID
}

// Message returns the message associated to this commit.
func (c *Commit) Message() string {
	if splitted := strings.Split(c.history.Comment, "\nFROM"); len(splitted) != 1 {
		return splitted[0]
	}

	return c.history.Comment
}

// Tags returns the tags associated to this commit.
func (c *Commit) Tags() []string {
	return c.history.Tags
}

// CreatedBy returns the operations that created this commit.
func (c *Commit) CreatedBy() string {
	return c.history.CreatedBy
}

// CreationDate returns the creation date of the commit.
func (c *Commit) CreationDate() *time.Time {
	return c.history.Created
}

// Size returns the size of rootfs change contained in this commit.
func (c *Commit) Size() int64 {
	return c.history.Size
}

// IsCreatedByOCITree returns true if the commit was
// made using libocitree.
func (c *Commit) WasCreatedByOcitree() bool {
	return strings.HasPrefix(c.CreatedBy(), CommitPrefix)
}

// Operation returns the operation used to create this commit
func (c *Commit) Operation() CommitOperation {
	if !c.WasCreatedByOcitree() {
		return UnknownCommitOperation
	}

	splitted := strings.SplitN(c.history.CreatedBy[len(CommitPrefix):], " ", 2)
	return commitOperationFromString(splitted[0])
}

// Parent returns the parent commit.
func (c *Commit) Parent() *Commit {
	return c.parent
}
