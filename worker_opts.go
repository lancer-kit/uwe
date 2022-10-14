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
	NoRestart RestartOption = 1 << iota
	RestartOnFail
	RestartOnError
	RestartWithReinit
)
