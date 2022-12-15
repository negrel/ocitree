package reference

import (
	"testing"

	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	for _, test := range []struct {
		testName      string
		name          string
		expectedName  string
		expectedError error
	}{
		{
			testName:      "InvalidName",
			name:          "...",
			expectedError: reference.ErrReferenceInvalidFormat,
		},
		{
			testName:     "ValidFullyQualifiedName",
			name:         "docker.io/library/archlinux",
			expectedName: "docker.io/library/archlinux",
		},
		{
			testName:     "ValidShortName",
			name:         "archlinux",
			expectedName: "docker.io/library/archlinux",
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			name, err := NameFromString(test.name)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, err, test.expectedError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedName, name.String())
		})
	}
}
