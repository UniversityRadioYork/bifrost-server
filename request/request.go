package request

import (
	"fmt"
	"github.com/UniversityRadioYork/baps3-go"
	"log"
)

// Request is the type of a request.
// It contains a Message, as well as a channel to use for responses to the requesting client.
type Request struct {
	Contents *baps3.Message
	Response chan<- *baps3.Message
}

// NewRequest creates a new Request with the given contents and response channel.
func NewRequest(contents *baps3.Message, response chan<- *baps3.Message) *Request {
	return &Request{
		Contents: contents,
		Response: response,
	}
}

// Handler is the type of functions that handle a request.
// The function is given a broadcast channel, a response channel, and the request arguments.
// It returns a Boolean flag specifying whether the server should stop, and an error.
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

	msg := rq.Contents

	// TODO: handle bad command
	cmdfunc, ok := m.handlers[msg.Word()]
	if ok {
		finished, lerr = cmdfunc(m.broadcast, rq.Response, msg.Args())
	} else {
		lerr = fmt.Errorf("FIXME: unknown command %q", msg.Word())
	}

	acktype := "???"
	lstr := "Success"
	if lerr == nil {
		acktype = "OK"
	} else {
		// TODO: proper error distinguishment
		acktype = "FAIL"
		lstr = lerr.Error()
	}

	rq.Response <- makeAck(acktype, lstr, msg)
	return finished
}

func makeAck(acktype, lstr string, msg *baps3.Message) *baps3.Message {
	log.Printf("Sending ack: %q, %q", acktype, lstr)

	rmsg := baps3.NewMessage(baps3.RsAck).AddArg(acktype).AddArg(lstr)

	// Append the entire request onto the end of the acknowledgement.
	rmsg.AddArg(msg.Word().String())
	for _, arg := range msg.Args() {
		rmsg.AddArg(arg)
	}

	return rmsg
}
