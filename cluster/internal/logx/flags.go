package logx

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	zapx "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var levelStrings = map[string]zapcore.Level{
	"debug": zap.DebugLevel,
	"info":  zap.InfoLevel,
	"error": zap.ErrorLevel,
	"panic": zap.PanicLevel,
}

var stackLevelStrings = map[string]zapcore.Level{
	"info":  zap.InfoLevel,
	"error": zap.ErrorLevel,
	"panic": zap.PanicLevel,
}

type encoderFlag struct {
	setFunc func(zapx.NewEncoderFunc)
	value   string
}

var _ flag.Value = &encoderFlag{}

func (ev *encoderFlag) String() string {
	return ev.value
}

func (ev *encoderFlag) Type() string {
	return "encoder"
}

func (ev *encoderFlag) Set(flagValue string) error {
	val := strings.ToLower(flagValue)
	switch val {
	case "json":
		ev.setFunc(newJSONEncoder)
	case "console":
		ev.setFunc(newConsoleEncoder)
	default:
		return fmt.Errorf("invalid encoder value \"%s\"", flagValue)
	}
	ev.value = flagValue
	return nil
}

type levelFlag struct {
	setFunc func(zapcore.LevelEnabler)
	value   string
}

var _ flag.Value = &levelFlag{}

func (ev *levelFlag) Set(flagValue string) error {
	level, validLevel := levelStrings[strings.ToLower(flagValue)]
	if !validLevel {
		logLevel, err := strconv.Atoi(flagValue)
		if err != nil {
			return fmt.Errorf("invalid log level \"%s\"", flagValue)
		}
		if logLevel > 0 {
			intLevel := -1 * logLevel
			ev.setFunc(zap.NewAtomicLevelAt(zapcore.Level(int8(intLevel))))
		} else {
			return fmt.Errorf("invalid log level \"%s\"", flagValue)
		}
	} else {
		ev.setFunc(zap.NewAtomicLevelAt(level))
	}
	ev.value = flagValue
	return nil
}

func (ev *levelFlag) String() string {
	return ev.value
}

func (ev *levelFlag) Type() string {
	return "level"
}

type stackTraceFlag struct {
	setFunc func(zapcore.LevelEnabler)
	value   string
}

var _ flag.Value = &stackTraceFlag{}

func (ev *stackTraceFlag) Set(flagValue string) error {
	level, validLevel := stackLevelStrings[strings.ToLower(flagValue)]
	if !validLevel {
		return fmt.Errorf("invalid stacktrace level \"%s\"", flagValue)
	}
	ev.setFunc(zap.NewAtomicLevelAt(level))
	ev.value = flagValue
	return nil
}

func (ev *stackTraceFlag) String() string {
	return ev.value
}

func (ev *stackTraceFlag) Type() string {
	return "level"
}

type timeEncodingFlag struct {
	setFunc func(zapcore.TimeEncoder)
	value   string
}

var _ flag.Value = &timeEncodingFlag{}

func (ev *timeEncodingFlag) String() string {
	return ev.value
}

func (ev *timeEncodingFlag) Type() string {
	return "time-encoding"
}

func (ev *timeEncodingFlag) Set(flagValue string) error {
	val := strings.ToLower(flagValue)
	switch val {
	case "rfc3339nano":
		ev.setFunc(zapcore.RFC3339NanoTimeEncoder)
	case "rfc3339":
		ev.setFunc(zapcore.RFC3339TimeEncoder)
	case "iso8601":
		ev.setFunc(zapcore.ISO8601TimeEncoder)
	case "millis":
		ev.setFunc(zapcore.EpochMillisTimeEncoder)
	case "nanos":
		ev.setFunc(zapcore.EpochNanosTimeEncoder)
	case "epoch":
		ev.setFunc(zapcore.EpochTimeEncoder)
	default:
		return fmt.Errorf("invalid time-encoding value \"%s\"", flagValue)
	}

	ev.value = flagValue
	return nil
}
