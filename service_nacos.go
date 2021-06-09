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
	client, err := nacos.NewServiceClient(addr, nacos.DefaultNameSpaceID(namespace), nacos.Log(NewZapLogger(log)), nacos.LogLevel(log.Level()))
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
	c.log.Trace(ctx).Debug("register service", zap.String("name", svc.Name()), zap.String("ip", ip), zap.Uint("port", port), zap.Uint32("weight", svc.Weight()), zap.String("group", svc.Group()), zap.Any("metadata", svc.Metadata()))
	err := c.client.RegisterInstance(ip, port, svc.Name(), nacos.ParamWeight(float64(svc.Weight())), nacos.ParamMetadata(svc.Metadata()), nacos.ParamGroupName(svc.Group()))
	if err != nil {
		return err
	}
	ch := c.client.HeartBeatErr()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-ch:
			if !ok {
				return nil
			}
			return err
		}
	}
}

func (c *nacosSC) Deregister(ctx context.Context, svc MicroService) error {
	ip, port := svc.Discovery()
	c.log.Trace(ctx).Debug("deregister service", zap.String("name", svc.Name()), zap.String("ip", ip), zap.Uint("port", port), zap.Uint32("weight", svc.Weight()), zap.String("group", svc.Group()), zap.Any("metadata", svc.Metadata()))
	return c.client.DeregisterInstance(ip, port, svc.Name(), nacos.ParamWeight(float64(svc.Weight())), nacos.ParamMetadata(svc.Metadata()), nacos.ParamGroupName(svc.Group()))
}

func (c *nacosSC) ServiceInstances(ctx context.Context, name string, group string) ([]*MicroServiceInfo, error) {
	svc, err := c.client.GetService(name, true, nacos.ParamGroupName(group))
	if err != nil {
		return nil, err
	}
	instances := make([]*MicroServiceInfo, 0)
	for _, v := range svc.Instances {
		instances = append(instances, &MicroServiceInfo{
			IP:       v.Ip,
			Port:     uint(v.Port),
			Name:     name,
			Weight:   uint32(v.Weight),
			Metadata: v.Metadata,
			Group:    group,
		})
	}
	return instances, nil
}
