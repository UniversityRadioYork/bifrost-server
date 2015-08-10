package pool

import (
	"testing"
	"time"

	"github.com/UniversityRadioYork/baps3-go"
	"gopkg.in/tomb.v2"
)

// TestOhai ensures that new clients are sent an OHAI message.
func TestOhai(t *testing.T) {
	bcast := make(chan *baps3.Message)
	ucast := make(chan *baps3.Message)
	dcon := make(chan struct{})
	quit := make(chan struct{})

	cli := Client{Broadcast: bcast, Unicast: ucast, Disconnect: dcon}

	pool := New("demo", quit)
	var tomb tomb.Tomb
	tomb.Go(func() error { return pool.Run(&tomb) })

	reply := make(chan bool)

	change := NewAddChange(&cli, reply)
	pool.Changes <- change
	select {
	case rp := <-reply:
		if !rp {
			t.Fatal("pool rejected our client")
		}
		t.Log("got add-change reply")
	case <-time.After(time.Second * 5):
		t.Fatal("pool appears to have locked")
	}

	select {
	case msg := <-bcast:
		t.Fatalf("Got unexpected broadcast %q", msg.String())
	case msg := <-ucast:
		if w := msg.Word(); w != baps3.RsOhai {
			t.Fatalf("expected OHAI, got %q", w.String())
		}
		if args := msg.Args(); len(args) != 1 {
			t.Fatalf("expected 1 arg, got %q", args)
		}
		t.Logf("got OHAI: %q", msg)
	case <-time.After(time.Second * 5):
		t.Fatal("OHAI appears to have locked")
	}

	tomb.Kill(nil)

	select {
	case <-dcon:
		t.Log("disconnect signalled")
	case <-time.After(time.Second * 5):
		t.Fatal("pool doesn't appear to be sending disconnect")
	}

	rchange := NewRemoveChange(&cli, reply)
	pool.Changes <- rchange
	select {
	case rp := <-reply:
		if !rp {
			t.Fatal("pool rejected our client disconnect")
		}
		t.Log("got remove-change reply")
	case <-time.After(time.Second * 5):
		t.Fatal("pool disconnect appears to have locked")
	}

	select {
	case <-tomb.Dead():
		t.Log("pool now dead")
	case <-time.After(time.Second * 5):
		t.Fatal("pool doesn't appear to be dying")
	}
}
