package tcpserver

import (
	"log"
	"net"

	"github.com/UniversityRadioYork/bifrost-server/pool"
	"github.com/UniversityRadioYork/bifrost-server/request"
	"gopkg.in/tomb.v2"
)

// Serve creates and runs a Bifrost server using TCP as a transport.
// It will respond to requests using the functions in requestMap.
// New clients will be served 'OHAI <serverid>'.
func Serve(requestMap request.Map, serverid, hostport string) {
	ln, err := net.Listen("tcp", hostport)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := ln.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	cpQuit := make(chan struct{})

	var t tomb.Tomb

	p := pool.New(serverid, cpQuit)
	t.Go(func() error { return p.Run(&t) })

	requests := make(chan *request.Request)

	t.Go(func() error { return acceptLoop(ln, requests, p, &t) })

	router := request.NewRouter(requestMap, p.Broadcast)
	requestLoop(requests, router, &t)

	if kerr := t.Killf("main loop closing"); kerr != nil {
		log.Fatal(kerr)
	}

	// To close the accept loop, we have to kill off the acceptor.
	if lerr := ln.Close(); lerr != nil {
		log.Fatal(lerr)
	}

	log.Println(t.Wait())
}

func acceptLoop(ln net.Listener, requests chan<- *request.Request, p *pool.Pool, t *tomb.Tomb) (err error) {
	defer func() { log.Println("accept loop closing") }()

	for {
		conn, cerr := ln.Accept()
		if cerr != nil {
			log.Println(err)
			err = cerr
			break
		}

		t.Go(func() error { return handleConnection(conn, requests, p, t) })
	}

	return
}

func requestLoop(requests <-chan *request.Request, router *request.Router, t *tomb.Tomb) {
	for {
		select {
		case <-t.Dying():
			return
		case r, more := <-requests:
			if !more {
				return
			}
			log.Printf("received request: %q", r.Contents)
			if finished := router.Dispatch(r); finished {
				return
			}
		}
	}
}
