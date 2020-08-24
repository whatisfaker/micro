package micro

import (
	"context"

	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

type noopSC struct {
	log *log.Factory
}

var _ ServiceCenter = (*noopSC)(nil)

func newNoopSC(log *log.Factory) *noopSC {
	return &noopSC{log: log}
}

func (c *noopSC) Register(ctx context.Context, svc MicroService) error {
	ip, port := svc.Discovery()
	c.log.Trace(ctx).Debug("register service", zap.String("name", svc.Name()), zap.String("ip", ip), zap.Uint("port", port), zap.Uint32("weight", svc.Weight()), zap.String("group", svc.Group()), zap.Any("metadata", svc.Metadata()))
	return nil
}

func (c *noopSC) Deregister(ctx context.Context, svc MicroService) error {
	ip, port := svc.Discovery()
	c.log.Trace(ctx).Debug("deregister service", zap.String("name", svc.Name()), zap.String("ip", ip), zap.Uint("port", port), zap.Uint32("weight", svc.Weight()), zap.String("group", svc.Group()), zap.Any("metadata", svc.Metadata()))
	return nil
}

func (c *noopSC) ServiceInstances(ctx context.Context, name string, group string) ([]*MicroServiceInfo, error) {
	return nil, nil
}
