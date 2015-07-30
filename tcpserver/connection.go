package tcpserver

import (
	"io"
	"log"
	"net"

	"github.com/UniversityRadioYork/baps3-go"
	"github.com/UniversityRadioYork/bifrost-server/pool"
	"github.com/UniversityRadioYork/bifrost-server/request"
	"gopkg.in/tomb.v2"
)

func handleConnection(conn net.Conn, requests chan<- *request.Request, p *pool.Pool, t *tomb.Tomb) (err error) {
	defer func() { log.Println("connection write side closing") }()

	responses := make(chan *baps3.Message)
	disconnect := make(chan struct{})
	reply := make(chan bool)

	client := pool.Client{
		Broadcast:  responses,
		Disconnect: disconnect,
	}

	// Tell the client pool we've arrived, and how to contact us.
	p.Changes <- pool.NewAddChange(&client, reply)
	if r := <-reply; !r {
		log.Println("connection refused by client pool")
		return
	}

	t.Go(func() error { return handleConnectionRead(conn, requests, responses) })
	err = handleConnectionWrite(conn, responses, t.Dying())

	// If we get here, the write loop has closed.
	// This only happens if the responses channel is dead, which is either
	// from the client pool or the read loop closing it.

	// Tell the client pool we're off.
	log.Println("connection write side signalling closure")
	p.Changes <- pool.NewRemoveChange(&client, reply)
	if r := <-reply; !r {
		log.Println("connection removal refused by client pool")
		return
	}

	// Now close the actual connection.
	if err := conn.Close(); err != nil {
		log.Printf("couldn't close connection: %q", err)
	}

	return
}

func handleConnectionWrite(conn net.Conn, responses <-chan *baps3.Message, quit <-chan struct{}) (err error) {
	for {
		select {
		case <-quit:
			log.Println("connection has quit signal")
			// The write routine has responsibility for listening
			// to the server quit channel to see if the connection
			// must be closed.
			return
		case response, more := <-responses:
			if !more {
				return
			}
			if werr := writeResponse(conn, response); err != nil {
				log.Printf("can't write: %s", werr)
				return
			}
		}
	}
}

func handleConnectionRead(conn net.Conn, requests chan<- *request.Request, responses chan<- *baps3.Message) (err error) {
	defer func() { log.Println("connection read side closing") }()

	// Ensure the write portion is closed when reading stops.
	// The closing of the responses channel will do this.
	// Note that the requests channel is later closed when the writing
	// section stops.
	defer func() {
		close(responses)
	}()

	buf := make([]byte, 1024)
	tok := baps3.NewTokeniser()

	for {
		nbytes, rerr := conn.Read(buf)
		if rerr != nil {
			if rerr != io.EOF {
				// TODO: handle error correctly, send error to client
				log.Printf("connection read error: %q", err)
				err = rerr
			}
			return
		}

		lines, _, terr := tok.Tokenise(buf[:nbytes])
		if terr != nil {
			// TODO: handle error correctly, retry tokenising perhaps
			log.Printf("connection tokenise error: %q", err)
			terr = err
			return
		}

		for _, line := range lines {
			msg, err := baps3.LineToMessage(line)
			if err != nil {
				log.Printf("bad message: %q", line)
			} else {
				requests <- request.NewRequest(msg, responses)
			}
		}
	}
}

func writeResponse(conn net.Conn, message *baps3.Message) error {
	bytes, err := message.Pack()
	if err != nil {
		return err
	}
	if _, err := conn.Write(bytes); err != nil {
		return err
	}
	return nil
}
