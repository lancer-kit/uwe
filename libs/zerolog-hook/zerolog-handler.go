package zerolog_hook

import (
	"github.com/lancer-kit/uwe/v3"
	"github.com/rs/zerolog"
)

// ChiefHandler returns default `EventHandler` that can be used for `Chief.SetEventHandler(...)`.
func ChiefHandler(log zerolog.Logger) func(event uwe.Event) {
	return func(event uwe.Event) {
		var level zerolog.Level
		switch event.Level {
		case uwe.LvlFatal, uwe.LvlError:
			level = zerolog.ErrorLevel
		case uwe.LvlInfo:
			level = zerolog.InfoLevel
		default:
			level = zerolog.WarnLevel
		}
		l := log.WithLevel(level)
		for s, i := range event.Fields {
			l = l.Interface(s, i)
		}

		l.Msg(event.Message)
	}
}
