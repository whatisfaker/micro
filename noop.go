package micro

import (
	"context"

	"go.uber.org/zap"
)

type msNp struct {
	params      *paramMap
	discoveryIP string
	name        string
}

var _ MicroService = (*msNp)(nil)

func newNpMicroService(name string, params ...Param) (*msNp, error) {
	p := &paramMap{
		metadata: map[string]interface{}{},
		weight:   defaultMSWeight,
	}
	for _, v := range params {
		v.apply(p)
	}
	c := &msNp{
		params: p,
		name:   name,
	}
	var err error
	if p.discoveryIP != "" {
		p.discoveryIP, err = getOutboundIP()
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *msNp) Start(ctx context.Context) error {
	return nil
}

func (c *msNp) Name() string {
	return c.name
}

func (c *msNp) Discovery() (string, uint) {
	return c.discoveryIP, 0
}

func (c *msNp) Weight() uint32 {
	return c.params.weight
}

func (c *msNp) Group() string {
	return "DEFAULT_GROUP"
}

func (c *msNp) Metadata() map[string]interface{} {
	return c.params.metadata
}

func (c *msNp) Shutdown(ctx context.Context) {}

//RegisterNoop 注册不暴露端口的微服务
func (c *MSManager) RegisterNoop(name string, params ...Param) error {
	svc, err := newNpMicroService(name, params...)
	if err != nil {
		c.log.Normal().Error("register noop", zap.Error(err), zap.String("name", name))
		return err
	}
	c.svcs = append(c.svcs, svc)
	return nil
}
