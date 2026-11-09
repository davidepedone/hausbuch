// Command hausbuch runs the household wiki server and manages its accounts.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	_ "time/tzdata" // the image has no time zone database, and TZ needs one

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
	"github.com/davidepedone/hausbuch/internal/user"
	"github.com/davidepedone/hausbuch/internal/web"
)

// version is the running release, set by the release build through the linker.
var version = "dev"

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
	backupDir := envOr("HAUSBUCH_BACKUP_DIR", filepath.Join(filepath.Dir(dbPath), "backups"))

	switch cmd {
	case "serve":
		return serve(addr, dbPath, backupDir)
	case "healthcheck":
		if err := healthcheck(addr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "user":
		database, err := db.Open(context.Background(), dbPath, backupDir)
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

func serve(addr, dbPath, backupDir string) int {
	timeout := user.DefaultSessionTimeout
	if v := os.Getenv("HAUSBUCH_SESSION_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			slog.Error("HAUSBUCH_SESSION_TIMEOUT must be a positive duration such as 720h", "value", v)
			return 1
		}
		timeout = d
	}
	keep := 7
	if v := os.Getenv("HAUSBUCH_BACKUP_KEEP"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			slog.Error("HAUSBUCH_BACKUP_KEEP must be a whole number of at least 1", "value", v)
			return 1
		}
		keep = n
	}
	lead, err := leadTimes()
	if err != nil {
		slog.Error(err.Error())
		return 1
	}
	database, err := db.Open(context.Background(), dbPath, backupDir)
	if err != nil {
		slog.Error("cannot open database", "path", dbPath, "err", err)
		return 1
	}
	go backUpDaily(database, backupDir, keep)
	users := user.NewStore(database)
	users.SessionTimeout = timeout
	pages := page.NewStore(database)
	if err := pages.Reindex(context.Background()); err != nil {
		slog.Error("cannot build the search index", "err", err)
		return 1
	}
	go cleanUp(users, pages)
	items := stock.NewStore(database)
	items.Lead = lead

	srv := &http.Server{
		Addr:              addr,
		Handler:           web.NewHandler(users, pages, items, version),
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("listening", "addr", addr)
	err = srv.ListenAndServe()
	slog.Error("server stopped", "err", err)
	return 1
}

// cleanUp removes expired sessions and pages deleted for too long, at start and then daily. Signing in removes
// expired sessions too, and creating or renaming a page removes such pages first.
func cleanUp(users *user.Store, pages *page.Store) {
	for {
		if err := users.DeleteExpiredSessions(context.Background()); err != nil {
			slog.Error("remove expired sessions", "err", err)
		}
		if err := pages.RemoveExpired(context.Background()); err != nil {
			slog.Error("remove deleted pages", "err", err)
		}
		time.Sleep(24 * time.Hour)
	}
}

// backUpDaily makes the day's copy of the database if it is missing, at start and then every hour, so that every day
// the server runs gets one whatever time it started. A failure is retried at the next check.
func backUpDaily(database *sql.DB, dir string, keep int) {
	for {
		if err := db.DailyBackup(context.Background(), database, dir, keep, time.Now()); err != nil {
			slog.Error("daily backup", "dir", dir, "err", err)
		}
		time.Sleep(time.Hour)
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

// leadTimes returns how many days before their expiry date each section's items count as expiring.
func leadTimes() (map[stock.Section]int, error) {
	lead := stock.DefaultLead()
	for section, key := range map[stock.Section]string{
		stock.Medicines: "HAUSBUCH_MEDICINES_LEAD_DAYS",
		stock.Pantry:    "HAUSBUCH_PANTRY_LEAD_DAYS",
		stock.Supplies:  "HAUSBUCH_SUPPLIES_LEAD_DAYS",
	} {
		if v := os.Getenv(key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("%s must be a whole number of at least 0, not %q", key, v)
			}
			lead[section] = n
		}
	}
	return lead, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
