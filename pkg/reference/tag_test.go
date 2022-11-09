package reference

import (
	"testing"

	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/require"
)

func TestTagFromString(t *testing.T) {
	for _, test := range []struct {
		name          string
		tag           string
		expectedError error
	}{
		{
			name:          "EmptyTag",
			tag:           "",
			expectedError: wrapParseError(repositoryTagParseErrorType, reference.ErrTagInvalidFormat),
		},
		{
			name:          "InvalidTag",
			tag:           ".",
			expectedError: wrapParseError(repositoryTagParseErrorType, reference.ErrTagInvalidFormat),
		},
		{
			name:          "ValidTag",
			tag:           "1.0.0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tag, err := TagFromString(test.tag)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.tag, tag.Tag())
			require.Equal(t, test.tag, tag.String())
		})
	}
}
