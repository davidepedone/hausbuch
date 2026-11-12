package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// postJSON sends body as JSON the way a page's script does.
func (b *browser) postJSON(path, body string) response {
	b.app.t.Helper()
	req, err := http.NewRequest(http.MethodPost, b.app.srv.URL+path, strings.NewReader(body))
	if err != nil {
		b.app.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return b.do(req)
}

const subscription = `{"endpoint":"https://push.example/abc","keys":{"p256dh":"BPk","auth":"xyz"}}`

func TestServiceWorkerNeedsNoSession(t *testing.T) {
	r := newApp(t).browser().get("/sw.js")
	if r.StatusCode != http.StatusOK || !strings.Contains(r.Header.Get("Content-Type"), "javascript") ||
		!strings.Contains(r.body, "showNotification") {
		t.Errorf("/sw.js: %s, %q", r.Status, r.Header.Get("Content-Type"))
	}
}

func TestNotificationsPlainHTTPInstallation(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	if body := b.view("/account/password"); strings.Contains(strings.ToLower(body), "notification") {
		t.Errorf("notifications mentioned:\n%s", body)
	}
	if r := b.postJSON("/account/notifications/subscribe", subscription); r.StatusCode != http.StatusNotFound {
		t.Errorf("subscribe: %s", r.Status)
	}
}

func TestNotificationsHTTPSInstallation(t *testing.T) {
	a := newAppWithPasskeys(t)
	body := a.signedInAs("anna", "Anna").view("/account/password")
	mustContain(t, body, `<h2 id="notifications"`, `data-endpoints="[]"`, `name="time" value="09:00"`,
		`value="medicines" checked`, `value="pantry" checked`, `value="supplies" checked`)
	if strings.Contains(body, `data-key=""`) {
		t.Error("no VAPID key on the page")
	}
}

func TestSaveNotificationChoices(t *testing.T) {
	a := newAppWithPasskeys(t)
	b := a.signedInAs("anna", "Anna")
	r := b.post("/account/notifications", url.Values{"time": {"18:30"}, "section": {"medicines"}})
	if r.StatusCode != http.StatusOK {
		t.Fatal(r.Status)
	}
	mustContain(t, r.body, "Notification choices saved")
	body := b.view("/account/password")
	mustContain(t, body, `name="time" value="18:30"`, `value="medicines" checked`)
	if strings.Contains(body, `value="pantry" checked`) || strings.Contains(body, `value="supplies" checked`) {
		t.Error("unchecked sections shown checked")
	}
	if r := b.post("/account/notifications", url.Values{"time": {"25:00"}}); r.StatusCode != http.StatusBadRequest ||
		!strings.Contains(r.body, "such as 09:00") {
		t.Errorf("bad time: %s", r.Status)
	}
}

func TestSubscribeAndUnsubscribe(t *testing.T) {
	a := newAppWithPasskeys(t)
	phone := a.signedInAs("anna", "Anna")
	if r := phone.postJSON("/account/notifications/subscribe", subscription); r.StatusCode != http.StatusNoContent {
		t.Fatal(r.Status)
	}
	mustContain(t, phone.view("/account/password"), `data-endpoints="[&#34;https://push.example/abc&#34;]"`)

	laptop := a.browser()
	laptop.signIn("anna", "spoon kettle garden")
	mustContain(t, laptop.view("/account/password"), `data-endpoints="[]"`)
	if r := laptop.postJSON("/account/notifications/unsubscribe", `{"endpoint":"https://push.example/abc"}`); r.StatusCode != http.StatusNoContent {
		t.Fatal(r.Status)
	}
	mustContain(t, phone.view("/account/password"), `https://push.example/abc`)

	if r := phone.postJSON("/account/notifications/unsubscribe", `{"endpoint":"https://push.example/abc"}`); r.StatusCode != http.StatusNoContent {
		t.Fatal(r.Status)
	}
	mustContain(t, phone.view("/account/password"), `data-endpoints="[]"`)

	for _, bad := range []string{"", "{", `{"endpoint":"http://push.example/abc","keys":{"p256dh":"a","auth":"b"}}`} {
		if r := phone.postJSON("/account/notifications/subscribe", bad); r.StatusCode != http.StatusBadRequest {
			t.Errorf("subscribe %q: %s", bad, r.Status)
		}
	}
}

func TestServiceWorkerRegisters(t *testing.T) {
	a := newAppWithPasskeys(t)
	var scope string
	chromeAt(t, a, a.srv.URL, func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.Navigate(url + "/account/password"),
			chromedp.WaitVisible("[data-push-button]", chromedp.ByQuery),
			chromedp.Evaluate(`navigator.serviceWorker.ready.then((r) => r.scope)`, &scope,
				func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
		}
	})
	if scope != a.srv.URL+"/" {
		t.Errorf("scope %q", scope)
	}
}

func TestSafariTabOnIPhone(t *testing.T) {
	a := newAppWithPasskeys(t)
	var unsupported, button bool
	chromeAt(t, a, a.srv.URL, func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.ActionFunc(func(ctx context.Context) error {
				// What Safari has outside a Home Screen app: no PushManager.
				_, err := page.AddScriptToEvaluateOnNewDocument(`delete window.PushManager`).Do(ctx)
				return err
			}),
			chromedp.Navigate(url + "/account/password"),
			chromedp.WaitVisible("[data-push-unsupported]", chromedp.ByQuery),
			chromedp.Evaluate(`!document.querySelector('[data-push-unsupported]').hidden`, &unsupported),
			chromedp.Evaluate(`!document.querySelector('[data-push-button]').hidden`, &button),
		}
	})
	if !unsupported || button {
		t.Errorf("Home Screen explanation shown %v, button shown %v", unsupported, button)
	}
}

func TestPermissionRefused(t *testing.T) {
	a := newAppWithPasskeys(t)
	var errorText, status string
	chromeAt(t, a, a.srv.URL, func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.ActionFunc(func(ctx context.Context) error {
				return cdpbrowser.SetPermission(&cdpbrowser.PermissionDescriptor{Name: "notifications"},
					cdpbrowser.PermissionSettingDenied).WithOrigin(url).Do(ctx)
			}),
			chromedp.Navigate(url + "/account/password"),
			chromedp.WaitVisible("[data-push-button]", chromedp.ByQuery),
			chromedp.Click("[data-push-button]", chromedp.ByQuery),
			chromedp.WaitVisible("[data-push-error]", chromedp.ByQuery),
			chromedp.Text("[data-push-error]", &errorText, chromedp.ByQuery),
			chromedp.Text("[data-push-status]", &status, chromedp.ByQuery),
		}
	})
	if !strings.Contains(errorText, "didn't allow notifications") || status != "Notifications are off for this device." {
		t.Errorf("error %q, status %q", errorText, status)
	}
}
