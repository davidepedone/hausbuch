// Command hausbuch runs the household wiki server and manages its accounts.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/user"
	"github.com/davidepedone/hausbuch/internal/web"
)

const usage = `Usage:
  hausbuch [serve]    run the server
  hausbuch healthcheck
  hausbuch user ...   manage accounts (hausbuch user help)
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	addr := envOr("HAUSBUCH_LISTEN", ":8080")
	dbPath := envOr("HAUSBUCH_DB_PATH", "/data/hausbuch.db")

	switch cmd {
	case "serve":
		return serve(addr, dbPath)
	case "healthcheck":
		if err := healthcheck(addr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "user":
		database, err := db.Open(context.Background(), dbPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer database.Close()
		return runUser(context.Background(), user.NewStore(database), args, stdin, stdout, stderr)
	default:
		fmt.Fprint(stderr, usage)
		return 2
	}
}

func serve(addr, dbPath string) int {
	timeout := user.DefaultSessionTimeout
	if v := os.Getenv("HAUSBUCH_SESSION_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			slog.Error("HAUSBUCH_SESSION_TIMEOUT must be a positive duration such as 720h", "value", v)
			return 1
		}
		timeout = d
	}
	database, err := db.Open(context.Background(), dbPath)
	if err != nil {
		slog.Error("cannot open database", "path", dbPath, "err", err)
		return 1
	}
	users := user.NewStore(database)
	users.SessionTimeout = timeout
	go removeExpiredSessions(users)

	srv := &http.Server{
		Addr:              addr,
		Handler:           web.NewHandler(users),
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("listening", "addr", addr)
	err = srv.ListenAndServe()
	slog.Error("server stopped", "err", err)
	return 1
}

// removeExpiredSessions runs at start and then daily; signing in removes expired sessions too.
func removeExpiredSessions(users *user.Store) {
	for {
		if err := users.DeleteExpiredSessions(context.Background()); err != nil {
			slog.Error("remove expired sessions", "err", err)
		}
		time.Sleep(24 * time.Hour)
	}
}

// healthcheck asks the server listening on addr for /healthz. The image has no shell or HTTP client, so its
// HEALTHCHECK runs this.
func healthcheck(addr string) error {
	url, err := healthURL(addr)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", url, resp.Status)
	}
	return nil
}

// healthURL turns a listen address into a URL reachable from inside the same host: a listener on all interfaces
// (":8080", "0.0.0.0:8080") is reached through the loopback address.
func healthURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("HAUSBUCH_LISTEN %q: %w", addr, err)
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/healthz", nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
