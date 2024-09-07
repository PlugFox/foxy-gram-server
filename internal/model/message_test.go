package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessageHash(t *testing.T) {
	testcases := []struct {
		Name         string
		Message      *Message
		ExpectedHash string
	}{
		{
			Name: "Message with fields",
			Message: &Message{
				ID:          1,
				ChatID:      1,
				SenderID:    1,
				Text:        "Hello",
				Unixtime:    1234567890,
				IsForwarded: false,
			},
			ExpectedHash: "317fa2537c2e86dd73adc7e8b0a2413859baecf944ac86e30b6c8a4fa727aaea",
		},
		{
			Name: "Message with missing fields",
			Message: &Message{
				ID: 1,
			},
			ExpectedHash: "cbf9fb2a0f617fc842985b3e53f781880884b45040adf97bfdd68c532aa43c36",
		},
	}

	InitHashFunction()
	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			hash, err := testcase.Message.Hash()
			require.NoError(t, err)
			require.NotEmpty(t, hash)
			hash2, _ := testcase.Message.Hash()
			require.Equal(t, hash, hash2)
			require.Equal(t, testcase.ExpectedHash, hash)
		})
	}
}
