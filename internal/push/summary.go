package push

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// Notifier sends the daily summaries that are due. Subscriber is the installation's address, given to push services
// as the sender's contact; Client is nil for the default HTTP client.
type Notifier struct {
	Push       *Store
	Stock      *stock.Store
	Subscriber string
	Client     webpush.HTTPClient
}

// maxBody is how much of a summary's body is spent on names: push services take about 4 KB once encrypted.
const maxBody = 3000

// SendDue sends the summary of every user with a subscription whose time has come today and who has had none today.
func (n *Notifier) SendDue(ctx context.Context) error {
	now := n.Push.Now().In(time.Local)
	today, minute := now.Format(time.DateOnly), now.Hour()*60+now.Minute()
	due, err := n.due(ctx, today, minute)
	if err != nil || len(due) == 0 {
		return err
	}
	keys, err := n.Push.Keys(ctx)
	if err != nil {
		return err
	}
	warnings, err := n.Stock.Warnings(ctx)
	if err != nil {
		return err
	}
	for _, u := range due {
		var chosen []stock.Item
		for _, it := range warnings {
			if u.choice.Sections[it.Section] {
				chosen = append(chosen, it)
			}
		}
		news, err := n.Push.summarize(ctx, u.id, today, chosen)
		if err != nil {
			return err
		}
		if len(news) > 0 {
			if err := n.send(ctx, u, keys, message(news), len(news)); err != nil {
				return err
			}
		}
	}
	return nil
}

type dueUser struct {
	id       int64
	username string
	choice   Choice
}

func (n *Notifier) due(ctx context.Context, today string, minute int) ([]dueUser, error) {
	def := DefaultChoice()
	rows, err := n.Push.DB.QueryContext(ctx, `
		SELECT u.id, u.username, coalesce(c.minutes, ?), coalesce(c.medicines, 1), coalesce(c.pantry, 1),
			coalesce(c.supplies, 1), coalesce(c.last_day, '')
		FROM app_user u LEFT JOIN push_choice c ON c.user_id = u.id
		WHERE EXISTS (SELECT 1 FROM push_subscription s WHERE s.user_id = u.id)`, def.Minutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var due []dueUser
	for rows.Next() {
		var u dueUser
		var medicines, pantry, supplies bool
		var lastDay string
		if err := rows.Scan(&u.id, &u.username, &u.choice.Minutes, &medicines, &pantry, &supplies,
			&lastDay); err != nil {
			return nil, err
		}
		if minute < u.choice.Minutes || lastDay == today {
			continue
		}
		u.choice.Sections = map[stock.Section]bool{stock.Medicines: medicines, stock.Pantry: pantry,
			stock.Supplies: supplies}
		due = append(due, u)
	}
	return due, rows.Err()
}

// summarize returns the rows of current the user wasn't told of in their state, then remembers current as told and
// today as the day of the user's last summary.
func (s *Store) summarize(ctx context.Context, userID int64, today string, current []stock.Item) ([]stock.Item,
	error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }() // a no-op after Commit
	told, err := toldStates(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	var news []stock.Item
	for _, it := range current {
		if told[it.ID] != it.State {
			news = append(news, it)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM push_told WHERE user_id = ?", userID); err != nil {
		return nil, err
	}
	for _, it := range current {
		if _, err := tx.ExecContext(ctx, "INSERT INTO push_told (user_id, item_id, state) VALUES (?, ?, ?)",
			userID, it.ID, it.State); err != nil {
			return nil, err
		}
	}
	def := DefaultChoice()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO push_choice (user_id, minutes, medicines, pantry, supplies, last_day) VALUES (?, ?, 1, 1, 1, ?)
		ON CONFLICT (user_id) DO UPDATE SET last_day = excluded.last_day`, userID, def.Minutes, today); err != nil {
		return nil, err
	}
	return news, tx.Commit()
}

func toldStates(ctx context.Context, tx *sql.Tx, userID int64) (map[int64]stock.State, error) {
	rows, err := tx.QueryContext(ctx, "SELECT item_id, state FROM push_told WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	told := map[int64]stock.State{}
	for rows.Next() {
		var id int64
		var state stock.State
		if err := rows.Scan(&id, &state); err != nil {
			return nil, err
		}
		told[id] = state
	}
	return told, rows.Err()
}

// payload is what the service worker shows.
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// message is the summary of items, already in the order to show: expired first, then by expiry date.
func message(items []stock.Item) []byte {
	title := fmt.Sprintf("%d things to check", len(items))
	if len(items) == 1 {
		title = "1 thing to check"
	}
	var lines []string
	size := 0
	for i, it := range items {
		state := "expiring"
		if it.State == stock.Expired {
			state = "expired"
		}
		line := fmt.Sprintf("%s (%s, %s)", it.Name, it.Section.Label(), state)
		need := len(line)
		if rest := len(items) - i - 1; rest > 0 {
			need += len(fmt.Sprintf("\nand %d more", rest)) // room to say so after this line
		}
		if size+need > maxBody {
			lines = append(lines, fmt.Sprintf("and %d more", len(items)-i))
			break
		}
		lines = append(lines, line)
		size += len(line) + 1
	}
	b, err := json.Marshal(payload{Title: title, Body: strings.Join(lines, "\n")})
	if err != nil {
		panic(err) // two strings always marshal
	}
	return b
}

// send delivers msg, the summary of that many rows, to every device of the user, removing those the push service no longer knows.
func (n *Notifier) send(ctx context.Context, u dueUser, keys Keys, msg []byte, rows int) error {
	subs, err := n.Push.subscriptions(ctx, u.id)
	if err != nil {
		return err
	}
	delivered := 0
	for _, sub := range subs {
		ws := &webpush.Subscription{Endpoint: sub.Endpoint, Keys: webpush.Keys{P256dh: sub.Keys.P256dh,
			Auth: sub.Keys.Auth}}
		resp, err := webpush.SendNotificationWithContext(ctx, msg, ws, &webpush.Options{
			HTTPClient: n.Client, Subscriber: n.Subscriber, Topic: "summary", TTL: 24 * 60 * 60,
			Urgency: webpush.UrgencyNormal, VAPIDPublicKey: keys.Public, VAPIDPrivateKey: keys.Private,
		})
		if err != nil {
			slog.Warn("notification not delivered", "user", u.username, "err", err)
			continue
		}
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
			slog.Info("notification subscription gone", "user", u.username)
			if err := n.Push.remove(ctx, sub.Endpoint); err != nil {
				return err
			}
		case resp.StatusCode >= 300:
			slog.Warn("notification not delivered", "user", u.username, "status", resp.StatusCode)
		default:
			delivered++
		}
	}
	slog.Info("notification summary sent", "user", u.username, "rows", rows, "devices", delivered)
	return nil
}
