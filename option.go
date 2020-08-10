package micro

import (
	"strings"

	"github.com/whatisfaker/zaptrace/log"
)

const (
	ccTypeNacos int8 = iota + 1
	ccTypeFile

	scTypeNacos int8 = iota + 1
	scTypeNoop

	defaultConfPath = "config.yaml"
)

type options struct {
	addr         string
	configKey    string
	confPath     string
	ccType       int8
	scType       int8
	namespace    string
	logLevel     string
	logger       *log.Factory
	mysqlTracer  bool
	redisTracer  bool
	mongoTracer  bool
	influxTracer bool
}

type Option interface {
	apply(*options)
}

type option struct {
	f func(*options)
}

func (c *option) apply(o *options) {
	c.f(o)
}

func newOption(f func(*options)) *option {
	return &option{
		f: f,
	}
}

//NameSpace 服务中心/配置中心的命名空间
func NameSpace(namespace string) Option {
	return newOption(func(o *options) {
		o.namespace = namespace
	})
}

//FileConfigCenter 使用文件配置中心
func FileConfigCenter(path string) Option {
	return newOption(func(o *options) {
		o.ccType = ccTypeFile
		o.confPath = strings.Trim(path, " ")
	})
}

//NacosAddr
func NacosAddr(e string) Option {
	return newOption(func(o *options) {
		o.addr = e
	})
}

func LogLevel(level string) Option {
	return newOption(func(o *options) {
		o.logLevel = level
	})
}

func ConfigKey(key string) Option {
	return newOption(func(o *options) {
		o.configKey = key
	})
}

//Logger 设置日志
func Logger(logger *log.Factory) Option {
	return newOption(func(o *options) {
		o.logger = logger
	})
}

func EnableMySQLTracer() Option {
	return newOption(func(o *options) {
		o.mysqlTracer = true
	})
}

func EnableRedisTracer() Option {
	return newOption(func(o *options) {
		o.redisTracer = true
	})
}

func EnableMongoTracer() Option {
	return newOption(func(o *options) {
		o.mongoTracer = true
	})
}

func EnableInfluxTracer() Option {
	return newOption(func(o *options) {
		o.influxTracer = true
	})
}
