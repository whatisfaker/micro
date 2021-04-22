package micro

import (
	"context"
	"net"
	"time"

	"github.com/whatisfaker/ms"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

type msTCP struct {
	params      *paramMap
	srv         *ms.Server
	listen      string
	discoveryIP string
	port        uint
	name        string
	log         *log.Factory
	initFunc    func(context.Context, *ms.Server)
}

var _ MicroService = (*msTCP)(nil)

func newTCPMicroService(name string, listen string, initFunc func(context.Context, *ms.Server), log *log.Factory, params ...Param) (*msTCP, error) {
	p := &paramMap{
		metadata:    map[string]interface{}{},
		weight:      defaultMSWeight,
		tcpCodec:    nil,
		tcpRoute:    nil,
		tcpIdleTime: time.Minute,
	}
	for _, v := range params {
		v.apply(p)
	}
	c := &msTCP{
		params:   p,
		name:     name,
		listen:   listen,
		log:      log,
		initFunc: initFunc,
	}
	var err error
	c.discoveryIP, c.port, err = split2ipport(listen, p.discoveryIP)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *msTCP) Start(ctx context.Context) error {
	opts := make([]ms.ServerOption, 0)
	opts = append(opts, ms.BufferSize(1024), ms.Logger(NewZapLogger(c.log.With(zap.String("ms", "tcp")))))
	if c.params.tcpCodec != nil {
		opts = append(opts, ms.Codec(c.params.tcpCodec))
	}
	if c.params.tcpRoute != nil {
		opts = append(opts, ms.RouterKeyExtract(c.params.tcpRoute))
	}
	if c.params.tcpIdleTime > 0 {
		opts = append(opts, ms.ConnMaxIdleTime(c.params.tcpIdleTime))
	}
	if c.params.tcpBufSizeMin > 0 {
		if c.params.tcpBufSizeMax > c.params.tcpBufSizeMin {
			opts = append(opts, ms.BufferSize(c.params.tcpBufSizeMin, c.params.tcpBufSizeMax))
		} else {
			opts = append(opts, ms.BufferSize(c.params.tcpBufSizeMin))
		}
	}
	c.srv = ms.NewServer(opts...)
	if c.initFunc != nil {
		c.initFunc(ctx, c.srv)
	}
	tcpListen, err := net.Listen("tcp", c.listen)
	if err != nil {
		return err
	}
	return c.srv.Serve(ctx, tcpListen)
}

func (c *msTCP) Name() string {
	return c.name
}

func (c *msTCP) Discovery() (string, uint) {
	return c.discoveryIP, c.port
}

func (c *msTCP) Weight() uint32 {
	return c.params.weight
}

func (c *msTCP) Group() string {
	return MSGroupTCPServer
}

func (c *msTCP) Metadata() map[string]interface{} {
	return c.params.metadata
}

func (c *msTCP) Shutdown(ctx context.Context) {
	if !c.params.tcpManulShutdown {
		if c.srv != nil {
			_ = c.srv.Shutdown(ctx)
		}
	}
}

//RegisterTCP 注册tcp的微服务
func (c *MSManager) RegisterTCP(name string, listen string, initFunc func(context.Context, *ms.Server), params ...Param) error {
	svc, err := newTCPMicroService(name, listen, initFunc, c.log.With(zap.String("srv_tcp", name)), params...)
	if err != nil {
		c.log.Normal().Error("register tcp", zap.Error(err), zap.String("name", name), zap.String("listen", listen))
		return err
	}
	c.svcs = append(c.svcs, svc)
	return nil
}
