package micro

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type fileCC struct {
	path string
	log  *log.Factory
}

var _ ConfigCenter = (*fileCC)(nil)

func newFileCC(path string, log *log.Factory) *fileCC {
	return &fileCC{
		path: path,
		log:  log,
	}
}

func (c *fileCC) SetConfig(ctx context.Context, cfg interface{}) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		if err != nil {
			c.log.Trace(ctx).Error("SetConfig", zap.Error(err))
		}
		return err
	}
	err = ioutil.WriteFile(c.path, b, 0755)
	if err != nil {
		c.log.Trace(ctx).Error("SetConfig", zap.Error(err))
	}
	return err
}

func (c *fileCC) RemoveConfig(ctx context.Context, cfg interface{}) error {
	err := os.Remove(c.path)
	if err != nil {
		c.log.Trace(ctx).Error("RemoveConfig", zap.Error(err))
	}
	return err
}

func (c *fileCC) GetConfig(ctx context.Context, cfg interface{}) error {
	b, err := ioutil.ReadFile(c.path)
	if err != nil {
		c.log.Trace(ctx).Error("RemoveConfig", zap.Error(err))
		return err
	}
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		c.log.Trace(ctx).Error("RemoveConfig", zap.Error(err))
	}
	return err
}

func (c *fileCC) GetConfigAndWatch(ctx context.Context, cfg interface{}, cb func(string, string, interface{}, error)) error {
	b, err := ioutil.ReadFile(c.path)
	if err != nil {
		c.log.Trace(ctx).Error("GetConfigAndWatch", zap.Error(err))
		return err
	}
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		c.log.Trace(ctx).Error("GetConfigAndWatch", zap.Error(err))
		return err
	}
	c.log.Trace(ctx).Warn("GetConfigAndWatch(Watch)", zap.Error(ErrNotApplicable))
	cb("", "", nil, ErrNotApplicable)
	return nil
}
