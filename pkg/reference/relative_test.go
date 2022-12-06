package reference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelativeFromString(t *testing.T) {
	for _, test := range []struct {
		name                 string
		reference            string
		expectedBase         string
		expectedOffset       uint
		expectedErrorMessage string
	}{
		{
			name:           "WithoutOffset",
			reference:      "docker.io/library/archlinux:latest",
			expectedBase:   "docker.io/library/archlinux:latest",
			expectedOffset: 0,
		},
		{
			name:           "WithoutOffset/ShortIdentifier",
			reference:      "0622ce1ea915",
			expectedBase:   "0622ce1ea915",
			expectedOffset: 0,
		},
		{
			name:           "WithoutOffset/Identifier",
			reference:      "c07b565814ed2ab787ebc839780f034a4e0dd37c32d87bac8fc541023c61bd6a",
			expectedBase:   "c07b565814ed2ab787ebc839780f034a4e0dd37c32d87bac8fc541023c61bd6a",
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
			name:           "WithOffset/Tilde/ShortIdentifier",
			reference:      "0622ce1ea915:~287",
			expectedBase:   "0622ce1ea915",
			expectedOffset: 287,
		},
		{
			name:           "WithOffset/Circumflex/ShortIdentifier",
			reference:      "0622ce1ea915:^^^^",
			expectedBase:   "0622ce1ea915",
			expectedOffset: 4,
		},
		{
			name:           "WithOffset/Tilde/Identifier",
			reference:      "c07b565814ed2ab787ebc839780f034a4e0dd37c32d87bac8fc541023c61bd6a:~4",
			expectedBase:   "c07b565814ed2ab787ebc839780f034a4e0dd37c32d87bac8fc541023c61bd6a",
			expectedOffset: 4,
		},
		{
			name:           "WithOffset/Circumflex/Identifier",
			reference:      "c07b565814ed2ab787ebc839780f034a4e0dd37c32d87bac8fc541023c61bd6a:^",
			expectedBase:   "c07b565814ed2ab787ebc839780f034a4e0dd37c32d87bac8fc541023c61bd6a",
			expectedOffset: 1,
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
			name:                 "InvalidBaseRef",
			reference:            "archlinux:...",
			expectedErrorMessage: "failed to parse local base reference: failed to parse repository local reference: invalid reference format",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := RelativeFromString(test.reference)
			if test.expectedErrorMessage != "" {
				require.Error(t, err)
				require.Equal(t, test.expectedErrorMessage, err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedBase, ref.Base().AbsoluteReference())
			require.Equal(t, test.expectedOffset, ref.Offset())
		})
	}
}
