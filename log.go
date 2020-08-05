package micro

import (
	"fmt"

	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

type zapLogger struct {
	zaplogger *log.Factory
}

func NewZapLogger(logger *log.Factory) *zapLogger {
	return &zapLogger{
		zaplogger: logger,
	}
}

func (c *zapLogger) Error(msg string, params ...interface{}) {
	c.zaplogger.Normal().Error(msg, zap.String("ext", fmt.Sprintln(msg, params)))
}

func (c *zapLogger) Warn(msg string, params ...interface{}) {
	c.zaplogger.Normal().Warn(msg, zap.String("ext", fmt.Sprintln(msg, params)))
}

func (c *zapLogger) Info(msg string, params ...interface{}) {
	c.zaplogger.Normal().Info(msg, zap.String("ext", fmt.Sprintln(msg, params)))
}

func (c *zapLogger) Debug(msg string, params ...interface{}) {
	c.zaplogger.Normal().Debug(msg, zap.String("ext", fmt.Sprintln(msg, params)))
}

func (c *zapLogger) Level(level string) {
	c.zaplogger.SetLevel(level)
}

func (c *zapLogger) SetLevel(level string) {
	c.zaplogger.SetLevel(level)
}