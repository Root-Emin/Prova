package auth

import (
	"strings"
	"testing"

	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCodeConfig() config.AuthConfig {
	return config.AuthConfig{CodeLength: 6, CodePepper: "a-pepper-long-enough"}
}

func TestNewLoginCodeService_RejectsWeakPepper(t *testing.T) {
	_, err := NewLoginCodeService(config.AuthConfig{CodeLength: 6, CodePepper: ""})
	assert.Error(t, err, "an empty pepper leaves the digest column brute-forceable")

	_, err = NewLoginCodeService(config.AuthConfig{CodeLength: 6, CodePepper: "short"})
	assert.Error(t, err)
}

func TestLoginCodeService_GenerateShape(t *testing.T) {
	svc, err := NewLoginCodeService(testCodeConfig())
	require.NoError(t, err)

	for i := 0; i < 200; i++ {
		code, digest, err := svc.Generate()
		require.NoError(t, err)

		assert.Len(t, code, 6)
		assert.True(t, strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) == -1,
			"code must be all digits, got %q", code)
		assert.NotEmpty(t, digest)
		assert.NotContains(t, digest, code, "the digest must not carry the code")
	}
}

// Codes with leading zeros are as valid as any other. A generator that skipped
// them would quietly shrink the keyspace by ten percent.
func TestLoginCodeService_ProducesLeadingZeroCodes(t *testing.T) {
	svc, err := NewLoginCodeService(testCodeConfig())
	require.NoError(t, err)

	var seenLeadingZero bool
	for i := 0; i < 2000 && !seenLeadingZero; i++ {
		code, _, err := svc.Generate()
		require.NoError(t, err)
		if strings.HasPrefix(code, "0") {
			seenLeadingZero = true
		}
	}
	assert.True(t, seenLeadingZero, "expected at least one code starting with 0 in 2000 draws")
}

func TestLoginCodeService_Matches(t *testing.T) {
	svc, err := NewLoginCodeService(testCodeConfig())
	require.NoError(t, err)

	code, digest, err := svc.Generate()
	require.NoError(t, err)

	assert.True(t, svc.Matches(digest, code))
	assert.False(t, svc.Matches(digest, "000000"+code))
	assert.False(t, svc.Matches(digest, ""))
	assert.False(t, svc.Matches("", code))
}

// The pepper is what a leaked database does not contain. A digest computed
// under one pepper must be worthless under another.
func TestLoginCodeService_DigestIsPepperBound(t *testing.T) {
	svcA, err := NewLoginCodeService(config.AuthConfig{CodeLength: 6, CodePepper: "pepper-number-one-x"})
	require.NoError(t, err)
	svcB, err := NewLoginCodeService(config.AuthConfig{CodeLength: 6, CodePepper: "pepper-number-two-y"})
	require.NoError(t, err)

	code, digest, err := svcA.Generate()
	require.NoError(t, err)

	assert.True(t, svcA.Matches(digest, code))
	assert.False(t, svcB.Matches(digest, code))
}
