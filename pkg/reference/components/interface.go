package components

import (
	"errors"
)

var (
	ErrNotIdentifierNorTag = errors.New("not an identifier nor a tag")
)

// Named define object with a name.
type Named interface {
	Name() string
}

// Tagged is an object which has a tag.
type Tagged interface {
	Tag() string
}

// Identifier is an object which has an ID.
type Identifier interface {
	ID() string
}

// IdentifierOrTag returns either an ID or a tag.
type IdentifierOrTag interface {
	// returned ID will starts with "@sha256:" and tag with ":"
	IdOrTag() string
}

func IdentifierOrTagFromString(idtag string) (IdentifierOrTag, error) {
	id, err := IdFromString(idtag)
	if err == nil {
		return id, nil
	}

	tag, err := TagFromString(idtag)
	if err == nil {
		return tag, nil
	}

	return nil, ErrNotIdentifierNorTag
}
