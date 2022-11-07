package libocitree

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ReferenceOffsetRegex = regexp.MustCompile(`(\^+|~\d+)$`)

	ErrRepoNameContainsTagOrDigest        = errors.New("repository name can't contain any tag or digest")
	ErrRepoReferenceIsNotNamed            = errors.New("repository reference is not a named reference")
	ErrRemoteRepoReferenceContainsHeadTag = errors.New("remote repository reference contains a HEAD tag")
	ErrRemoteRepoReferenceIsRelative      = errors.New("remote repository reference is relative")
)

// Reference define a repository reference.
// Repository references are similar to docker reference with one difference.
//
// Tags can be used as relative reference as in git:
//
// - HEAD^, HEAD~1: correspond to the commit just before the HEAD reference.
//
// - HEAD^^, HEAD~2: correspond to the commit just before the HEAD^ reference.
type Reference struct {
	name   string
	tag    string
	offset uint
}

// ParseNamedReference parses a reference string and returns a new Reference object.
// If the reference doesn't contain a tag, it default to HEAD tag.
// An error is returned if parsing fails or reference isn't named.
func ParseNamedReference(refStr string) (Reference, error) {
	return parseReference(refStr, HeadTag)
}

// ParseRemoteNamedReference apply ParseNamedReference on the given string and ensure
// the reference is absolute and doesn't contains a HEAD tag.
func ParseRemoteNamedReference(refStr string) (ref Reference, err error) {
	ref, err = parseReference(refStr, LatestTag)
	if err != nil {
		return ref, err
	}

	if ref.tag == HeadTag {
		return ref, ErrRemoteRepoReferenceContainsHeadTag
	}

	if ref.IsRelative() {
		return ref, ErrRemoteRepoReferenceIsRelative
	}

	return ref, nil
}

func parseReference(refStr, defaultTag string) (ref Reference, err error) {
	// Parse offset if match regex
	if offsetIndex := ReferenceOffsetRegex.FindStringIndex(refStr); offsetIndex != nil {
		ref.offset, err = parseOffset(refStr[offsetIndex[0]:offsetIndex[1]])
		if err != nil {
			return ref, err
		}
		refStr = refStr[:offsetIndex[0]]
	}

	// Parse docker ref
	dockerRef, err := reference.ParseAnyReference(refStr)
	if err != nil {
		return ref, fmt.Errorf("failed to parse docker reference: %w", err)
	}

	// Ensure reference is named
	dockerNamedRef, isNamed := dockerRef.(reference.Named)
	if !isNamed {
		return ref, ErrRepoReferenceIsNotNamed
	}
	ref.name = dockerNamedRef.Name()

	// Add default HEAD tag if needed
	if tagged, isTagged := dockerRef.(reference.Tagged); isTagged {
		ref.tag = tagged.Tag()
	} else {
		ref.tag = defaultTag
	}

	return ref, nil
}

// HeadReferenceFromNamedReference returns an absolute reference based on the
// given reference.Named with a HEAD tag.
func HeadReferenceFromNamedReference(ref reference.Named) Reference {
	return Reference{
		name:   ref.Name(),
		tag:    HeadTag,
		offset: 0,
	}
}

// Name implements reference.Named
func (r *Reference) Name() string {
	return r.name
}

// Tag implements reference.Tagged
func (r *Reference) Tag() string {
	return r.tag
}

// String implements reference.Reference
func (r *Reference) String() string {
	if r.offset == 0 {
		return fmt.Sprintf("%v:%v", r.name, r.tag)
	} else {
		return fmt.Sprintf("%v:%v~%v", r.name, r.tag, r.offset)
	}
}

// Offset returns relative offset reference.
func (r *Reference) Offset() uint {
	return r.offset
}

// IsRelative returns true if the reference isn't absolute and
// contains an offset.
func (r *Reference) IsRelative() bool {
	return r.offset != 0
}

// ToRemoteReference returns a valid remote reference by replacing
// invalid component (tag, offset) with default values.
//
// `docker.io/library/archlinux:HEAD` --> `docker.io/library/archlinux:latest`
//
// `docker.io/library/archlinux:HEAD~2` --> `docker.io/library/archlinux:latest`
//
// `docker.io/library/archlinux:edge` --> `docker.io/library/archlinux:edge`
//
// `docker.io/library/archlinux:edge~3` --> `docker.io/library/archlinux:edge`
func (r *Reference) ToRemoteReference() Reference {
	ref := Reference{
		name: r.Name(),
		tag:  r.Tag(),
	}
	if r.tag == HeadTag {
		ref.tag = LatestTag
	}
	ref.offset = 0

	return ref
}

func parseOffset(s string) (uint, error) {
	if s[0] == '^' {
		return uint(len(s)), nil
	} else if s[0] == '~' {
		i, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0, err
		}

		if i < 0 {
			panic("parseOffset argument don't match ReferenceOffsetRegex")
		}

		return uint(i), nil
	}

	panic("parseOffset argument don't match ReferenceOffsetRegex")
}
