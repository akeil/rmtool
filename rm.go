package rm

import (
	"akeil.net/akeil/rm/internal/logging"
	"strings"
)

func SetLogLevel(level string) {
	var lvl logging.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = logging.LevelDebug
	case "info":
		lvl = logging.LevelInfo
	case "warning":
		lvl = logging.LevelWarning
	case "error":
		lvl = logging.LevelError
	default:
		lvl = logging.LevelNone
	}
	logging.SetLevel(lvl)
}
