package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Setting environment variables.
func setEnvVars(t *testing.T, vars map[string]string) {
	t.Helper()
	for key, value := range vars {
		err := os.Setenv(key, value)
		require.NoError(t, err, "failed to set env var %s", key)

		// Ensure that the env vars are cleared after the test
		t.Cleanup(func() {
			os.Unsetenv(key)
		})
	}
}

func TestConfigTelegramEnv(t *testing.T) {
	expected := &Config{
		Telegram: TelegramConfig{
			Token:     "123",
			Timeout:   10 * time.Second,
			Chats:     []int64{1, 2, 3},
			Admins:    []int64{1, 2, 3},
			Whitelist: []int64{1, 2, 3},
			Blacklist: []int64{1, 2, 3},
			IgnoreVia: false,
		},
	}

	setEnvVars(t, map[string]string{
		"TELEGRAM_TOKEN":      "123",
		"TELEGRAM_TIMEOUT":    "10s",
		"TELEGRAM_CHATS":      "1,2,3",
		"TELEGRAM_ADMINS":     "1,2,3",
		"TELEGRAM_WHITELIST":  "1,2,3",
		"TELEGRAM_BLACKLIST":  "1,2,3",
		"TELEGRAM_IGNORE_VIA": "false",
	})

	actual, err := MustLoadConfig()
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.NotNil(t, actual.Telegram)

	// Compare each field with the expected values
	require.Equal(t, expected.Telegram.Token, actual.Telegram.Token)
	require.Equal(t, expected.Telegram.Timeout, actual.Telegram.Timeout)
	require.Equal(t, expected.Telegram.Chats, actual.Telegram.Chats)
	require.Equal(t, expected.Telegram.Admins, actual.Telegram.Admins)
	require.Equal(t, expected.Telegram.Whitelist, actual.Telegram.Whitelist)
	require.Equal(t, expected.Telegram.Blacklist, actual.Telegram.Blacklist)
	require.Equal(t, expected.Telegram.IgnoreVia, actual.Telegram.IgnoreVia)
}
