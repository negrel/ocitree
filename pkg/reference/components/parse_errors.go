package components

import (
	"fmt"
)

var _ error = ParseError{}

type parseErrorType uint

const (
	nameParseErrorType parseErrorType = iota
	tagParseErrorType
	idParseErrorType
)

func (rk parseErrorType) String() string {
	switch rk {
	case nameParseErrorType:
		return "name"
	case tagParseErrorType:
		return "tag"
	case idParseErrorType:
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
