package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatHash(t *testing.T) {
	testcases := []struct {
		Name         string
		Chat         *Chat
		ExpectedHash string
	}{
		{
			Name: "Chat with all fields",
			Chat: &Chat{
				ID:        1,
				Title:     "Test chat",
				Username:  "testchat",
				Type:      "group",
				IsPrivate: false,
			},
			ExpectedHash: "7f223aa54651a323d7ac381825c768547c03157d216fec46d466cb3e3d56f9d9",
		},
		{
			Name: "Chat with missing fields",
			Chat: &Chat{
				ID: 1,
			},
			ExpectedHash: "95303ab31255ac1f260cc17b8b9c78059bd27f708232159b6738ea680a7f09b1",
		},
	}

	InitHashFunction()

	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			hash, err := testcase.Chat.Hash()
			require.NoError(t, err)
			require.NotEmpty(t, hash)

			hash2, _ := testcase.Chat.Hash()
			require.Equal(t, hash, hash2)
			require.Equal(t, testcase.ExpectedHash, hash)
		})
	}
}
