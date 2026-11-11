package user

import (
	_ "embed"
	"errors"
	"strings"
	"sync"
	"unicode/utf8"
)

var (
	ErrPasswordTooShort = errors.New("the password must be at least 15 characters")
	ErrPasswordTooLong  = errors.New("the password can be at most 256 characters")
	ErrPasswordCommon   = errors.New("the password is too common")
	ErrPasswordPersonal = errors.New("the password must differ from the username and display name")
)

// MinPasswordLength and MaxPasswordLength bound a new password, in characters.
const (
	MinPasswordLength = 15
	MaxPasswordLength = 256
)

//go:embed common-passwords.txt
var commonPasswordsFile string

var commonPasswords = sync.OnceValue(func() map[string]bool {
	m := map[string]bool{}
	for line := range strings.Lines(commonPasswordsFile) {
		if line = strings.TrimSuffix(line, "\n"); line != "" && !strings.HasPrefix(line, "#") {
			m[line] = true
		}
	}
	return m
})

// checkNewPassword applies the rules for a new password, after NIST SP 800-63B: long enough and not guessable, with
// no required mix of letters, digits or symbols. Existing passwords are never checked against them.
func checkNewPassword(password, username, displayName string) error {
	switch n := utf8.RuneCountInString(password); {
	case n < MinPasswordLength:
		return ErrPasswordTooShort
	case n > MaxPasswordLength:
		return ErrPasswordTooLong
	}
	lower := strings.ToLower(password)
	if lower == strings.ToLower(strings.TrimSpace(username)) || lower == strings.ToLower(strings.TrimSpace(displayName)) {
		return ErrPasswordPersonal
	}
	if commonPasswords()[lower] {
		return ErrPasswordCommon
	}
	return nil
}
