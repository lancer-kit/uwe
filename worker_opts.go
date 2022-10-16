package uwe

type WorkerOpts interface {
	thisIsOption()
}

type RestartOption int

func (opt RestartOption) Is(mode RestartOption) bool {
	return opt&mode == mode
}

func (RestartOption) thisIsOption() {}

const (
	RestartOnFail RestartOption = 1 << iota
	RestartOnError
	RestartWithReInit

	NoRestart     RestartOption = -1
	StopAppOnFail RestartOption = -2
)

const (
	Restart          = RestartOnFail | RestartOnError
	RestartAndReInit = RestartOnFail | RestartOnError | RestartWithReInit
)
