package uwe

type WorkerOpts interface {
	thisIsOption()
}

// RestartOption is a behavior strategy
// for workers in case of error exit or panic.
type RestartOption int

func (opt RestartOption) Is(mode RestartOption) bool {
	return opt&mode == mode
}

func (RestartOption) thisIsOption() {}

const (
	// RestartOnFail strategy to restart worker ONLY if it panics.
	// Worker will be restarted by calling the Run() method again.
	RestartOnFail RestartOption = 1 << iota
	// RestartOnError strategy to restart worker
	// ONLY if the Run() method return error.
	// Worker will be restarted by calling the Run() method again.
	RestartOnError
	// RestartWithReInit strategy that adds reinitialization before restart.
	// RestartWithReInit works only with RestartOnFail and/or RestartOnError.
	// Worker will be reinitialized and restarted
	// by calling the Init() and Run() method again.
	RestartWithReInit

	// NoRestart is a default strategy.
	//Worker wouldn't be restarted.
	NoRestart RestartOption = -1
	// StopAppOnFail strategy to whole stop app
	// in case if Worker panicked or exited with error.
	StopAppOnFail RestartOption = -2
)

const (
	// Restart is a strategy to restart Worker
	// in case of panic or exit with error.
	Restart = RestartOnFail | RestartOnError
	// RestartAndReInit is a strategy to reinitialize and restart Worker
	// in case of panic or exit with error.
	RestartAndReInit = RestartOnFail | RestartOnError | RestartWithReInit
)
