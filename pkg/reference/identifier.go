package reference

import (
	"errors"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ErrNotAnIdentifier = errors.New("not an identifier")
)

var _ Reference = Identifier{}

// Identifier define a unique commit identifier.
type Identifier struct {
	id string
}

// IdentifierFromString returns an Identifier from the given string after
// validating it. An error is returned if the identifier is invalid.
func IdentifierFromString(idstr string) (Identifier, error) {
	id, err := reference.ParseAnyReference(idstr)
	if err != nil {
		return Identifier{}, wrapParseError(identifierParseErrorType, err)
	}

	// Not an identifier
	if _, isNamed := id.(Named); isNamed {
		return Identifier{}, wrapParseError(identifierParseErrorType, ErrNotAnIdentifier)
	}

	return Identifier{id: id.String()}, nil
}

// IsFullIdentifier returns true if the reference is a full identifier.
func (i Identifier) IsFullIdentifier() bool {
	return len(i.id) == 64
}

// AbsoluteReference implements Reference
func (i Identifier) AbsoluteReference() string {
	return i.id
}
