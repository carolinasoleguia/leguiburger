// Package orderstream distributes committed invalidations, never customer data.
package orderstream

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"strings"
	"sync"
	"time"
)

type Hub struct {
	mu            sync.Mutex
	ready         bool
	next          int
	subscriptions map[int]*subscription
}
type subscription struct {
	order, cap string
	changed    chan struct{}
}

func New() *Hub            { return &Hub{subscriptions: map[int]*subscription{}} }
func (h *Hub) Ready() bool { h.mu.Lock(); defer h.mu.Unlock(); return h.ready }
func (h *Hub) setReady(v bool) {
	h.mu.Lock()
	h.ready = v
	for _, s := range h.subscriptions {
		select {
		case s.changed <- struct{}{}:
		default:
		}
	}
	h.mu.Unlock()
}
func (h *Hub) Subscribe(order, cap string) (<-chan struct{}, func(), bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, s := range h.subscriptions {
		if s.cap == cap {
			n++
		}
	}
	if !h.ready || len(h.subscriptions) >= 1024 || n >= 3 {
		return nil, func() {}, false
	}
	h.next++
	id := h.next
	s := &subscription{order: order, cap: cap, changed: make(chan struct{}, 1)}
	h.subscriptions[id] = s
	return s.changed, func() { h.mu.Lock(); delete(h.subscriptions, id); h.mu.Unlock() }, true
}
func (h *Hub) publish(order string) {
	if _, e := uuid.Parse(strings.TrimPrefix(order, "tenant:")); e != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.subscriptions {
		if s.order == order {
			select {
			case s.changed <- struct{}{}:
			default:
			}
		}
	}
}

// A single dedicated session per API instance, separate from the request pool.
// Supabase's transaction pool (6543) cannot LISTEN; its session pool uses 5432.
func (h *Hub) Run(ctx context.Context, dsn string) {
	for ctx.Err() == nil {
		h.listen(ctx, dsn)
		h.setReady(false)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}
func (h *Hub) listen(ctx context.Context, dsn string) {
	cfg, e := pgx.ParseConfig(dsn)
	if e != nil {
		return
	}
	if strings.HasSuffix(cfg.Host, ".pooler.supabase.com") && cfg.Port == 6543 {
		cfg.Port = 5432
	}
	cfg.ConnectTimeout = 10 * time.Second
	conn, e := pgx.ConnectConfig(ctx, cfg)
	if e != nil {
		return
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = conn.Close(c)
	}()
	if _, e = conn.Exec(ctx, "LISTEN legui_order_changed"); e != nil {
		return
	}
	h.setReady(true)
	for ctx.Err() == nil {
		wait, cancel := context.WithTimeout(ctx, 30*time.Second)
		n, e := conn.WaitForNotification(wait)
		cancel()
		if e != nil {
			if ctx.Err() != nil {
				return
			}
			if wait.Err() == context.DeadlineExceeded {
				ping, c := context.WithTimeout(ctx, 5*time.Second)
				e = conn.Ping(ping)
				c()
				if e == nil {
					continue
				}
			}
			return
		}
		h.publish(n.Payload)
	}
}
