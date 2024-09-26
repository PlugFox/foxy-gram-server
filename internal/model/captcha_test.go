package model

import (
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/global"
	"github.com/stretchr/testify/require"
)

func TestCaptchaHash(t *testing.T) {
	testcases := []struct {
		Name         string
		Captcha      *Captcha
		ExpectedHash string
	}{
		{
			Name: "Captcha with all fields",
			Captcha: &Captcha{
				ID:         1,
				Digits:     "123456",
				Length:     6,
				Width:      200,
				Height:     100,
				Expiration: 10,
				UserID:     1,
				ChatID:     1,
				MessageID:  1,
				ExpiresAt:  time.Time{},
				UpdatedAt:  time.Time{},
			},
			ExpectedHash: "2942d3c0d40ba7395a56e03802cd5999372145f48b5863e87c2c0ef8e61f2b8e",
		},
		{
			Name: "Captcha with missing fields",
			Captcha: &Captcha{
				ID: 1,
			},
			ExpectedHash: "0ecf4641150794b30a691cb480acb4561966e51ff4f865a77d141c9849f966ac",
		},
	}

	InitHashFunction()

	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			hash, err := testcase.Captcha.Hash()
			require.NoError(t, err)
			require.NotEmpty(t, hash)

			hash2, _ := testcase.Captcha.Hash()
			require.Equal(t, hash, hash2)
			require.Equal(t, testcase.ExpectedHash, hash)
		})
	}
}

func TestBytesToString(t *testing.T) {
	bytes := []byte{1, 2, 3, 4, 5}

	var strNumbers []string
	for _, b := range bytes {
		strNumbers = append(strNumbers, strconv.Itoa(int(b)))
	}

	str := strings.Join(strNumbers, "")

	require.NotEmpty(t, str)
	require.Equal(t, "12345", str)
}

func TestGenerateNewCaptcha(t *testing.T) {
	global.Config = &config.Config{
		Captcha: config.CaptchaConfig{
			Length:     6,
			Width:      200,
			Height:     100,
			Expiration: 10,
		},
	}

	captcha, err := GenerateCaptcha(io.Discard)
	require.NoError(t, err)
	require.NotNil(t, captcha)
	require.NotEmpty(t, captcha.Digits)
	require.NotEmpty(t, captcha.ExpiresAt)
}
