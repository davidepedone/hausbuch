package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"golang.org/x/term"

	"github.com/davidepedone/hausbuch/internal/user"
)

const userUsage = `Usage:
  hausbuch user add USERNAME "DISPLAY NAME"   add an account; asks for its password
  hausbuch user list                           list accounts
  hausbuch user set-password USERNAME          set an account's password; asks for it
  hausbuch user disable USERNAME               stop an account from signing in and end its sessions
  hausbuch user enable USERNAME                let a disabled account sign in again
  hausbuch user unlock USERNAME                lift the lock after repeated failed sign-ins
  hausbuch user link USERNAME                  print a one-time sign-in link, valid for 15 minutes

Passwords are asked for without echo, or read from standard input when it is not a terminal.
`

// runUser runs a user command; publicURL is HAUSBUCH_URL, or "" when not set.
func runUser(ctx context.Context, store *user.Store, publicURL string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, userUsage)
		return 2
	}
	var err error
	switch cmd, args := args[0], args[1:]; {
	case cmd == "add" && len(args) == 2:
		err = addUser(ctx, store, args[0], args[1], stdin, stdout, stderr)
	case cmd == "list" && len(args) == 0:
		err = listUsers(ctx, store, stdout)
	case cmd == "set-password" && len(args) == 1:
		var password string
		if password, err = readNewPassword(stdin, stderr); err == nil {
			err = done(stdout, store.SetPassword(ctx, args[0], password), "Password of %s changed.", args[0])
		}
	case cmd == "disable" && len(args) == 1:
		err = done(stdout, store.Disable(ctx, args[0]), "Disabled %s and ended their sessions.", args[0])
	case cmd == "enable" && len(args) == 1:
		err = done(stdout, store.Enable(ctx, args[0]), "Enabled %s.", args[0])
	case cmd == "unlock" && len(args) == 1:
		err = done(stdout, store.Unlock(ctx, args[0]), "Unlocked %s.", args[0])
	case cmd == "link" && len(args) == 1:
		err = printLink(ctx, store, publicURL, args[0], stdout)
	case cmd == "help":
		fmt.Fprint(stdout, userUsage)
	default:
		fmt.Fprint(stderr, userUsage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// done prints the confirmation when err is nil, and returns err.
func done(stdout io.Writer, err error, format string, args ...any) error {
	if err == nil {
		fmt.Fprintf(stdout, format+"\n", args...)
	}
	return err
}

func addUser(ctx context.Context, store *user.Store, username, displayName string, stdin io.Reader, stdout, stderr io.Writer) error {
	password, err := readNewPassword(stdin, stderr)
	if err != nil {
		return err
	}
	u, err := store.Add(ctx, username, displayName, password)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Added %s (%s).\n", u.Username, u.DisplayName)
	return nil
}

func printLink(ctx context.Context, store *user.Store, publicURL, username string, stdout io.Writer) error {
	token, err := store.CreateLink(ctx, username)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Sign-in link for %s, valid for %d minutes and for one sign-in:\n%s/link/%s\n",
		username, int(user.LinkLifetime.Minutes()), strings.TrimRight(publicURL, "/"), token)
	if publicURL == "" {
		fmt.Fprintln(stdout, "Open it at this installation's address; set HAUSBUCH_URL to have it printed whole.")
	}
	return nil
}

func listUsers(ctx context.Context, store *user.Store, stdout io.Writer) error {
	users, err := store.List(ctx)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USERNAME\tDISPLAY NAME\tSTATE")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\n", u.Username, u.DisplayName, state(u, store.Now()))
	}
	return w.Flush()
}

func state(u user.User, now time.Time) string {
	var s []string
	if u.Disabled {
		s = append(s, "disabled")
	}
	if u.LockedUntil.After(now) {
		s = append(s, "locked until "+u.LockedUntil.Format("15:04"))
	}
	if len(s) == 0 {
		return "active"
	}
	return strings.Join(s, ", ")
}

// readNewPassword asks for a password twice without echo when stdin is a terminal, otherwise it reads the first line
// of stdin, so that it can be piped in. Passwords are never taken from arguments, which end up in shell history and
// process lists.
func readNewPassword(stdin io.Reader, prompts io.Writer) (string, error) {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		first, err := promptPassword(f, prompts, "Password: ")
		if err != nil {
			return "", err
		}
		second, err := promptPassword(f, prompts, "Repeat password: ")
		if err != nil {
			return "", err
		}
		if first != second {
			return "", errors.New("the two passwords don't match")
		}
		return first, nil
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func promptPassword(f *os.File, prompts io.Writer, prompt string) (string, error) {
	fmt.Fprint(prompts, prompt)
	b, err := term.ReadPassword(int(f.Fd()))
	fmt.Fprintln(prompts)
	return string(b), err
}
