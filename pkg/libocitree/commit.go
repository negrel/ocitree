package libocitree

import (
	"time"

	"github.com/containers/common/libimage"
)

// Commit define the history of a single layer.
type Commit struct {
	history libimage.ImageHistory
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

// Comment returns the comment associated to this commit.
func (c *Commit) Comment() string {
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
