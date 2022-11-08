package reference

import (
	"fmt"

	"github.com/containers/image/v5/docker/reference"
)

var _ error = ParseError{}

type ParseError struct {
	err error
}

func wrapParseError(err error) error {
	if err == nil {
		panic("can't wrap nil error")
	}

	return ParseError{
		err: err,
	}
}

// Error implements error
func (e ParseError) Error() string {
	return fmt.Sprintf("failed to parse repository reference/name: %v", e.err)
}

func parseRef(refStr string) (Named, error) {
	ref, err := reference.ParseNormalizedNamed(refStr)
	if err != nil {
		return nil, wrapParseError(err)
	}

	return ref, nil
}
