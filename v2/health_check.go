package uwe

import (
	"encoding/json"

	"github.com/lancer-kit/sam"
	"github.com/lancer-kit/uwe/v2/socket"
	"github.com/urfave/cli"
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

// CliCheckCommand returns `cli.Command`, which allows you to check the health of a running instance **Application**
// with `ServiceSocket` enabled using `(Chief) .EnableServiceSocket(...)`
func CliCheckCommand(app AppInfo, workerListProvider func(c *cli.Context) []WorkerName) cli.Command {
	const detailsFlag = "details"
	return cli.Command{
		Name:  "check",
		Usage: "receives information about the status of a running service through an open service socket",
		Action: func(c *cli.Context) error {
			client := socket.NewClient(app.SocketName())
			resp, err := client.Send(socket.Request{Action: StatusAction})
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			if resp.Status != socket.StatusOk {
				return cli.NewExitError(resp.Error, 1)
			}

			stateInfo, err := ParseStateInfo(resp.Data)
			if err != nil {
				return cli.NewExitError("invalid response:"+err.Error(), 1)
			}

			for _, worker := range workerListProvider(c) {
				state := stateInfo.Workers[worker]
				if state != WStateRun {
					return cli.NewExitError(worker+" is not active", 7)
				}
			}

			if !c.Bool(detailsFlag) {
				return nil
			}

			data, err := json.MarshalIndent(stateInfo, "", "  ")
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			println(string(data))
			return nil
		},

		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  detailsFlag + ", d",
				Usage: "if true, then prints the detailed json result to the stdout, otherwise the output will be empty",
			},
		},
	}
}
