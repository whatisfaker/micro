package micro

import (
	"context"

	"github.com/magicdvd/nacos-client"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

type nacosSC struct {
	client nacos.ServiceCmdable
	log    *log.Factory
}

var _ ServiceCenter = (*nacosSC)(nil)

func newNacosSC(addr string, namespace string, log *log.Factory) (*nacosSC, error) {
	client, err := nacos.NewServiceClient(addr, nacos.DefaultNameSpaceID(namespace))
	if err != nil {
		return nil, err
	}
	return &nacosSC{
		client: client,
		log:    log,
	}, nil
}

func (c *nacosSC) Register(ctx context.Context, svc MicroService) error {
	ip, port := svc.Discovery()
	// c.log.Trace(ctx).Debug("register service", zap.String("name", svc.Name()), zap.String("ip", ip), zap.Uint("port", port), zap.Uint32("weight", svc.Weight()), zap.String("group", svc.Group()), zap.Any("metadata", svc.Metadata()))
	// return c.client.RegisterService(svc.Name(), svc.Group(), ip, port, svc.Weight(), svc.Metadata(), defaultTTL, defaultService)
	return c.client.RegisterInstance(ip, port, svc.Name(), nacos.ParamWeight(float64(svc.Weight())), nacos.ParamMetadata(svc.Metadata()), nacos.ParamGroupName(svc.Group()))
}

func (c *nacosSC) Deregister(ctx context.Context, svc MicroService) error {
	ip, port := svc.Discovery()
	c.log.Trace(ctx).Debug("deregister service", zap.String("name", svc.Name()), zap.String("ip", ip), zap.Uint("port", port), zap.Uint32("weight", svc.Weight()), zap.String("group", svc.Group()), zap.Any("metadata", svc.Metadata()))
	return c.client.DeregisterInstance(ip, port, svc.Name(), nacos.ParamWeight(float64(svc.Weight())), nacos.ParamMetadata(svc.Metadata()), nacos.ParamGroupName(svc.Group()))
}
