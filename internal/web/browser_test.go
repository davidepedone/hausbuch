package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	cdpwebauthn "github.com/chromedp/cdproto/webauthn"
	"github.com/chromedp/chromedp"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// inChrome serves h and runs in a headless Chrome the actions that steps returns for the server's URL, after signing
// in as a user it adds to a.
func inChrome(t *testing.T, a *testApp, h http.Handler, steps func(url string) chromedp.Tasks) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	chromeAt(t, a, srv.URL, steps)
}

// chromeAt is inChrome for a server already running at url.
func chromeAt(t *testing.T, a *testApp, url string, steps func(url string) chromedp.Tasks) {
	t.Helper()
	a.addUser("anna", "Anna", "secret123")
	// Without its sandbox, which Ubuntu 24.04's AppArmor rules keep Chrome from setting up; it only opens the test
	// server's pages. Starting Chrome alone can take more than chromedp's default 20 seconds on a busy CI runner.
	ctx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoSandbox, chromedp.WSURLReadTimeout(time.Minute))...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url+"/login"),
		chromedp.SendKeys("[name=username]", "anna", chromedp.ByQuery),
		chromedp.SendKeys("[name=password]", "secret123\n", chromedp.ByQuery),
		chromedp.WaitVisible("[role=search]", chromedp.ByQuery),
		steps(url),
	); err != nil {
		t.Fatal(err)
	}
}

func TestKnownNameKeepsWhatWasTyped(t *testing.T) {
	a := newApp(t)
	if _, err := a.stock.Add(context.Background(), stock.Pantry, a.addUser("marco", "Marco", "pw").ID,
		stock.Values{Name: "Pasta", Quantity: stock.One, Category: "Long-life", Location: "Cellar"}); err != nil {
		t.Fatal(err)
	}
	// The product's values are held back until the location has been typed, as on a slow connection.
	asked, answer := make(chan struct{}, 1), make(chan struct{})
	h := NewHandler(a.users, a.pages, a.stock, "dev", nil)
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pantry/product" {
			asked <- struct{}{}
			<-answer
		}
		h.ServeHTTP(w, r)
	})
	const form = `form[action="/pantry/items"] `
	var location, focused string
	inChrome(t, a, slow, func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.Navigate(url + "/pantry"),
			chromedp.Click("summary", chromedp.ByQuery),
			chromedp.SendKeys(form+"[name=name]", "Pasta", chromedp.ByQuery),
			chromedp.Click(form+"[name=location]", chromedp.ByQuery),
			chromedp.ActionFunc(func(context.Context) error { <-asked; return nil }),
			chromedp.SendKeys(form+"[name=location]", "Kitchen", chromedp.ByQuery),
			chromedp.ActionFunc(func(context.Context) error { close(answer); return nil }),
			// The category, left alone, gets the product's.
			chromedp.Poll(`document.querySelector('`+form+`[name=category]').value === "Long-life"`, nil),
			chromedp.Value(form+"[name=location]", &location, chromedp.ByQuery),
			chromedp.Evaluate(`document.activeElement.name || document.activeElement.tagName`, &focused),
		}
	})
	if location != "Kitchen" || focused != "location" {
		t.Errorf("location %q, focus on %q", location, focused)
	}
}

func TestChooseACategoryUnderTheField(t *testing.T) {
	a := newApp(t)
	const field = `form[action="/supplies/items"] [name=category]`
	var chosen, typed string
	inChrome(t, a, NewHandler(a.users, a.pages, a.stock, "dev", nil), func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			// An empty section opens its add form by itself.
			chromedp.Navigate(url + "/supplies"),
			chromedp.Click(`//button[@data-suggestion][text()="Laundry"]`, chromedp.BySearch),
			chromedp.Value(field, &chosen, chromedp.ByQuery),
			chromedp.SendKeys(field, " room", chromedp.ByQuery),
			chromedp.Value(field, &typed, chromedp.ByQuery),
		}
	})
	if chosen != "Laundry" || typed != "Laundry room" {
		t.Errorf("chosen %q, then typed %q", chosen, typed)
	}
}

func TestChooseALocationUnderTheField(t *testing.T) {
	a := newApp(t)
	for _, location := range []string{"Cellar", "Fridge"} {
		if _, err := a.stock.Add(context.Background(), stock.Pantry, a.addUser("u"+location, location, "pw").ID,
			stock.Values{Name: location + " thing", Quantity: stock.One, Location: location}); err != nil {
			t.Fatal(err)
		}
	}
	const field = `form[action="/medicines/items"] [name=location]`
	var chosen, typed string
	inChrome(t, a, NewHandler(a.users, a.pages, a.stock, "dev", nil), func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			// An empty section opens its add form by itself.
			chromedp.Navigate(url + "/medicines"),
			chromedp.Click(`//input[@name="location"]/ancestor::div[@data-suggestions]//button[text()="Cellar"]`, chromedp.BySearch),
			chromedp.Value(field, &chosen, chromedp.ByQuery),
			chromedp.SendKeys(field, ", top shelf", chromedp.ByQuery),
			chromedp.Value(field, &typed, chromedp.ByQuery),
		}
	})
	if chosen != "Cellar" || typed != "Cellar, top shelf" {
		t.Errorf("chosen %q, then typed %q", chosen, typed)
	}
}

func TestShoppingListInTheBrowser(t *testing.T) {
	a := newApp(t)
	marco := a.addUser("marco", "Marco", "pw")
	ctx := context.Background()
	for _, e := range []stock.Entry{
		{Name: "Paracetamol", Section: stock.Medicines, Shop: "Pharmacy"},
		{Name: "Butter", Section: stock.Pantry, Shop: "Migros"},
		{Name: "Bread"},
	} {
		if err := a.stock.AddEntry(ctx, marco.ID, e); err != nil {
			t.Fatal(err)
		}
	}
	const shown = `[...document.querySelectorAll("#entries [data-entry]:not([hidden])")].map((e) => e.querySelector(".font-medium").textContent)`
	const butterBought = `[...document.querySelectorAll("#entries section")].some((s) =>
		s.querySelector("h2").textContent === "Bought" && s.textContent.includes("Butter"))`
	var atThePharmacy, afterRefresh []string
	var typed, pressed string
	inChrome(t, a, NewHandler(a.users, a.pages, a.stock, "dev", nil), func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.Navigate(url + "/shopping"),
			chromedp.Click(`[data-filter-chip="pharmacy"]`, chromedp.ByQuery),
			chromedp.Evaluate(shown, &atThePharmacy),
			chromedp.Click("summary", chromedp.ByQuery),
			chromedp.SendKeys(`form[action="/shopping/entries"] [name=name]`, "Ric", chromedp.ByQuery),
			// Bought by someone else while the list is open.
			chromedp.ActionFunc(func(context.Context) error {
				return a.stock.Tick(ctx, a.entry("Butter").ID, true)
			}),
			chromedp.Poll(butterBought, nil, chromedp.WithPollingTimeout(15*time.Second)),
			chromedp.Evaluate(shown, &afterRefresh),
			chromedp.Value(`form[action="/shopping/entries"] [name=name]`, &typed, chromedp.ByQuery),
			chromedp.AttributeValue(`[data-filter-chip="pharmacy"]`, "aria-pressed", &pressed, nil, chromedp.ByQuery),
		}
	})
	want := []string{"Paracetamol", "Bread"}
	if !slices.Equal(atThePharmacy, want) || !slices.Equal(afterRefresh, want) {
		t.Errorf("at the pharmacy %v, after the refresh %v, want %v", atThePharmacy, afterRefresh, want)
	}
	if typed != "Ric" || pressed != "true" {
		t.Errorf("after the refresh: typed %q, pharmacy chip pressed %q", typed, pressed)
	}
}

func TestPasskeysInTheBrowser(t *testing.T) {
	a := newAppWithPasskeys(t)
	ctx := context.Background()
	var authenticator cdpwebauthn.AuthenticatorID
	var afterOffer, fromSuggestions, afterSignIn, whenLocked, whenDisabled, whenCancelled string
	var listed, afterRemoving, afterAdding int
	const passkeyButton = `[data-passkey-button]`
	// loaded waits for passkeys.js, deferred, to be listening on the buttons.
	loaded := chromedp.Poll(`document.readyState === "complete"`, nil)
	signOut := chromedp.Tasks{
		chromedp.Click(`form[action="/logout"] button`, chromedp.ByQuery),
		chromedp.WaitVisible(`[data-passkey-sign-in] `+passkeyButton, chromedp.ByQuery),
		loaded,
	}
	passkeys := func(n *int) chromedp.Action {
		return chromedp.Evaluate(`document.querySelectorAll('form[action="/account/passkeys/remove"]').length`, n)
	}
	errorText := func(text *string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.WaitVisible(`[data-passkey-error]`, chromedp.ByQuery),
			chromedp.Text(`[data-passkey-error]`, text, chromedp.ByQuery),
		}
	}
	act := func(f func() error) chromedp.Action {
		return chromedp.ActionFunc(func(context.Context) error { return f() })
	}
	chromeAt(t, a, a.srv.URL, func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			cdpwebauthn.Enable(),
			chromedp.ActionFunc(func(ctx context.Context) (err error) {
				authenticator, err = cdpwebauthn.AddVirtualAuthenticator(&cdpwebauthn.VirtualAuthenticatorOptions{
					Protocol: cdpwebauthn.AuthenticatorProtocolCtap2, Ctap2version: cdpwebauthn.Ctap2versionCtap21,
					Transport: cdpwebauthn.AuthenticatorTransportInternal, HasResidentKey: true, HasUserVerification: true,
					IsUserVerified: true, AutomaticPresenceSimulation: true}).Do(ctx)
				return err
			}),
			// Offer after the password.
			loaded,
			chromedp.Click(`[data-passkey-create] `+passkeyButton, chromedp.ByQuery),
			chromedp.WaitVisible(`a[href="/pages/new"]`, chromedp.ByQuery),
			chromedp.Location(&afterOffer),

			// The username field's suggestions: the virtual authenticator picks the passkey by itself.
			chromedp.Click(`form[action="/logout"] button`, chromedp.ByQuery),
			chromedp.WaitVisible(`a[href="/pages/new"]`, chromedp.ByQuery),
			chromedp.Location(&fromSuggestions),
			// Without suggestions from here on, so that the login page waits for the button.
			chromedp.ActionFunc(func(ctx context.Context) error {
				_, err := page.AddScriptToEvaluateOnNewDocument(
					`PublicKeyCredential.isConditionalMediationAvailable = async () => false`).Do(ctx)
				return err
			}),

			// Passkey sign-in.
			signOut,
			chromedp.Click(`[data-passkey-sign-in] `+passkeyButton, chromedp.ByQuery),
			chromedp.WaitVisible("[role=search]", chromedp.ByQuery),
			chromedp.Location(&afterSignIn),

			// Locked username.
			signOut,
			act(func() error {
				for range 3 {
					_, _ = a.users.Authenticate(ctx, "anna", "wrong")
				}
				return nil
			}),
			chromedp.Click(`[data-passkey-sign-in] `+passkeyButton, chromedp.ByQuery),
			chromedp.WaitVisible("[role=search]", chromedp.ByQuery),
			chromedp.Location(&whenLocked),

			// Remove a passkey, add a passkey.
			chromedp.Navigate(url + "/account/password"),
			passkeys(&listed),
			chromedp.Click(`form[action="/account/passkeys/remove"] button`, chromedp.ByQuery),
			act(func() error {
				// The removal ends in a reload; the database tells when it is done. Anna, the only account, has id 1.
				for range 50 {
					if keys, err := a.users.Passkeys(ctx, 1); err != nil || len(keys) == 0 {
						return err
					}
					time.Sleep(100 * time.Millisecond)
				}
				return nil
			}),
			chromedp.Navigate(url + "/account/password"),
			loaded,
			passkeys(&afterRemoving),
			// A new device's authenticator, without the passkey removed from the account.
			chromedp.ActionFunc(func(ctx context.Context) error {
				return cdpwebauthn.ClearCredentials(authenticator).Do(ctx)
			}),
			chromedp.Click(`[data-passkey-create] `+passkeyButton, chromedp.ByQuery),
			chromedp.WaitVisible(`form[action="/account/passkeys/remove"]`, chromedp.ByQuery),
			passkeys(&afterAdding),

			// Disabled account.
			signOut,
			act(func() error { return a.users.Disable(ctx, "anna") }),
			chromedp.Click(`[data-passkey-sign-in] `+passkeyButton, chromedp.ByQuery),
			errorText(&whenDisabled),

			// Passkey cancelled: the authenticator has none.
			act(func() error { return a.users.Enable(ctx, "anna") }),
			chromedp.Reload(),
			loaded,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return cdpwebauthn.ClearCredentials(authenticator).Do(ctx)
			}),
			chromedp.Click(`[data-passkey-sign-in] `+passkeyButton, chromedp.ByQuery),
			errorText(&whenCancelled),
		}
	})
	for _, got := range []string{afterOffer, fromSuggestions, afterSignIn, whenLocked} {
		if got != a.srv.URL+"/" {
			t.Errorf("after the offer %s, from the suggestions %s, after the passkey sign-in %s, when locked %s",
				afterOffer, fromSuggestions, afterSignIn, whenLocked)
			break
		}
	}
	if listed != 1 || afterRemoving != 0 || afterAdding != 1 {
		t.Errorf("passkeys listed %d, after removing %d, after adding %d", listed, afterRemoving, afterAdding)
	}
	const failed = "The passkey sign-in didn't complete. Try again, or sign in with your password."
	if whenDisabled != failed || whenCancelled != failed {
		t.Errorf("disabled: %q, cancelled: %q", whenDisabled, whenCancelled)
	}
}
