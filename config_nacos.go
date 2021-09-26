package micro

import (
	"context"
	"errors"

	"github.com/magicdvd/nacos-client"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	nacosDefaultGroup = "DEFAULT_GROUP"
)

type nacosCC struct {
	client nacos.ServiceCmdable
	key    string
	log    *log.Factory
}

var _ ConfigCenter = (*nacosCC)(nil)

func newNacosCC(addr string, namespace string, key string, log *log.Factory) (*nacosCC, error) {
	client, err := nacos.NewServiceClient(addr, nacos.DefaultTenant(namespace), nacos.Log(NewZapLogger(log)), nacos.LogLevel(log.Level()))
	if err != nil {
		return nil, err
	}
	return &nacosCC{
		client: client,
		key:    key,
		log:    log,
	}, nil
}

func (c *nacosCC) SetConfig(ctx context.Context, cfg interface{}) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	err = c.client.PublishConfig(c.key, nacosDefaultGroup, string(b))
	if err != nil {
		c.log.Trace(ctx).Error("SetConfig", zap.Error(err))
	}
	return err
}

func (c *nacosCC) RemoveConfig(ctx context.Context, cfg interface{}) error {
	err := c.client.RemoveConfig(c.key, nacosDefaultGroup)
	if err != nil {
		c.log.Trace(ctx).Error("RemoveConfig", zap.Error(err))
	}
	return err
}

func (c *nacosCC) GetConfig(ctx context.Context, cfg interface{}) error {
	if c.key == "" {
		err := errors.New("nacos config key is empty")
		c.log.Trace(ctx).Error("GetConfig", zap.Error(err))
		return err
	}
	str, err := c.client.GetConfig(c.key, nacosDefaultGroup)
	if err != nil {
		c.log.Trace(ctx).Error("GetConfig", zap.Error(err))
		return err
	}
	if str != "" {
		return yaml.Unmarshal([]byte(str), cfg)
	}
	return nil
}
