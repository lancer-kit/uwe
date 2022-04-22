package logrus_hook

import (
	"github.com/lancer-kit/uwe/v3"
	"github.com/sirupsen/logrus"
)

// ChiefHandler returns default `EventHandler` that can be used for `Chief.SetEventHandler(...)`.
func ChiefHandler(entry *logrus.Entry) uwe.EventHandler {
	return func(event uwe.Event) {
		var level logrus.Level
		switch event.Level {
		case uwe.LvlFatal, uwe.LvlError:
			level = logrus.ErrorLevel
		case uwe.LvlInfo:
			level = logrus.InfoLevel
		default:
			level = logrus.WarnLevel
		}

		entry.WithFields(event.Fields).Log(level, event.Message)
	}
}
