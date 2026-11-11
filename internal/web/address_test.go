package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func addressOf(t *testing.T, proxies, remote string, forwarded ...string) string {
	t.Helper()
	p, err := ParseTrustedProxies(proxies)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	r.RemoteAddr = remote
	for _, f := range forwarded {
		r.Header.Add("X-Forwarded-For", f)
	}
	return (&server{proxies: p}).address(r)
}

func TestHeaderFromAnUnknownSenderIsIgnored(t *testing.T) {
	if got := addressOf(t, "", "203.0.113.5:4000", "198.51.100.7"); got != "203.0.113.5" {
		t.Errorf("no trusted proxies: %s, want the connection's address", got)
	}
	if got := addressOf(t, "10.0.0.0/8", "203.0.113.5:4000", "198.51.100.7"); got != "203.0.113.5" {
		t.Errorf("a sender that isn't a listed proxy: %s, want the connection's address", got)
	}
}

func TestAddressBehindAListedProxy(t *testing.T) {
	for _, c := range []struct {
		name, proxies, remote string
		forwarded             []string
		want                  string
	}{
		{"one proxy", "172.16.0.0/12", "172.18.0.3:4000", []string{"198.51.100.7"}, "198.51.100.7"},
		// The client can put anything in front: only what the trusted proxies appended counts.
		{"forged entries before", "172.16.0.0/12", "172.18.0.3:4000", []string{"192.0.2.99, 198.51.100.7"}, "198.51.100.7"},
		{"two proxies", "172.16.0.0/12, 10.0.0.1", "172.18.0.3:4000", []string{"198.51.100.7, 10.0.0.1"}, "198.51.100.7"},
		{"headers repeated", "172.16.0.0/12", "172.18.0.3:4000", []string{"192.0.2.99", "198.51.100.7"}, "198.51.100.7"},
		{"no header", "172.16.0.0/12", "172.18.0.3:4000", nil, "172.18.0.3"},
		{"malformed header", "172.16.0.0/12", "172.18.0.3:4000", []string{"not-an-address"}, "172.18.0.3"},
		{"IPv6 proxy", "::1", "[::1]:4000", []string{"2001:db8:1:2:3:4:5:6"}, "2001:db8:1:2::/64"},
	} {
		if got := addressOf(t, c.proxies, c.remote, c.forwarded...); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

func TestAddressKeys(t *testing.T) {
	for remote, want := range map[string]string{
		"192.0.2.1:4000":              "192.0.2.1",
		"[2001:db8:1:2:3:4:5:6]:4000": "2001:db8:1:2::/64",
		"[2001:db8:1:2:ffff::1]:4000": "2001:db8:1:2::/64", // the same /64
		"[::ffff:192.0.2.1]:4000":     "192.0.2.1",         // IPv4-mapped
		"[fe80::1%en0]:4000":          "fe80::/64",
		"[2001:db8:1:3:3:4:5:6]:4000": "2001:db8:1:3::/64",
	} {
		if got := addressOf(t, "", remote); got != want {
			t.Errorf("%s: %s, want %s", remote, got, want)
		}
	}
}

func TestParseTrustedProxies(t *testing.T) {
	p, err := ParseTrustedProxies(" 172.16.0.0/12 ,10.0.0.1,, ::1 ")
	if err != nil || len(p) != 3 {
		t.Fatalf("valid list: %v, %v", p, err)
	}
	for _, bad := range []string{"proxy", "10.0.0.0/33", "10.0.0.1:80"} {
		if _, err := ParseTrustedProxies(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
