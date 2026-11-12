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

func TestTenantAndBuyerTopicsStaySeparate(t *testing.T) {
	h := New()
	h.setReady(true)
	a, b := uuid.NewString(), uuid.NewString()
	buyer, rb, _ := h.Subscribe(a, "buyer")
	defer rb()
	staff, rs, _ := h.Subscribe("tenant:"+a, "staff")
	defer rs()
	foreign, rf, _ := h.Subscribe("tenant:"+b, "other")
	defer rf()
	h.publish("tenant:" + a)
	select {
	case <-staff:
	default:
		t.Fatal("staff invalidation missing")
	}
	select {
	case <-buyer:
		t.Fatal("tenant event reached buyer topic")
	default:
	}
	select {
	case <-foreign:
		t.Fatal("event reached other tenant")
	default:
	}
	h.publish("tenant:invalid")
	select {
	case <-staff:
		t.Fatal("invalid topic accepted")
	default:
	}
}
