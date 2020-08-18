package micro

import (
	"context"
	"net"

	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type msGRPC struct {
	params      *paramMap
	srv         *grpc.Server
	listen      string
	discoveryIP string
	port        uint
	name        string
	log         *log.Factory
	initFunc    func(context.Context, *grpc.Server)
}

var _ MicroService = (*msGRPC)(nil)

func newGRPCMicroService(name string, listen string, initFunc func(context.Context, *grpc.Server), log *log.Factory, params ...Param) (*msGRPC, error) {
	p := &paramMap{
		enableTracer: true,
		metadata:     map[string]interface{}{},
		weight:       defaultMSWeight,
	}
	for _, v := range params {
		v.apply(p)
	}
	var err error
	c := &msGRPC{
		params:   p,
		initFunc: initFunc,
		name:     name,
		listen:   listen,
		log:      log,
	}
	c.discoveryIP, c.port, err = split2ipport(listen, p.discoveryIP)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *msGRPC) Start(ctx context.Context) error {
	grpcListen, err := net.Listen("tcp", c.listen)
	if err != nil {
		return err
	}
	if c.params.enableTracer {
		tracer := opentracing.GlobalTracer()
		c.srv = grpc.NewServer(
			grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
			grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)))
	} else {
		c.srv = grpc.NewServer()
	}
	return c.srv.Serve(grpcListen)
}

func (c *msGRPC) Name() string {
	return c.name
}

func (c *msGRPC) Discovery() (string, uint) {
	return c.discoveryIP, c.port
}

func (c *msGRPC) Weight() uint32 {
	return c.params.weight
}

func (c *msGRPC) Group() string {
	return "GRPC"
}

func (c *msGRPC) Metadata() map[string]interface{} {
	return c.params.metadata
}

func (c *msGRPC) Shutdown(ctx context.Context) {
	if c.srv != nil {
		c.srv.GracefulStop()
	}
}

//RegisterGRPC 注册grpc的微服务
func (c *MSManager) RegisterGRPC(name string, listen string, initFunc func(context.Context, *grpc.Server), params ...Param) error {
	svc, err := newGRPCMicroService(name, listen, initFunc, c.log.With(zap.String("srv_grpc", name)), params...)
	if err != nil {
		c.log.Normal().Error("register grpc", zap.Error(err), zap.String("name", name), zap.String("listen", listen))
		return err
	}
	c.svcs = append(c.svcs, svc)
	return nil
}
