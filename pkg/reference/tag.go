package reference

import "github.com/containers/image/v5/docker/reference"

var (
	HeadTag   = Tag{tag: Head}
	LatestTag = Tag{tag: Latest}
)

var _ reference.Tagged = Tag{}

// Tag define the tag component of a repository reference.
type Tag struct {
	tag string
}

var archlinuxName = Name{"docker.io/library/archlinux"}

// NameFromString returns a Tag from the given string after validating it.
func TagFromString(tag string) (Tag, error) {
	ref, err := reference.WithTag(archlinuxName, tag)
	if err != nil {
		return Tag{}, wrapParseError(repositoryTagParseErrorType, err)
	}

	return Tag{
		tag: ref.Tag(),
	}, nil
}

func TagFromTagged(tag Tagged) Tag {
	return Tag{tag: tag.Tag()}
}

// String implements reference.Reference
func (t Tag) String() string {
	return t.tag
}

// Tag implements Tagged
func (t Tag) Tag() string {
	return t.tag
}