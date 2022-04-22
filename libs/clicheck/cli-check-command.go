package clicheck

import (
	"encoding/json"

	"github.com/lancer-kit/uwe/v3"
	"github.com/lancer-kit/uwe/v3/socket"
	"github.com/urfave/cli"
)

// CliCheckCommand returns `cli.Command`, which allows you to check the health of a running instance **Application**
// with `ServiceSocket` enabled using `(Chief) .EnableServiceSocket(...)`
func CliCheckCommand(app uwe.AppInfo, workerListProvider func(c *cli.Context) []uwe.WorkerName) cli.Command {
	const detailsFlag = "details"
	return cli.Command{
		Name:  "check",
		Usage: "receives information about the status of a running service through an open service socket",
		Action: func(c *cli.Context) error {
			client := socket.NewClient(app.SocketName())
			resp, err := client.Send(socket.Request{Action: uwe.StatusAction})
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			if resp.Status != socket.StatusOk {
				return cli.NewExitError(resp.Error, 1)
			}

			stateInfo, err := uwe.ParseStateInfo(resp.Data)
			if err != nil {
				return cli.NewExitError("invalid response:"+err.Error(), 1)
			}

			for _, worker := range workerListProvider(c) {
				state := stateInfo.Workers[worker]
				if state != uwe.WStateRun {
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
