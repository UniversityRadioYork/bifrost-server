// pool implements a generic client pool for Bifrost servers.
// This is (hopefully) independent of transport, but is mainly intended for TCP/text Bifrost.

package pool

import (
	"fmt"
	"log"

	"github.com/UniversityRadioYork/baps3-go"
	"gopkg.in/tomb.v2"
)

// A Client is a pair of channels alllowing a Pool to talk to server clients.
// The Client is agnostic of client transport (Text/TCP, HTTP/TCP, 9P, etc).
type Client struct {
	// Channel for sending broadcast messages to this client.
	Broadcast chan<- *baps3.Message
	// Channel for sending disconnect messages to this client.
	Disconnect chan<- struct{}
}

// Change is a request to the client pool to add or remove a client.
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
	inner     *poolInner
	Changes   chan<- *Change
	Broadcast chan<- *baps3.Message
}

// New creates a new client pool.
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

func (p *poolInner) run(t *tomb.Tomb) (err error) {
	defer func() { log.Println("client pool is closing") }()

	for {
		select {
		case change := <-p.changes:
			p.handleChange(change)

			log.Printf("clientPool: now %d clients", len(p.contents))

			// If we're quitting, we're now waiting for all of the
			// connections to close so we can quit.
			if p.quitting && 0 == len(p.contents) {
				return
			}
		case broadcast := <-p.broadcast:
			log.Println("broadcast: %q", broadcast)
			for client, _ := range p.contents {
				client.Broadcast <- broadcast
			}
		case <-t.Dying():
			log.Println("client pool is beginning to close")

			p.quitting = true

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
	var err error = nil
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
		// Technically, this is a unicast, but this shouldn't make any
		// difference.
		change.client.Broadcast <- baps3.NewMessage(baps3.RsOhai).AddArg(p.serverid)
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
