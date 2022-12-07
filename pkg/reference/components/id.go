package components

import (
	"errors"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ErrIdContainsName     = errors.New("identifier contais name")
	ErrIdContainsTag      = errors.New("identifier contais tag")
	ErrIdContainsNoDigest = errors.New("identifier contains no digest")
)

var _ Identifier = ID{}

type ID struct {
	id string
}

func IdFromString(idstr string) (ID, error) {
	ref, err := reference.ParseAnyReference(idstr)
	if err != nil {
		return ID{}, wrapParseError(idParseErrorType, err)
	}

	if _, isNamed := ref.(reference.Named); isNamed {
		return ID{}, ErrIdContainsName
	}

	if _, isTagged := ref.(reference.Tagged); isTagged {
		return ID{}, ErrIdContainsTag
	}

	if d, isDigested := ref.(reference.Digested); isDigested {
		return ID{id: d.Digest().Encoded()}, nil
	}

	return ID{}, ErrIdContainsNoDigest
}

// ID implements Identifier
func (i ID) ID() string {
	return i.id
}

// String implements fmt.Stringer
func (i ID) String() string {
	return i.id
}
