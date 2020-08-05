package micro

import (
	"context"
	"net"

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
	initFunc    func(*ms.Server)
}

var _ MicroService = (*msTCP)(nil)

func newTCPMicroService(name string, listen string, initFunc func(*ms.Server), log *log.Factory, params ...Param) (*msTCP, error) {
	p := &paramMap{
		metadata: map[string]interface{}{},
		weight:   defaultMSWeight,
		tcpCodec: nil,
		tcpRoute: nil,
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
	c.srv = ms.NewServer(opts...)
	c.initFunc(c.srv)
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
	return "TCP_SERVER"
}

func (c *msTCP) Metadata() map[string]interface{} {
	return c.params.metadata
}

func (c *msTCP) Shutdown(ctx context.Context) {
	if c.srv != nil {
		_ = c.srv.Shutdown(ctx)
	}
}
