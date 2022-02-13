package socket

import "encoding/json"

const (
	// StatusOk means that command was successfully processed.
	StatusOk = 0
	// StatusErr means that command was not processed,
	// please check the `Response.Error` for details.
	StatusErr = 13
	// StatusInternalErr means than command was not sent or decoding of response failed.
	StatusInternalErr = -1
)

// Action is a pair of command name and command handler.
type Action struct {
	Name    string
	Handler ActionFunc
}

// ActionFunc is a specified handler of the socket command.
type ActionFunc func(request Request) Response

func defaultHandler(Request) Response {
	return Response{Status: StatusErr, Error: "unknown_action"}
}

// Request is a pair of command name and command arguments.
type Request struct {
	Action string          `json:"ActionFunc"`
	Args   json.RawMessage `json:"args"`
}

// Response is the result of executing the command handler.
type Response struct {
	Status int             `json:"status"`
	Error  string          `json:"error,omitempty"`
	Data   json.RawMessage `json:"data"`
}

// NewResponse correctly fills `Response` with passed arguments.
func NewResponse(status int, data interface{}, errorStr string) Response {
	val := json.RawMessage{}
	if data != nil {
		var err error
		val, err = json.Marshal(data)
		if err != nil {
			status = StatusInternalErr
			errorStr = err.Error()
		}

	}
	return Response{Status: status, Data: val, Error: errorStr}
}
