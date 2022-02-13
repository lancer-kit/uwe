package socket

import (
	"encoding/json"
	"net"

	"github.com/pkg/errors"
)

// Client provides the ability to communicate over the socket
// with some application running `Server`.
type Client struct {
	socketName string
}

// NewClient returns new `Client`.
func NewClient(socketName string) *Client {
	return &Client{socketName: socketName}
}

// Send tries to send a command in the `Request` through the socket to the `Server` and process the `Response`.
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
