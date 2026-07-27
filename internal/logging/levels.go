package logging

import "log/slog"

// Git Flow Plus uses seven log levels, extending slog's built-in four
// (Debug/Info/Warn/Error) with Trace (finer than Debug, for very verbose
// diagnostic detail), Success (a distinguished "this operation completed"
// signal sitting between Info and Warn), and Fatal (an unrecoverable
// failure, above Error). Values are chosen to interleave with slog's own
// level constants (LevelDebug=-4, LevelInfo=0, LevelWarn=4, LevelError=8)
// so ordering comparisons (level >= threshold) behave correctly across
// both sets.
const (
	LevelTrace   = slog.Level(-8)
	LevelDebug   = slog.LevelDebug
	LevelInfo    = slog.LevelInfo
	LevelSuccess = slog.Level(2)
	LevelWarn    = slog.LevelWarn
	LevelError   = slog.LevelError
	LevelFatal   = slog.Level(12)
)

// levelLabel returns the display label for l, used by the console handler
// and by ParseLevel's inverse. Levels between the named constants (which
// slog permits, e.g. via WithAttrs-free custom levels) round down to the
// nearest lower named level, matching slog's own convention for unnamed
// levels.
func levelLabel(l slog.Level) string {
	switch {
	case l < LevelDebug:
		return "TRACE"
	case l < LevelInfo:
		return "DEBUG"
	case l < LevelSuccess:
		return "INFO"
	case l < LevelWarn:
		return "SUCCESS"
	case l < LevelError:
		return "WARN"
	case l < LevelFatal:
		return "ERROR"
	default:
		return "FATAL"
	}
}

// ParseLevel maps a case-insensitive level name (as used in config.json's
// "logging.level" field or the GITFLOWPLUS_LOG_LEVEL environment variable)
// to its slog.Level. Unrecognized names fall back to LevelInfo.
func ParseLevel(name string) slog.Level {
	switch name {
	case "trace", "TRACE", "Trace":
		return LevelTrace
	case "debug", "DEBUG", "Debug":
		return LevelDebug
	case "success", "SUCCESS", "Success":
		return LevelSuccess
	case "warn", "WARN", "Warn", "warning", "WARNING", "Warning":
		return LevelWarn
	case "error", "ERROR", "Error":
		return LevelError
	case "fatal", "FATAL", "Fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}
