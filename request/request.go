package request

import (
	"fmt"
	"github.com/UniversityRadioYork/baps3-go"
	"log"
)

// Request is the type of a request.
// It contains a Message, as well as a channel to use for responses to the requesting client.
// For use in ACKs later on, it also contains the raw, tokenised line.
type Request struct {
	Raw      []string
	Contents *baps3.Message
	Response chan<- *baps3.Message
}

// NewRequest creates a new Request with the given contents and response channel.
func NewRequest(raw []string, contents *baps3.Message, response chan<- *baps3.Message) *Request {
	return &Request{
		Raw:      raw,
		Contents: contents,
		Response: response,
	}
}

// Handler is the type of functions that handle a request.
// The function is given, in turn:
//
// - A broadcast channel (sending messages to all connected clients);
// - A unicast channel (sending messages only to the requesting client);
// - The request arguments (not including the word: we assume that this is
//   implied by the Handler being part of a Map from words to Handlers.)
// It returns a Boolean flag specifying whether the server should stop due to
// this request (useful for requests like RqQuit), and an error (may be nil).
type Handler func(chan<- *baps3.Message, chan<- *baps3.Message, []string) (bool, error)

// Map is a map from requests (as message words) to Handlers.
type Map map[baps3.MessageWord]Handler

// A Router is a combination of a Map and auxiliary state to be passed to request handlers.
type Router struct {
	handlers  Map
	broadcast chan<- *baps3.Message
}

// NewRouter creates a new Router with the given handlers and broadcast channel.
func NewRouter(handlers Map, broadcast chan<- *baps3.Message) *Router {
	return &Router{
		handlers:  handlers,
		broadcast: broadcast,
	}
}

// Dispatch dispatches the given request according to the request router.
// It also takes a broadcast channel to pass to the request handler.
func (m *Router) Dispatch(rq *Request) bool {
	var lerr error
	finished := false
	iswhat := false

	msg := rq.Contents

	// TODO: handle bad command
	cmdfunc, ok := m.handlers[msg.Word()]
	if ok {
		finished, lerr = cmdfunc(m.broadcast, rq.Response, msg.Args())
	} else {
		lerr = fmt.Errorf("unknown request %q", rq.Raw[0])
		iswhat = true
	}

	lstr := "Success"
	if lerr != nil {
		lstr = lerr.Error()
	}

	acktype := "OK"
	if lerr == nil {
		// Intentionally left blank.  This would be "OK", but we set it
		// above to initialise acktype.
	} else if iswhat {
		// TODO: make more robust
		acktype = "WHAT"
	} else {
		// TODO: proper error distinguishment
		acktype = "FAIL"
	}

	rq.Response <- makeAck(rq.Raw, acktype, lstr)
	return finished
}

func makeAck(raw []string, acktype, lstr string) *baps3.Message {
	log.Printf("Sending ack: %q, %q", acktype, lstr)

	rmsg := baps3.NewMessage(baps3.RsAck).AddArg(acktype).AddArg(lstr)

	// Append the entire raw request onto the end of the acknowledgement.
	for _, arg := range raw {
		rmsg.AddArg(arg)
	}

	return rmsg
}
