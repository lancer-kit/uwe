package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
)

// Server is a handler that opens a `net.Socket`
// and accepts commands and writes responses in JSON format.
type Server struct {
	socketName string
	handlers   map[string]ActionFunc
	errors     chan error
}

// NewServer creates a new server with some actions.
func NewServer(socketName string, actions ...Action) *Server {
	handlers := map[string]ActionFunc{}
	for _, action := range actions {
		handlers[action.Name] = action.Handler
	}
	return &Server{socketName: socketName, handlers: handlers, errors: make(chan error)}
}

// Errors returns a channel with errors.
func (sw *Server) Errors() <-chan error {
	return sw.errors
}

// SetHandler adds new or replaces the command (action) handler.
func (sw *Server) SetHandler(name string, action ActionFunc) {
	sw.handlers[name] = action
}

// Serve creates the UNIX socket and starts listening for incoming commands.
// When command accepted server tries to decode message into `Request`.
// In case when the server has the handler for `Request` command
// it executes and writes a response in JSON format to the socket.
func (sw *Server) Serve(ctx context.Context) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		var ok bool
		err, ok = r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
	}()

	if err := sw.removeSocket(); err != nil {
		return err
	}

	// Creating the unix domain TCP socket
	var lc net.ListenConfig
	localSocket, err := lc.Listen(ctx, "unix", sw.socketName)
	if err != nil {
		return errors.Wrap(err, "unable to create unix domain socket")
	}

	// // Set the permissions 700 on this
	if err = os.Chmod(sw.socketName, 0700); err != nil {
		return errors.Wrap(err, "unable to change the permissions for the socket")
	}

	conns := make(chan net.Conn)

	go func() {
		for {
			socketConn, err := localSocket.Accept()
			if err != nil {
				sw.errors <- errors.Wrap(err, "accept failed")
				continue
			}
			conns <- socketConn
		}
	}()

	// Initiate and listen to the socket
	for {
		select {
		case <-ctx.Done():
			err = localSocket.Close()
			if err != nil {
				sw.errors <- errors.Wrap(err, "accept failed")
				return nil
			}

			if err := sw.removeSocket(); err != nil {
				sw.errors <- err
				return nil
			}
			return nil
		case socketConn := <-conns:
			err = sw.processSockRequest(socketConn)
			if err != nil {
				sw.errors <- errors.Wrap(err, "process failed")
				continue
			}
		}

	}
}

func (sw *Server) processSockRequest(conn net.Conn) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		var ok bool
		err, ok = r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
	}()

	defer func() {
		err = conn.Close()
	}()

	decode := json.NewDecoder(conn)
	encode := json.NewEncoder(conn)

	var in Request
	err = decode.Decode(&in)
	if err != nil {
		return errors.Wrap(err, "unable to decode input")
	}

	handler, ok := sw.handlers[in.Action]
	if !ok {
		handler = defaultHandler
	}

	result := handler(in)

	// Send response back to the socket request
	err = encode.Encode(result)
	if err != nil {
		return errors.Wrap(err, "unable to encode input")
	}

	return nil
}

func (sw *Server) removeSocket() error {
	_, err := os.Stat(sw.socketName)
	if os.IsNotExist(err) {
		return nil
	}
	if err := os.Remove(sw.socketName); err != nil {
		return errors.Wrap(err, "unable to remove the socket")
	}

	return nil
}
