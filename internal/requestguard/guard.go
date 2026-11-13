// Package requestguard rejects bursts before checkout work reaches PostgreSQL.
package requestguard

import (
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

// Render terminates public traffic through its Cloudflare edge. Trust its
// overwritten visitor header only inside Render, never on a direct/local server.
func ClientIP(r *http.Request) string {
	if os.Getenv("RENDER") == "true" {
		if ip, e := netip.ParseAddr(r.Header.Get("CF-Connecting-IP")); e == nil {
			return ip.Unmap().String()
		}
	}
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		host = r.RemoteAddr
	}
	if ip, e := netip.ParseAddr(strings.Trim(host, "[]")); e == nil {
		return ip.Unmap().String()
	}
	return "unknown"
}

type entry struct {
	count        int
	start, until time.Time
}
type Guard struct {
	mu      sync.Mutex
	entries map[string]entry
}

func New() *Guard { return &Guard{entries: make(map[string]entry)} }

// A fixed one-minute burst window creates a five-minute cooling-off period.
// Blocked requests do not extend the period. State is bounded and expires.
func (g *Guard) Allow(key string, limit int, now time.Time) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	e, exists := g.entries[key]
	if now.Before(e.until) {
		return e.until.Sub(now)
	}
	if !exists && len(g.entries) >= 10000 {
		for k, v := range g.entries {
			if !now.Before(v.until) && now.Sub(v.start) >= time.Minute {
				delete(g.entries, k)
			}
		}
		if len(g.entries) >= 10000 {
			return time.Minute
		}
	}
	if now.Sub(e.start) >= time.Minute {
		e = entry{start: now}
	}
	e.count++
	if e.count > limit {
		e.until = now.Add(5 * time.Minute)
	}
	g.entries[key] = e
	if now.Before(e.until) {
		return e.until.Sub(now)
	}
	return 0
}
