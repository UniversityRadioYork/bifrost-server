package request

import (
	"testing"

	"github.com/UniversityRadioYork/baps3-go"
)

// TestACKSuccess tests whether the correct ACK is sent on success.
func TestAckSuccess(t *testing.T) {
	cast := make(chan *baps3.Message)

	router := NewRouter(
		map[baps3.MessageWord]Handler{
			baps3.RqRead: func(_, _ chan<- *baps3.Message, _ []string) (bool, error) {
				return false, nil
			},
		},
		cast,
	)

	rs := make(chan *baps3.Message, 1)
	rq := NewRequest(baps3.NewMessage(baps3.RqRead).AddArg("1").AddArg("/foo/bar"), rs)

	router.Dispatch(rq)

	select {
	case msg := <-rs:
		if word := msg.Word(); word != baps3.RsAck {
			t.Fatalf("expected ACK, got %q", word.String())
		}
		args := msg.Args()
		if len(args) != 5 || args[0] != "OK" || args[1] != "Success" || args[2] != "read" || args[3] != "1" || args[4] != "/foo/bar" {
			t.Fatalf(`expected "ACK OK Success read 1 /foo/bar", got %q`, msg.String())
		}
	default:
		t.Fatal("no response sent")
	}
}
