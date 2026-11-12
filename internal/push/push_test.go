package push

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/stock"
	"github.com/davidepedone/hausbuch/internal/user"
)

var ctx = context.Background()

// browser is a device subscribed to a fake push service: it keeps its keys to decrypt what it receives.
type browser struct {
	key      *ecdh.PrivateKey
	auth     []byte
	endpoint string
	gone     bool // the push service answers 410 for it
	got      []payload
	raw      [][]byte
}

// fixture is a database with users, a stock, and a fake push service the browsers subscribe to.
type fixture struct {
	t        *testing.T
	db       *sql.DB
	now      time.Time
	users    *user.Store
	stock    *stock.Store
	push     *Store
	notifier *Notifier
	anna     user.User
	session  user.Session

	mu       sync.Mutex
	browsers map[string]*browser
	server   *httptest.Server
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hausbuch.db")
	database, err := db.Open(ctx, path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	f := &fixture{t: t, db: database, now: time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local), browsers: map[string]*browser{}}
	clock := func() time.Time { return f.now }
	f.users = user.NewStore(database)
	f.users.Now = clock
	f.stock = stock.NewStore(database)
	f.stock.Now = clock
	f.push = NewStore(database)
	f.push.Now = clock
	f.server = httptest.NewTLSServer(http.HandlerFunc(f.receive))
	t.Cleanup(f.server.Close)
	f.notifier = &Notifier{Push: f.push, Stock: f.stock, Subscriber: "https://home.example.org",
		Client: f.server.Client()}
	f.anna, err = f.users.Add(ctx, "anna", "Anna", "spoon kettle garden")
	if err != nil {
		t.Fatal(err)
	}
	f.session = f.signIn(f.anna)
	return f
}

func (f *fixture) signIn(u user.User) user.Session {
	f.t.Helper()
	sess, err := f.users.CreateSession(ctx, u)
	if err != nil {
		f.t.Fatal(err)
	}
	return sess
}

// subscribe makes a new browser and subscribes it in the session.
func (f *fixture) subscribe(sess user.Session) *browser {
	f.t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		f.t.Fatal(err)
	}
	b := &browser{key: key, auth: make([]byte, 16)}
	_, _ = rand.Read(b.auth)
	f.mu.Lock()
	b.endpoint = f.server.URL + "/push/" + base64.RawURLEncoding.EncodeToString(b.auth)
	f.browsers[b.endpoint] = b
	f.mu.Unlock()
	var sub Subscription
	sub.Endpoint = b.endpoint
	sub.Keys.P256dh = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	sub.Keys.Auth = base64.RawURLEncoding.EncodeToString(b.auth)
	if err := f.push.Subscribe(ctx, user.SessionHash(sess.Token), sess.User.ID, sub); err != nil {
		f.t.Fatal(err)
	}
	return b
}

func (f *fixture) receive(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	b := f.browsers[f.server.URL+r.URL.Path]
	if b == nil {
		http.NotFound(w, r)
		return
	}
	if b.gone {
		w.WriteHeader(http.StatusGone)
		return
	}
	b.raw = append(b.raw, body)
	plain, err := decrypt(b, body)
	if err != nil {
		f.t.Errorf("decrypt: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var p payload
	if err := json.Unmarshal(plain, &p); err != nil {
		f.t.Errorf("payload %q: %v", plain, err)
	}
	b.got = append(b.got, p)
	w.WriteHeader(http.StatusCreated)
}

// decrypt opens an aes128gcm message (RFC 8188) encrypted for the browser as RFC 8291 says.
func decrypt(b *browser, body []byte) ([]byte, error) {
	salt, idLen := body[:16], int(body[20])
	serverKey, err := ecdh.P256().NewPublicKey(body[21 : 21+idLen])
	if err != nil {
		return nil, err
	}
	shared, err := b.key.ECDH(serverKey)
	if err != nil {
		return nil, err
	}
	info := append(append([]byte("WebPush: info\x00"), b.key.PublicKey().Bytes()...), serverKey.Bytes()...)
	ikm, err := hkdf.Key(sha256.New, shared, b.auth, string(info), 32)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, body[21+idLen:], nil)
	if err != nil {
		return nil, err
	}
	plain = bytes.TrimRight(plain, "\x00")
	return plain[:len(plain)-1], nil // the delimiter of the last record
}

func (f *fixture) at(hour, minute int) {
	y, m, d := f.now.Date()
	f.now = time.Date(y, m, d, hour, minute, 0, 0, time.Local)
}

func (f *fixture) nextDay() { f.now = f.now.AddDate(0, 0, 1) }

func (f *fixture) sendDue() {
	f.t.Helper()
	if err := f.notifier.SendDue(ctx); err != nil {
		f.t.Fatal(err)
	}
}

// add puts a row expiring in days into the section.
func (f *fixture) add(section stock.Section, name string, days int) int64 {
	f.t.Helper()
	id, err := f.stock.Add(ctx, section, f.anna.ID, stock.Values{Name: name, Quantity: stock.One,
		ExpiresOn: stock.Date(f.now).AddDate(0, 0, days)})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func bodies(b *browser) []string {
	var got []string
	for _, p := range b.got {
		got = append(got, p.Body)
	}
	return got
}

func TestKeysSurviveRestart(t *testing.T) {
	f := newFixture(t)
	first, err := f.push.Keys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewStore(f.db).Keys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first != again || first.Public == "" {
		t.Errorf("keys %+v, then %+v", first, again)
	}
}

func TestDefaults(t *testing.T) {
	f := newFixture(t)
	c, err := f.push.Choice(ctx, f.anna.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Minutes != 9*60 || !c.Sections[stock.Medicines] || !c.Sections[stock.Pantry] || !c.Sections[stock.Supplies] {
		t.Errorf("default choice %+v", c)
	}
}

func TestSubscriptionsEndWithTheSession(t *testing.T) {
	cases := map[string]func(f *fixture, laptop user.Session){
		"Sign out": func(f *fixture, _ user.Session) {
			if err := f.users.DeleteSession(ctx, f.session.Token); err != nil {
				t.Fatal(err)
			}
		},
		"Session ended unused": func(f *fixture, laptop user.Session) {
			f.now = f.now.Add(user.DefaultSessionTimeout / 2)
			if _, err := f.users.LookupSession(ctx, laptop.Token); err != nil { // the laptop is used
				t.Fatal(err)
			}
			f.now = f.now.Add(user.DefaultSessionTimeout/2 + time.Hour)
			if err := f.users.DeleteExpiredSessions(ctx); err != nil {
				t.Fatal(err)
			}
		},
		"Disabled account": func(f *fixture, _ user.Session) {
			if err := f.users.Disable(ctx, "anna"); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, end := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			phone := f.subscribe(f.session)
			laptopSession := f.signIn(f.anna)
			laptop := f.subscribe(laptopSession)
			end(f, laptopSession)
			f.add(stock.Pantry, "Yogurt", 1)
			f.at(9, 0)
			f.sendDue()
			if len(phone.got) != 0 {
				t.Errorf("the phone received %v", bodies(phone))
			}
			if name == "Disabled account" && len(laptop.got) != 0 {
				t.Errorf("the laptop received %v", bodies(laptop))
			}
			if name != "Disabled account" && len(laptop.got) != 1 {
				t.Errorf("the laptop, still signed in, received %v", bodies(laptop))
			}
		})
	}
}

func TestUnsubscribeOnlyOwnSession(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	other := f.signIn(f.anna)
	if err := f.push.Unsubscribe(ctx, user.SessionHash(other.Token), phone.endpoint); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.push.Endpoints(ctx, user.SessionHash(f.session.Token)); len(got) != 1 {
		t.Errorf("after another session's unsubscribe, endpoints %v", got)
	}
	if err := f.push.Unsubscribe(ctx, user.SessionHash(f.session.Token), phone.endpoint); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.push.Endpoints(ctx, user.SessionHash(f.session.Token)); len(got) != 0 {
		t.Errorf("after its own unsubscribe, endpoints %v", got)
	}
}

func TestSomethingStartedExpiring(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	f.add(stock.Pantry, "Yogurt", 7)
	f.at(8, 59)
	f.sendDue()
	if len(phone.got) != 0 {
		t.Fatalf("before 09:00, received %v", bodies(phone))
	}
	f.at(9, 0)
	f.sendDue()
	f.sendDue()
	if len(phone.got) != 1 || phone.got[0].Body != "Yogurt (Pantry, expiring)" {
		t.Errorf("received %v", bodies(phone))
	}
}

func TestNothingNew(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	f.add(stock.Pantry, "Yogurt", 3)
	f.at(9, 0)
	f.sendDue()
	f.nextDay()
	f.sendDue()
	if len(phone.got) != 1 {
		t.Errorf("received %v", bodies(phone))
	}
}

func TestExpiringThenExpired(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	f.add(stock.Pantry, "Yogurt", 0)
	f.at(9, 0)
	f.sendDue()
	f.nextDay()
	f.sendDue()
	want := []string{"Yogurt (Pantry, expiring)", "Yogurt (Pantry, expired)"}
	if got := bodies(phone); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("received %v, want %v", got, want)
	}
}

func TestSectionNotChosen(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	c := DefaultChoice()
	c.Sections[stock.Pantry] = false
	if err := f.push.SetChoice(ctx, f.anna.ID, c); err != nil {
		t.Fatal(err)
	}
	f.add(stock.Pantry, "Yogurt", 1)
	f.add(stock.Medicines, "Eye drops", 10)
	f.at(9, 0)
	f.sendDue()
	if len(phone.got) != 1 || phone.got[0].Body != "Eye drops (Medicines, expiring)" {
		t.Errorf("received %v", bodies(phone))
	}
}

func TestNoSectionChosen(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	if err := f.push.SetChoice(ctx, f.anna.ID, Choice{Minutes: 9 * 60}); err != nil {
		t.Fatal(err)
	}
	f.add(stock.Pantry, "Yogurt", 1)
	f.at(9, 0)
	f.sendDue()
	if len(phone.got) != 0 {
		t.Errorf("received %v", bodies(phone))
	}
}

func TestFirstSummary(t *testing.T) {
	f := newFixture(t)
	f.add(stock.Pantry, "Yogurt", -2)
	f.add(stock.Medicines, "Eye drops", 10)
	f.add(stock.Supplies, "Descaler", 3)
	f.at(9, 0)
	f.sendDue() // nobody subscribed yet
	f.at(10, 0)
	phone := f.subscribe(f.session)
	f.sendDue()
	if len(phone.got) != 1 || phone.got[0].Title != "3 things to check" {
		t.Fatalf("received %+v", phone.got)
	}
	want := "Yogurt (Pantry, expired)\nDescaler (Supplies, expiring)\nEye drops (Medicines, expiring)"
	if phone.got[0].Body != want {
		t.Errorf("body %q, want %q", phone.got[0].Body, want)
	}
}

func TestDatePushedBackThenDueAgain(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	id := f.add(stock.Pantry, "Yogurt", 2)
	f.at(9, 0)
	f.sendDue()
	if _, err := f.stock.Update(ctx, stock.Pantry, id, f.anna.ID, stock.Values{Name: "Yogurt", Quantity: stock.One,
		ExpiresOn: stock.Date(f.now).AddDate(0, 0, 30)}, stock.Pantry); err != nil {
		t.Fatal(err)
	}
	f.nextDay()
	f.sendDue()
	f.now = f.now.AddDate(0, 0, 25)
	f.sendDue()
	if len(phone.got) != 2 {
		t.Errorf("received %v", bodies(phone))
	}
}

func TestTwoRows(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	f.add(stock.Medicines, "Eye drops", 10)
	f.add(stock.Pantry, "Yogurt", -1)
	f.at(9, 0)
	f.sendDue()
	if len(phone.got) != 1 {
		t.Fatalf("received %v", bodies(phone))
	}
	if p := phone.got[0]; p.Title != "2 things to check" ||
		p.Body != "Yogurt (Pantry, expired)\nEye drops (Medicines, expiring)" {
		t.Errorf("received %+v", p)
	}
}

func TestManyRows(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	for i := range 40 {
		f.add(stock.Pantry, strings.Repeat("Long product name ", 6)+strconv.Itoa(i), 1)
	}
	f.at(9, 0)
	f.sendDue()
	if len(phone.got) != 1 {
		t.Fatalf("received %d messages", len(phone.got))
	}
	p := phone.got[0]
	lines := strings.Split(p.Body, "\n")
	last := lines[len(lines)-1]
	if p.Title != "40 things to check" || len(p.Body) > maxBody || len(lines) < 2 ||
		last != "and "+strconv.Itoa(40-(len(lines)-1))+" more" {
		t.Errorf("title %q, %d bytes, last line %q", p.Title, len(p.Body), last)
	}
}

func TestSentPayload(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	f.add(stock.Pantry, "Yogurt", 1)
	f.at(9, 0)
	f.sendDue()
	if len(phone.raw) != 1 || len(phone.got) != 1 {
		t.Fatalf("received %d requests", len(phone.raw))
	}
	for _, word := range []string{"Yogurt", "Pantry"} {
		if bytes.Contains(phone.raw[0], []byte(word)) {
			t.Errorf("the request to the push service contains %q", word)
		}
	}
}

func TestSubscriptionGone(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	laptop := f.subscribe(f.session)
	phone.gone = true
	f.add(stock.Pantry, "Yogurt", 1)
	f.at(9, 0)
	f.sendDue()
	if len(laptop.got) != 1 {
		t.Errorf("the laptop received %v", bodies(laptop))
	}
	got, err := f.push.Endpoints(ctx, user.SessionHash(f.session.Token))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != laptop.endpoint {
		t.Errorf("endpoints left %v", got)
	}
}

func TestSummarySentLog(t *testing.T) {
	f := newFixture(t)
	var log bytes.Buffer
	saved := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&log, nil)))
	t.Cleanup(func() { slog.SetDefault(saved) })
	f.subscribe(f.session)
	f.subscribe(f.signIn(f.anna))
	f.add(stock.Pantry, "Yogurt", 1)
	f.add(stock.Medicines, "Eye drops", 3)
	f.at(9, 0)
	f.sendDue()
	out := log.String()
	if !strings.Contains(out, `msg="notification summary sent" user=anna rows=2 devices=2`) {
		t.Errorf("log %q", out)
	}
	for _, leak := range []string{"Yogurt", "Eye drops", f.server.URL} {
		if strings.Contains(out, leak) {
			t.Errorf("log holds %q", leak)
		}
	}
}

func TestChangedChoices(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	laptop := f.subscribe(f.signIn(f.anna))
	if err := f.push.SetChoice(ctx, f.anna.ID, Choice{Minutes: 18*60 + 30,
		Sections: map[stock.Section]bool{stock.Medicines: true}}); err != nil {
		t.Fatal(err)
	}
	f.add(stock.Pantry, "Yogurt", 1)
	f.add(stock.Medicines, "Eye drops", 3)
	f.at(18, 29)
	f.sendDue()
	f.at(18, 30)
	f.sendDue()
	for name, b := range map[string]*browser{"phone": phone, "laptop": laptop} {
		if got := bodies(b); len(got) != 1 || got[0] != "Eye drops (Medicines, expiring)" {
			t.Errorf("the %s received %v", name, got)
		}
	}
}

func TestStartedLate(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	f.add(stock.Pantry, "Yogurt", 1)
	f.at(11, 0)
	f.sendDue()
	f.at(15, 0)
	f.sendDue()
	if len(phone.got) != 1 {
		t.Errorf("received %v", bodies(phone))
	}
}

func TestDayMissed(t *testing.T) {
	f := newFixture(t)
	phone := f.subscribe(f.session)
	f.at(9, 0)
	f.sendDue()
	f.add(stock.Pantry, "Yogurt", 7) // yesterday it became expiring, while the server was down
	f.now = f.now.AddDate(0, 0, 2)
	f.sendDue()
	if got := bodies(phone); len(got) != 1 || got[0] != "Yogurt (Pantry, expiring)" {
		t.Errorf("received %v", got)
	}
}

func TestSubscribeRefusesNonPush(t *testing.T) {
	f := newFixture(t)
	var sub Subscription
	sub.Endpoint = "http://push.example/abc"
	sub.Keys.P256dh, sub.Keys.Auth = "a", "b"
	if err := f.push.Subscribe(ctx, user.SessionHash(f.session.Token), f.anna.ID, sub); err != ErrSubscription {
		t.Errorf("plain HTTP endpoint: %v", err)
	}
}
