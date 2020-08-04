package micro

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	ifxclient "github.com/influxdata/influxdb1-client/v2"
	"github.com/jinzhu/gorm"
	"github.com/whatisfaker/conf"
	"github.com/whatisfaker/conf/amqp"
	"github.com/whatisfaker/gormzap"
	"github.com/whatisfaker/zaptrace/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

var ErrUnsupportedConfig = errors.New("unsupported config")
var ErrConfigShouldPtrOrStruct = errors.New("config should be a struct or struct's pointer")
var ErrEmptyTag = errors.New("empty tag value")

func newDeps(v interface{}, structTag string, log *log.Factory) (*Deps, error) {
	s := reflect.ValueOf(v)
	switch s.Type().Kind() {
	case reflect.Ptr:
		s = s.Elem()
		if s.Type().Kind() != reflect.Struct {
			return nil, ErrUnsupportedConfig
		}
	case reflect.Struct:
	default:
		return nil, ErrConfigShouldPtrOrStruct
	}
	l := s.NumField()
	d := &Deps{
		deps: make(map[string]interface{}),
		log:  log,
	}
	for i := 0; i < l; i++ {
		tag := s.Type().Field(i).Tag
		var key string
		var ok bool
		if key, ok = tag.Lookup(structTag); ok {
			key = strings.Trim(key, " ")
			if key == "" {
				return nil, ErrEmptyTag
			}
		}
		f := s.Field(i).Interface()
		switch s := f.(type) {
		case conf.MysqlConfig:
			dep, err := conf.MySQLClient(&s)
			if err != nil {
				log.Normal().Error("init mysql error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			dep.SetLogger(gormzap.New(log.ZapLogger))
			d.deps[key] = dep
		case *conf.MysqlConfig:
			dep, err := conf.MySQLClient(s)
			if err != nil {
				log.Normal().Error("init mysql error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			dep.SetLogger(gormzap.New(log.ZapLogger))
			d.deps[key] = dep
		case conf.RedisConfig:
			dep, err := conf.RedisClient(&s)
			if err != nil {
				log.Normal().Error("init redis error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			d.deps[key] = dep
		case *conf.RedisConfig:
			dep, err := conf.RedisClient(s)
			if err != nil {
				log.Normal().Error("init redis error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			d.deps[key] = dep
		case conf.MongoDBConfig:
			dep, err := conf.MongoDBClient(&s)
			if err != nil {
				log.Normal().Error("init mongodb error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			d.deps[key] = dep
		case *conf.MongoDBConfig:
			dep, err := conf.MongoDBClient(s)
			if err != nil {
				log.Normal().Error("init mongodb error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			d.deps[key] = dep
		case conf.RabbitMQConfig:
			dep := amqp.NewRabbitMQClient(s.Address, s.Username, s.Password, log.With(zap.String("agent", "rabbitmq")), false)
			d.deps[key] = dep
		case *conf.RabbitMQConfig:
			dep := amqp.NewRabbitMQClient(s.Address, s.Username, s.Password, log.With(zap.String("agent", "rabbitmq")), false)
			d.deps[key] = dep
		case conf.InfluxConfig:
			influxConfig := ifxclient.HTTPConfig{
				Addr:     fmt.Sprint("http://", s.Addr),
				Username: s.UserName,
				Password: s.Password,
				Timeout:  30 * time.Second,
			}
			influxClient, err := ifxclient.NewHTTPClient(influxConfig)
			if err != nil {
				log.Normal().Error("init influx db error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			_, _, err = influxClient.Ping(2 * time.Second)
			if err != nil {
				log.Normal().Error("init influx db error, ping error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			d.deps[key] = influxClient
		case *conf.InfluxConfig:
			influxConfig := ifxclient.HTTPConfig{
				Addr:     fmt.Sprint("http://", s.Addr),
				Username: s.UserName,
				Password: s.Password,
				Timeout:  30 * time.Second,
			}
			influxClient, err := ifxclient.NewHTTPClient(influxConfig)
			if err != nil {
				log.Normal().Error("init influx db error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			_, _, err = influxClient.Ping(2 * time.Second)
			if err != nil {
				log.Normal().Error("init influx db error, ping error", zap.String("key", key), zap.Error(err))
				return nil, err
			}
			d.deps[key] = influxClient
		default:
		}
	}
	return d, nil
}

type Deps struct {
	deps map[string]interface{}
	log  *log.Factory
}

func (c *Deps) GetMySQL(key string) *gorm.DB {
	if v, ok := c.deps[key]; ok {
		if r, ok := v.(*gorm.DB); ok {
			return r
		}
	}
	c.log.Normal().Fatal("miss gorm db key", zap.String("key", key))
	return nil
}

func (c *Deps) GetRedis(key string) redis.Cmdable {
	if v, ok := c.deps[key]; ok {
		if r, ok := v.(redis.Cmdable); ok {
			return r
		}
	}
	c.log.Normal().Fatal("miss redis client key", zap.String("key", key))
	return nil
}

func (c *Deps) GetMongoDB(key string) *mongo.Client {
	if v, ok := c.deps[key]; ok {
		if r, ok := v.(*mongo.Client); ok {
			return r
		}
	}
	c.log.Normal().Fatal("miss mongo db key", zap.String("key", key))
	return nil
}

func (c *Deps) GetInflux(key string) ifxclient.Client {
	if v, ok := c.deps[key]; ok {
		if r, ok := v.(ifxclient.Client); ok {
			return r
		}
	}
	c.log.Normal().Fatal("miss influx key", zap.String("key", key))
	return nil
}

func (c *Deps) GetRabbitMQ(key string) amqp.Client {
	if v, ok := c.deps[key]; ok {
		if r, ok := v.(amqp.Client); ok {
			return r
		}
	}
	c.log.Normal().Fatal("miss rabbitmq key", zap.String("key", key))
	return nil
}
