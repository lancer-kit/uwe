package socket

import (
	"encoding/json"
	"fmt"
	"net"
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
		return nil, fmt.Errorf("unable to encode input: %s", err)
	}

	response := &Response{}
	err = decode.Decode(response)
	if err != nil {
		return nil, fmt.Errorf("unable to decode input: %s", err)
	}
	return response, nil
}
