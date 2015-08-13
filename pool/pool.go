// Package pool implements a generic client pool for Bifrost servers.
// This is (hopefully) independent of transport, but is mainly intended for TCP/text Bifrost.
package pool

import (
	"fmt"
	"log"

	"github.com/UniversityRadioYork/baps3-go"
	"gopkg.in/tomb.v2"
)

// Client is a set of channels alllowing a Pool to talk to server clients.
// The Client is agnostic of client transport (Text/TCP, HTTP/TCP, 9P, etc),
// and the other end of each channel is expected to be wired to something
// handling said transport.
type Client struct {
	// Broadcast is a channel for sending messages intended for all clients
	// to this client.
	Broadcast chan<- *baps3.Message

	// Unicast is a channel for sending messages intended only for this
	// client to this client.
	Unicast chan<- *baps3.Message

	// Disconnect is a channel for sending signals for this client to
	// disconnect.
	//
	// When a Client receives this signal, it _must_ send a remove Change
	// to the Pool.
	//
	// This should either be constantly monitored, or buffered, as all
	// Clients are signalled on pool closure.
	Disconnect chan<- struct{}
}

// Change is a request to the client pool to add or remove a client.
// Changes can be created using NewAddChange and NewRemoveChange.
type Change struct {
	client *Client
	added  bool
	reply  chan<- bool
}

// NewAddChange creates a new Change for adding a client.
func NewAddChange(client *Client, reply chan<- bool) *Change {
	return &Change{client: client, added: true, reply: reply}
}

// NewRemoveChange creates a new Change for removing a client.
func NewRemoveChange(client *Client, reply chan<- bool) *Change {
	return &Change{client: client, added: false, reply: reply}
}

type poolInner struct {
	contents  map[*Client]struct{}
	changes   <-chan *Change
	quit      <-chan struct{}
	broadcast <-chan *baps3.Message
	quitting  bool
	serverid  string
}

// Pool is the front-facing interface to a client pool.
// It contains the pool innards, and channels to communicate with it while it is running.
type Pool struct {
	inner *poolInner

	// Changes is the channel to use to send Changes to the Pool.
	// These register or un-register a Client with the Pool.
	Changes chan<- *Change

	// Broadcast is the channel used to send messages to all Clients.
	// These messages are sent down the Client's Broadcast channel.
	Broadcast chan<- *baps3.Message
}

// New creates a new Pool.
// The new Pool has no Clients registered.
func New(serverid string, quit chan struct{}) *Pool {
	changes := make(chan *Change)
	broadcast := make(chan *baps3.Message)

	p := poolInner{
		contents:  make(map[*Client]struct{}),
		changes:   changes,
		quit:      quit,
		broadcast: broadcast,
		quitting:  false,
		serverid:  serverid,
	}

	return &Pool{
		inner:     &p,
		Changes:   changes,
		Broadcast: broadcast,
	}
}

// Run runs the client pool loop.
// It takes one argument:
//   t: a Tomb, whose death causes this client pool to die.
func (p *Pool) Run(t *tomb.Tomb) error {
	return p.inner.run(t)
}

func (p *poolInner) sendBroadcast(cast *baps3.Message) {
	for client := range p.contents {
		client.Broadcast <- cast
	}
}

func (p *poolInner) signalQuits() {
	// This channel can get signalled multiple times.  We
	// make sure we only send disconnect signals once.
	if !p.quitting {
		p.quitting = true

		for client := range p.contents {
			client.Disconnect <- struct{}{}
		}
	}
}

func (p *poolInner) run(t *tomb.Tomb) (err error) {
	for {
		select {
		case change := <-p.changes:
			p.handleChange(change)

			// If we're quitting, we're now waiting for all of the
			// connections to close so we can quit.
			if p.quitting && 0 == len(p.contents) {
				return
			}
		case cast := <-p.broadcast:
			p.sendBroadcast(cast)
		case <-t.Dying():
			p.signalQuits()

			// If we don't have any connections, then close right
			// now.  Otherwise, we wait for those connections to
			// close.
			if 0 == len(p.contents) {
				return
			}
		}
	}
}

func (p *poolInner) handleChange(change *Change) {
	var err error
	if change.added {
		log.Printf("adding client: %q", change.client)
		err = p.addClient(change.client)
	} else {
		log.Printf("removing client: %q", change.client)
		err = p.removeClient(change.client)
	}

	if err != nil {
		log.Println(err)
	}
	change.reply <- err == nil

	// If the change was a successful add, send OHAI to the client.
	// We have to do it _after_ we send the reply, else we could deadlock:
	// the client waiting for the reply and us waiting for the client.
	if err == nil && change.added {
		change.client.Unicast <- baps3.NewMessage(baps3.RsOhai).AddArg(p.serverid)
	}
}

func (p *poolInner) addClient(client *Client) error {
	// Don't allow adding when quitting.
	if p.quitting {
		return fmt.Errorf("addClient: quitting")
	}

	if _, ok := p.contents[client]; ok {
		return fmt.Errorf("addClient: client %q already present", client)
	}
	p.contents[client] = struct{}{}

	return nil
}

func (p *poolInner) removeClient(client *Client) error {
	if _, ok := p.contents[client]; ok {
		delete(p.contents, client)
		return nil
	}
	return fmt.Errorf("removeClient: client %q not present", client)
}
