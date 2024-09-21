package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserID(t *testing.T) {
	userID := UserID(123)
	require.Equal(t, int64(123), userID.ToInt64())
	require.Equal(t, "123", userID.ToString())
}

func TestUserHash(t *testing.T) {
	testcases := []struct {
		Name         string
		User         *User
		ExpectedHash string
	}{
		{
			Name: "User with all fields",
			User: &User{
				ID:           1,
				FirstName:    "John",
				LastName:     "Doe",
				Username:     "johndoe",
				LanguageCode: "en",
				IsPremium:    true,
				IsBot:        false,
			},
			ExpectedHash: "800c13dda822bb8ff8d6530e302d4581a5fcc06d6cf0e154076a838ff9097e67",
		},
		{
			Name: "User with missing fields",
			User: &User{
				ID: 1,
			},
			ExpectedHash: "dd5df2c087cfe8a1e5173a5b9fc5f0f96bda3a69a8698214d02bbc6d9553263d",
		},
	}

	InitHashFunction()
	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			hash, err := testcase.User.Hash()
			require.NoError(t, err)
			require.NotEmpty(t, hash)
			hash2, _ := testcase.User.Hash()
			require.Equal(t, hash, hash2)
			require.Equal(t, testcase.ExpectedHash, hash)
		})
	}
}
