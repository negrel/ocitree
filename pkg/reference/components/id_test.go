package components

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentifierFromString(t *testing.T) {
	for _, test := range []struct {
		name          string
		identifier    string
		expectedID    string
		expectedError error
	}{
		{
			name:          "WithNameAndTag",
			identifier:    "docker.io/library/ubuntu:18.04@sha256:98706f0f213dbd440021993a82d2f70451a73698315370ae8615cc468ac06624",
			expectedID:    "98706f0f213dbd440021993a82d2f70451a73698315370ae8615cc468ac06624",
			expectedError: ErrIdContainsName,
		},
		{
			name:       "WithoutName",
			identifier: "98706f0f213dbd440021993a82d2f70451a73698315370ae8615cc468ac06624",
			expectedID: "98706f0f213dbd440021993a82d2f70451a73698315370ae8615cc468ac06624",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			identifier, err := IdFromString(test.identifier)

			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedID, identifier.ID())
		})
	}
}
