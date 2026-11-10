package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// inChrome serves h and runs in a headless Chrome the actions that steps returns for the server's URL, after signing
// in as a user it adds to a.
func inChrome(t *testing.T, a *testApp, h http.Handler, steps func(url string) chromedp.Tasks) {
	t.Helper()
	a.addUser("anna", "Anna", "secret123")
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	// Without its sandbox, which Ubuntu 24.04's AppArmor rules keep Chrome from setting up; it only opens the test
	// server's pages.
	ctx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoSandbox)...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()
	// Starting Chrome alone can take most of 20 seconds on a busy CI runner.
	ctx, cancel = context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/login"),
		chromedp.SendKeys("[name=username]", "anna", chromedp.ByQuery),
		chromedp.SendKeys("[name=password]", "secret123\n", chromedp.ByQuery),
		chromedp.WaitVisible("[role=search]", chromedp.ByQuery),
		steps(srv.URL),
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
	h := NewHandler(a.users, a.pages, a.stock, "dev")
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
	inChrome(t, a, NewHandler(a.users, a.pages, a.stock, "dev"), func(url string) chromedp.Tasks {
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
	inChrome(t, a, NewHandler(a.users, a.pages, a.stock, "dev"), func(url string) chromedp.Tasks {
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
