package reference

import (
	"fmt"
)

var _ error = ParseError{}

type parseErrorType uint

const (
	localRepositoryParseErrorType parseErrorType = iota
	remoteRepositoryParseErrorType
	repositoryNameParseErrorType
	repositoryTagParseErrorType
	identifierParseErrorType
)

func (rk parseErrorType) String() string {
	switch rk {
	case localRepositoryParseErrorType:
		return "local reference"
	case remoteRepositoryParseErrorType:
		return "remote reference"
	case repositoryNameParseErrorType:
		return "name"
	case repositoryTagParseErrorType:
		return "tag"
	case identifierParseErrorType:
		return "identifier"

	default:
		panic("unknown reference kind")
	}
}

type ParseError struct {
	etype parseErrorType
	err   error
}

func wrapParseError(etype parseErrorType, err error) error {
	if err == nil {
		panic("can't wrap nil error")
	}

	return ParseError{
		etype: etype,
		err:   err,
	}
}

// Error implements error
func (e ParseError) Error() string {
	return fmt.Sprintf("failed to parse repository %v: %v", e.etype, e.err)
}
