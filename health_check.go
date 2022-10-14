package uwe

import (
	"encoding/json"

	"github.com/sheb-gregor/sam"
)

const (
	// StatusAction is a command useful for health-checks, because it returns status of all workers.
	StatusAction = "status"
	// PingAction is a simple command that returns the "pong" message.
	PingAction = "ping"
)

// AppInfo is a details of the *Application* build.
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Tag     string `json:"tag"`
}

// SocketName returns name of *Chief Service Socket*.
func (app AppInfo) SocketName() string {
	return "/tmp/_uwe_" + app.Name + ".socket"
}

// StateInfo is result the `StatusAction` command.
type StateInfo struct {
	App     AppInfo                  `json:"app"`
	Workers map[WorkerName]sam.State `json:"workers"`
}

// ParseStateInfo decodes `StateInfo` from the JSON response for the `StatusAction` command.
func ParseStateInfo(data json.RawMessage) (*StateInfo, error) {
	var res = new(StateInfo)
	err := json.Unmarshal(data, res)
	if err != nil {
		return nil, err
	}

	return res, nil

}
