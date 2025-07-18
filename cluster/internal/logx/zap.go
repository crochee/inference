// Package logx provides a zap based logger
package logx

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	zapx "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func newJSONEncoder(opts ...zapx.EncoderConfigOption) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	for _, opt := range opts {
		opt(&encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

func newConsoleEncoder(opts ...zapx.EncoderConfigOption) zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	for _, opt := range opts {
		opt(&encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// BindFlags will parse the given flagset for zap option flags and set the log options accordingly:
//   - zap-devel:
//     Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn)
//     Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
//   - zap-encoder: Zap log encoding (one of 'json' or 'console')
//   - zap-log-level: Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', 'panic'
//     or any integer value > 0 which corresponds to custom debug levels of increasing verbosity").
//   - zap-stacktrace-level: Zap Level at and above which stacktraces are captured (one of 'info', 'error' or 'panic')
//   - zap-time-encoding: Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'),
//     Defaults to 'epoch'.
func BindFlags(o *zapx.Options, fs *pflag.FlagSet) error {
	// Set Development mode value
	fs.BoolVar(&o.Development, "zap-devel", o.Development,
		"Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). "+
			"Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)")
	if err := viper.BindPFlag("zap-devel", fs.Lookup("zap-devel")); err != nil {
		return err
	}
	// Set Encoder value
	var encVal encoderFlag
	encVal.setFunc = func(fromFlag zapx.NewEncoderFunc) {
		o.NewEncoder = zapx.NewEncoderFunc(fromFlag)
	}
	fs.Var(&encVal, "zap-encoder", "Zap log encoding (one of 'json' or 'console')")
	if err := viper.BindPFlag("zap-encoder", fs.Lookup("zap-encoder")); err != nil {
		return err
	}

	// Set the Log Level
	var levelVal levelFlag
	levelVal.setFunc = func(fromFlag zapcore.LevelEnabler) {
		o.Level = fromFlag
	}
	fs.Var(&levelVal, "zap-log-level",
		"Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', 'panic'"+
			"or any integer value > 0 which corresponds to custom debug levels of increasing verbosity")
	if err := viper.BindPFlag("zap-log-level", fs.Lookup("zap-log-level")); err != nil {
		return err
	}

	// Set the StrackTrace Level
	var stackVal stackTraceFlag
	stackVal.setFunc = func(fromFlag zapcore.LevelEnabler) {
		o.StacktraceLevel = fromFlag
	}
	fs.Var(&stackVal, "zap-stacktrace-level",
		"Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').")
	if err := viper.BindPFlag("zap-stacktrace-level", fs.Lookup("zap-stacktrace-level")); err != nil {
		return err
	}

	// Set the time encoding
	var timeEncoderVal timeEncodingFlag
	timeEncoderVal.setFunc = func(fromFlag zapcore.TimeEncoder) {
		o.TimeEncoder = fromFlag
	}
	fs.Var(&timeEncoderVal, "zap-time-encoding", "Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.")
	if err := viper.BindPFlag("zap-time-encoding", fs.Lookup("zap-time-encoding")); err != nil {
		return err
	}
	return nil
}
