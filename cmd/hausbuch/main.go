// Command hausbuch runs the household wiki server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/web"
)

func main() {
	addr := envOr("HAUSBUCH_LISTEN", ":8080")
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	dbPath := envOr("HAUSBUCH_DB_PATH", "/data/hausbuch.db")
	_, err := db.Open(context.Background(), dbPath)
	if err != nil {
		slog.Error("cannot open database", "path", dbPath, "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           web.NewHandler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
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
