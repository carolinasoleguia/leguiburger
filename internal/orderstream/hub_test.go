package orderstream

import (
	"github.com/google/uuid"
	"testing"
)

func TestIsolationCoalescingAndRelease(t *testing.T) {
	h := New()
	if _, _, ok := h.Subscribe("x", "cap"); ok {
		t.Fatal("subscribed offline")
	}
	h.setReady(true)
	a, b := uuid.NewString(), uuid.NewString()
	ca, release, ok := h.Subscribe(a, "a")
	if !ok {
		t.Fatal("subscription failed")
	}
	cb, releaseB, _ := h.Subscribe(b, "b")
	defer releaseB()
	h.publish(a)
	h.publish(a)
	select {
	case <-ca:
	default:
		t.Fatal("missing event")
	}
	select {
	case <-ca:
		t.Fatal("did not coalesce")
	default:
	}
	select {
	case <-cb:
		t.Fatal("cross order notification")
	default:
	}
	release()
	h.publish(a)
	select {
	case <-ca:
		t.Fatal("released subscriber received event")
	default:
	}
	h.setReady(false)
	select {
	case <-cb:
	default:
		t.Fatal("listener loss did not wake streams")
	}
}
func TestCapabilityLimit(t *testing.T) {
	h := New()
	h.setReady(true)
	for i := 0; i < 3; i++ {
		if _, _, ok := h.Subscribe("order", "cap"); !ok {
			t.Fatal("limit too low")
		}
	}
	if _, _, ok := h.Subscribe("order", "cap"); ok {
		t.Fatal("capability limit missing")
	}
}
