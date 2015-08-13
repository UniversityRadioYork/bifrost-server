package request

import (
	"testing"

	"github.com/UniversityRadioYork/baps3-go"
)

func failIfNotAck(t *testing.T, msg *baps3.Message) {
}

// TestAckUnknown tests whether the correct ACK is sent on failure.
func TestAckUnknown(t *testing.T) {
	cast := make(chan *baps3.Message)

	router := NewRouter(
		map[baps3.MessageWord]Handler{
		// Intentionally left blank
		},
		cast,
		"",
	)

	rs := make(chan *baps3.Message, 1)
	rq := NewRequest([]string{"xyxxy", "1", "noseybonk"}, baps3.NewMessage(baps3.RqUnknown).AddArg("1").AddArg("noseybonk"), rs)

	router.Dispatch(rq)

	select {
	case msg := <-rs:
		failIfNotAck(t, msg)
		args := msg.Args()
		if len(args) != 5 || args[0] != "WHAT" || args[1] != `unknown request "xyxxy"` || args[2] != "xyxxy" || args[3] != "1" || args[4] != "noseybonk" {
			packed, err := msg.Pack()
			if err != nil {
				t.Fatalf("message packer failed")
			}
			t.Fatalf(`expected "ACK WHAT 'unknown request "xyxxy"' xyxxy 1 noseybonk, got: %s`, packed)
		}
	default:
		t.Fatal("no response sent")
	}
}

// TestACKSuccess tests whether the correct ACK is sent on success.
func TestAckSuccess(t *testing.T) {
	cast := make(chan *baps3.Message)

	router := NewRouter(
		map[baps3.MessageWord]Handler{
			baps3.RqRead: func(_, _ chan<- *baps3.Message, _ []string, _ interface{}) (bool, error) {
				return false, nil
			},
		},
		cast,
		"",
	)

	rs := make(chan *baps3.Message, 1)
	rq := NewRequest([]string{"read", "1", "/foo/bar"}, baps3.NewMessage(baps3.RqRead).AddArg("1").AddArg("/foo/bar"), rs)

	router.Dispatch(rq)

	select {
	case msg := <-rs:
		failIfNotAck(t, msg)
		args := msg.Args()
		if len(args) != 5 || args[0] != "OK" || args[1] != "Success" || args[2] != "read" || args[3] != "1" || args[4] != "/foo/bar" {
			t.Fatalf(`expected "ACK OK Success read 1 /foo/bar", got %q`, msg.String())
		}
	default:
		t.Fatal("no response sent")
	}
}
