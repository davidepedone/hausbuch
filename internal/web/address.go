package web

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

var errTrustedProxies = errors.New("HAUSBUCH_TRUSTED_PROXIES must list addresses or ranges such as 172.16.0.0/12, separated by commas")

// ParseTrustedProxies reads the proxies whose X-Forwarded-For header is believed: addresses or CIDR ranges separated
// by commas.
func ParseTrustedProxies(list string) ([]netip.Prefix, error) {
	var proxies []netip.Prefix
	for entry := range strings.SplitSeq(list, ",") {
		if entry = strings.TrimSpace(entry); entry == "" {
			continue
		}
		p, err := netip.ParsePrefix(entry)
		if err != nil {
			a, err := netip.ParseAddr(entry)
			if err != nil {
				return nil, errTrustedProxies
			}
			p = netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen())
		}
		proxies = append(proxies, p.Masked())
	}
	return proxies, nil
}

func (s *server) trusted(a netip.Addr) bool {
	return slices.ContainsFunc(s.proxies, func(p netip.Prefix) bool { return p.Contains(a) })
}

// address is the key of the request's client address, shared by the request limit, the sign-in lock and the log. It
// is the connection's address, unless that is a trusted proxy: then it is the nearest address in X-Forwarded-For
// that isn't one, since only the entries added by trusted proxies can be believed.
func (s *server) address(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	a, err := netip.ParseAddr(host)
	if err != nil {
		return r.RemoteAddr
	}
	a = a.Unmap()
	if s.trusted(a) {
		forwarded := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
		for i := len(forwarded) - 1; i >= 0 && s.trusted(a); i-- {
			f, err := netip.ParseAddr(strings.TrimSpace(forwarded[i]))
			if err != nil {
				break
			}
			a = f.Unmap()
		}
	}
	return addressKey(a)
}

// addressKey is the IPv4 address, or the /64 an IPv6 address belongs to, which a single home or server usually gets
// whole.
func addressKey(a netip.Addr) string {
	if a.Is4() {
		return a.String()
	}
	p, _ := a.WithZone("").Prefix(64)
	return p.String()
}
