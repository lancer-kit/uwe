package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
)

const (
	StatusOk  = 0
	StatusErr = 13
)

type Response struct {
	Status int         `json:"status"`
	Error  string      `json:"error,omitempty"`
	Data   interface{} `json:"data"`
}

type Request struct {
	Action string          `json:"ActionFunc"`
	Args   json.RawMessage `json:"args"`
}

type Action struct {
	Name    string
	Handler ActionFunc
}

type ActionFunc func(request Request) Response

func defaultHandler(_ Request) Response {
	return Response{Status: StatusErr, Error: "unknown_action"}
}

type Server struct {
	socketName string
	handlers   map[string]ActionFunc
	errors     chan error
}

func NewServer(socketName string, actions ...Action) *Server {
	handlers := map[string]ActionFunc{}
	for _, action := range actions {
		handlers[action.Name] = action.Handler
	}
	return &Server{socketName: socketName, handlers: handlers, errors: make(chan error)}
}

func (sw *Server) Errors() <-chan error {
	return sw.errors
}

func (sw *Server) SetHandler(name string, action ActionFunc) {
	sw.handlers[name] = action
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
		default:
			socketConn, err := localSocket.Accept()
			if err != nil {
				sw.errors <- errors.Wrap(err, "accept failed")
				continue
			}

			err = sw.processSockRequest(socketConn)
			if err != nil {
				sw.errors <- errors.Wrap(err, "process failed")
				continue
			}
		}

	}
}

type Client struct {
	socketName string
}

func NewClient(socketName string) *Client {
	return &Client{socketName: socketName}
}

func (client Client) Send(request Request) (*Response, error) {
	conn, err := net.Dial("unix", client.socketName)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	decode := json.NewDecoder(conn)
	encode := json.NewEncoder(conn)

	err = encode.Encode(request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to encode input")
	}

	response := &Response{}
	err = decode.Decode(response)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode input")
	}
	return response, nil
}
