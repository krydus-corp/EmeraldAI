/*
 * File: logger.go
 * Project: log
 * File Created: Tuesday, 25th August 2020 11:18:13 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package log

import "errors"

// A global variable so that log functions can be directly accessed
var log Logger

// Fields Type to pass when we want to call WithFields for structured logging
type Fields map[string]interface{}

const (
	//Debug has verbose message
	Debug = "debug"
	//Info is default log level
	Info = "info"
	//Warn is for logging messages about possible issues
	Warn = "warn"
	//Error is for logging errors
	Error = "error"
	//Fatal is for logging fatal messages. The system shutsdown after logging the message.
	Fatal = "fatal"
)

const (
	InstanceZapLogger int = iota
)

var (
	errInvalidLoggerInstance = errors.New("invalid logger instance")
)

// Logger is our contract for the logger
type Logger interface {
	Debugf(format string, args ...interface{})

	Infof(format string, args ...interface{})

	Warnf(format string, args ...interface{})

	Errorf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	Panicf(format string, args ...interface{})

	WithFields(keyValues Fields) Logger

	Initialized() bool
}

// Configuration stores the config for the logger
// For some loggers there can only be one level across writers, for such the level of Console is picked by default
type Configuration struct {
	EnableConsole bool   `mapstructure:"enable_console,omitempty" yaml:"enable_console,omitempty"`
	EnableFile    bool   `mapstructure:"enable_file,omitempty" yaml:"enable_file,omitempty"`
	JSONFormat    bool   `mapstructure:"json_format,omitempty" yaml:"json_format,omitempty"`
	Level         string `mapstructure:"level,omitempty" yaml:"level,omitempty"`
	FileLocation  string `mapstructure:"file_location,omitempty" yaml:"file_location,omitempty"`
}

// New returns an instance of logger
func New(config Configuration, loggerInstance int) error {
	switch loggerInstance {
	case InstanceZapLogger:
		logger, err := newZapLogger(config)
		if err != nil {
			return err
		}
		log = logger
		return nil

	default:
		return errInvalidLoggerInstance
	}
}

func Initialized() bool {
	return log != nil
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}

func WithFields(keyValues Fields) Logger {
	return log.WithFields(keyValues)
}
