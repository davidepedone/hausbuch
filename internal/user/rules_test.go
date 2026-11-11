package user

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewPasswordRules(t *testing.T) {
	for _, c := range []struct {
		password string
		want     error
	}{
		{strings.Repeat("a", 14), ErrPasswordTooShort},
		{strings.Repeat("ä", 14), ErrPasswordTooShort}, // characters, not bytes
		{"", ErrPasswordTooShort},
		{strings.Repeat("a", 257), ErrPasswordTooLong},
		{strings.Repeat("a", 256), nil},
		{"PasswordPassword", ErrPasswordCommon},
		{"Anna Rossi Bianchi", ErrPasswordPersonal}, // the display name, in another case
		{"ANNA.ROSSI.EXAMPLE", ErrPasswordPersonal}, // the username
		{"lemon chair river cloud", nil},            // a passphrase: no digits, symbols or capitals needed
	} {
		if err := checkNewPassword(c.password, "anna.rossi.example", "anna rossi bianchi"); !errors.Is(err, c.want) {
			t.Errorf("checkNewPassword(%.20q) = %v, want %v", c.password, err, c.want)
		}
	}
}

func TestRulesApplyToEveryNewPassword(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if _, err := s.Add(ctx, "anna", "Anna", "short"); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("Add with a short password: %v", err)
	}
	u, err := s.Add(ctx, "anna", "Anna", "spoon kettle garden")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPassword(ctx, "anna", "passwordpassword"); !errors.Is(err, ErrPasswordCommon) {
		t.Errorf("SetPassword with a common password: %v", err)
	}
	if err := s.SetPassword(ctx, "nobody", "lemon chair river cloud"); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("SetPassword of an unknown user: %v", err)
	}
	if err := s.ChangePassword(ctx, u.ID, "spoon kettle garden", "anna"); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("ChangePassword to a short password: %v", err)
	}
	if _, err := s.Authenticate(ctx, "anna", "spoon kettle garden", here); err != nil {
		t.Fatalf("a refused password replaced the old one: %v", err)
	}
}

func TestExistingShortPasswordStillWorks(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if _, err := s.Add(ctx, "anna", "Anna", "spoon kettle garden"); err != nil {
		t.Fatal(err)
	}
	// A password set before the rules existed.
	hash, err := hashPassword(ctx, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("UPDATE app_user SET password_hash = ?", hash); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, "anna", "pw", here); err != nil {
		t.Fatalf("sign-in with a short password set before the rules: %v", err)
	}
}
