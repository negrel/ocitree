package reference

import (
	"testing"

	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/require"
)

func TestRelativeFromString(t *testing.T) {
	for _, test := range []struct {
		name           string
		reference      string
		expectedBase   string
		expectedOffset uint
		expectedError  error
	}{
		{
			name:           "WithoutOffset",
			reference:      "docker.io/library/archlinux:latest",
			expectedBase:   "docker.io/library/archlinux:latest",
			expectedOffset: 0,
		},
		{
			name:           "WithoutOffsetNotNormalized",
			reference:      "archlinux",
			expectedBase:   "docker.io/library/archlinux:HEAD",
			expectedOffset: 0,
		},
		{
			name:           "WithOffset/Tilde",
			reference:      "docker.io/library/archlinux:latest~2",
			expectedBase:   "docker.io/library/archlinux:latest",
			expectedOffset: 2,
		},
		{
			name:           "WithOffset/Circumflex",
			reference:      "docker.io/library/archlinux:latest^^^",
			expectedBase:   "docker.io/library/archlinux:latest",
			expectedOffset: 3,
		},
		{
			name:           "WithOffsetNotNormalized/Tilde",
			reference:      "docker.io/library/archlinux:latest~99",
			expectedBase:   "docker.io/library/archlinux:latest",
			expectedOffset: 99,
		},
		{
			name:           "WithOffsetNotNormalized/Circumflex",
			reference:      "archlinux:^^",
			expectedBase:   "docker.io/library/archlinux:HEAD",
			expectedOffset: 2,
		},
		{
			name:          "InvalidBaseRef",
			reference:     "archlinux:...",
			expectedError: reference.ErrReferenceInvalidFormat,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := RelativeFromString(test.reference)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedBase, ref.Base().String())
			require.Equal(t, test.expectedOffset, ref.Offset())
		})
	}
}
