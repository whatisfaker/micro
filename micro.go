package micro

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	nacosgrpc "github.com/magicdvd/nacos-grpc"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/whatisfaker/micro/grpcpool"
	"github.com/whatisfaker/zaptrace/log"
	"github.com/whatisfaker/zaptrace/tracing"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const (
	EnvNacosAddr      = "NACOS_ADDR" //127.0.0.1:2379
	EnvConfFilePath   = "CONFIG_PATH"
	EnvLogLevel       = "LOG_LEVEL"
	EnvApplicationID  = "MS_APPLICATION_ID"
	EnvNacosConfigKey = "NACOS_CONFIG_KEY"
)

const (
	MSGroupTCPServer = "TCP_SERVER"
	MSGroupGRPC      = "GRPC"
	MSGroupWeb       = "DEFAULT_GROUP"
)

var ErrNoNacosAddr = errors.New("no addr option setting(ENV:NACOS_ADDR)")
var ErrNoFileConfigPathSet = errors.New("empty config path option setting(ENV:CONFIG_PATH)")
var ErrNoConfigKey = errors.New("no config key")

type MSManager struct {
	options      *options
	audit        *audit
	svcCenter    ServiceCenter
	confCenter   ConfigCenter
	svcs         []MicroService
	log          *log.Factory
	mysqlTracer  opentracing.Tracer
	redisTracer  opentracing.Tracer
	mongoTracer  opentracing.Tracer
	influxTracer opentracing.Tracer
}

var gMSManager *MSManager
var once sync.Once

func Manager() *MSManager {
	if gMSManager == nil {
		panic("micro service manager should be initilize first (InitMSManager)")
	}
	return gMSManager
}

//NewMSManager 创建配置管理器
func InitMSManager(opts ...Option) error {
	var err error
	once.Do(func() {
		lv := os.Getenv(EnvLogLevel)
		if lv == "" {
			lv = "info"
		}
		options := &options{
			confPath:  defaultConfPath,
			ccType:    ccTypeFile,
			scType:    scTypeNoop,
			namespace: "public",
			configKey: "go_config",
			logLevel:  lv,
			logger:    log.NewStdLogger(lv),
		}
		appID := os.Getenv(EnvApplicationID)
		if appID != "" {
			options.applicationID = appID
		} else {
			uuidObj, _ := uuid.NewRandom()
			options.applicationID = uuidObj.String()
		}
		addr := os.Getenv(EnvNacosAddr)
		if addr != "" {
			options.scType = scTypeNacos
			options.ccType = ccTypeNacos
			options.addr = addr
		}
		configKey := os.Getenv(EnvNacosConfigKey)
		if configKey != "" {
			options.configKey = configKey
		}
		//如果配置了文件路径，使用配置的文件配置中心
		fp := os.Getenv(EnvConfFilePath)
		if fp != "" {
			options.ccType = ccTypeFile
			options.confPath = fp
		}
		for _, v := range opts {
			v.apply(options)
		}
		options.logger.SetLevel(options.logLevel)
		var svcCenter ServiceCenter
		var confCenter ConfigCenter
		switch options.scType {
		case scTypeNacos:
			if len(options.addr) == 0 {
				err = ErrNoNacosAddr
				options.logger.Normal().Error("micro service manager initilize", zap.Error(err))
				return
			}
			svcCenter, err = newNacosSC(options.addr, options.namespace, options.logger.With(zap.String("srv", "nacos")))
			if err != nil {
				options.logger.Normal().Error("micro service manager initilize", zap.Error(err))
				return
			}
		case scTypeNoop:
			svcCenter = newNoopSC(options.logger.With(zap.String("srv", "noop")))
		default:
			svcCenter = newNoopSC(options.logger.With(zap.String("srv", "noop")))
		}

		switch options.ccType {
		case ccTypeNacos:
			if len(options.addr) == 0 {
				err = ErrNoNacosAddr
				options.logger.Normal().Error("micro service manager initilize", zap.Error(err))
				return
			}
			if options.configKey == "" {
				err = ErrNoConfigKey
				options.logger.Normal().Error("micro service manager initilize", zap.Error(err))
				return
			}
			confCenter, err = newNacosCC(options.addr, options.namespace, options.configKey, options.logger.With(zap.String("conf", "nacos")))
			if err != nil {
				options.logger.Normal().Error("micro service manager initilize", zap.Error(err))
				return
			}
		case ccTypeFile:
			if options.confPath == "" {
				err = ErrNoFileConfigPathSet
				return
			}
			confCenter = newFileCC(options.confPath, options.logger.With(zap.String("conf", "file")))
		default:
			confCenter = newFileCC(options.confPath, options.logger.With(zap.String("conf", "file")))
		}
		gMSManager = &MSManager{
			options: options,
			svcs:    make([]MicroService, 0),
			log:     options.logger,
			audit: &audit{
				log: options.logger.With(zap.String("audit", "audit")),
			},
			svcCenter:  svcCenter,
			confCenter: confCenter,
		}
	})
	return err
}

//ApplicationID 获取应用的唯一ID
func (c *MSManager) ApplicationID() string {
	return c.options.applicationID
}

//Register 通用注册微服务（满足MicroService接口即可)
func (c *MSManager) Register(svcs ...MicroService) {
	c.svcs = append(c.svcs, svcs...)
}

//ConfigCenter 获取配置中心
func (c *MSManager) ConfigCenter() ConfigCenter {
	return c.confCenter
}

func (c *MSManager) ServiceInstances(ctx context.Context, name string, group string) ([]*MicroServiceInfo, error) {
	return c.svcCenter.ServiceInstances(ctx, name, group)
}

//GlobalLogger 获取全局的日志管理
func (c *MSManager) GlobalLogger() *log.Factory {
	return c.log
}

//ParseConfig 解析配置文件获取对应的依赖客户端(*gorm.DB, mongo.Client, mqtt, redis等)
func (c *MSManager) ParseConfig(v interface{}, structTag ...string) (*Deps, error) {
	tag := "nacos"
	if len(structTag) > 0 {
		tag = structTag[0]
	}
	return newDeps(v, tag, c.log.With(zap.String("deps", "deps")))
}

//GetGRPCConnPool 根据服务名获取grpc的连接池
func (c *MSManager) GetGRPCConnPool(name string, opts ...grpc.DialOption) (*grpcpool.Pool, error) {
	if len(c.options.addr) == 0 {
		err := ErrNoNacosAddr
		c.log.Normal().Error("get grpc conn pool", zap.Error(err))
		return nil, err
	}
	target := nacosgrpc.Target(c.options.addr, name, nacosgrpc.OptionGroupName("GRPC"))
	return c.GetGRPCConnPoolDirect(target), nil
}

//GetGRPCConnPoolDirect 根据dial target直接获取连接池
func (c *MSManager) GetGRPCConnPoolDirect(target string, opts ...grpc.DialOption) *grpcpool.Pool {
	tracer := opentracing.GlobalTracer()
	options := make([]grpc.DialOption, 0)
	options = append(options, grpc.WithInsecure())
	if _, ok := tracer.(opentracing.NoopTracer); ok {
		options = append(options, opts...)
	} else if _, ok := tracer.(*opentracing.NoopTracer); ok {
		options = opts
	} else {
		options = append(options,
			grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)),
			grpc.WithStreamInterceptor(otgrpc.OpenTracingStreamClientInterceptor(tracer)))
		options = append(options, opts...)
	}
	grpcPool := grpcpool.NewPool(
		target,
		grpcpool.Option{MaxCap: 10, TTL: 10 * time.Minute, IdleTime: 5 * time.Minute},
		options...)
	return grpcPool
}

//Run 启动微服务
func (c *MSManager) Run(ctx context.Context, name string) error {
	return c.RunWith(ctx, name)
}

//RunWith 启动微服务伴随一些阻塞函数(mq consume, write gorutine)
func (c *MSManager) RunWith(ctx context.Context, name string, fns ...func(context.Context) error) error {
	//设置全局tracer
	tracer, closer, err := tracing.NewTracer(name, c.log)
	if err != nil {
		return err
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)
	c.mysqlTracer = tracer
	c.redisTracer = tracer
	c.influxTracer = tracer
	c.mongoTracer = tracer
	if c.options.mysqlTracer {
		tracer, closer, err := tracing.NewTracer("mysql", c.log)
		if err != nil {
			return err
		}
		defer closer.Close()
		c.mysqlTracer = tracer
	}
	if c.options.mysqlTracer {
		tracer, closer, err := tracing.NewTracer("redis", c.log)
		if err != nil {
			return err
		}
		defer closer.Close()
		c.redisTracer = tracer
	}
	if c.options.mysqlTracer {
		tracer, closer, err := tracing.NewTracer("mongo", c.log)
		if err != nil {
			return err
		}
		defer closer.Close()
		c.mongoTracer = tracer
	}
	if c.options.mysqlTracer {
		tracer, closer, err := tracing.NewTracer("influx", c.log)
		if err != nil {
			return err
		}
		defer closer.Close()
		c.influxTracer = tracer
	}
	ctx, cancel := context.WithCancel(ctx)
	grp, ctx := errgroup.WithContext(ctx)
	for i := range c.svcs {
		svc := c.svcs[i]
		grp.Go(func() error {
			err := c.svcCenter.Register(ctx, svc)
			if err != nil {
				return err
			}
			<-ctx.Done()
			err = ctx.Err()
			if err != nil {
				_ = c.svcCenter.Deregister(ctx, svc)
			}
			return err
		})
		grp.Go(func() error {
			ch := make(chan error)
			go func() {
				defer close(ch)
				ip, port := svc.Discovery()
				c.log.Trace(ctx).Info("start service", zap.String("name", svc.Name()), zap.String("ip", ip), zap.Uint("port", port))
				if err := svc.Start(ctx); err != nil {
					ch <- err
					return
				}
				ch <- nil
			}()
			select {
			case err := <-ch:
				return err
			case <-ctx.Done():
				cctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
				svc.Shutdown(cctx)
				cancel()
				return ctx.Err()
			}
		})
	}
	l := len(fns)
	for i := 0; i < l; i++ {
		fn := fns[i]
		grp.Go(func() error {
			c.log.Trace(ctx).Info("run with block function")
			return fn(ctx)
		})
	}
	grp.Go(func() error {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-signals:
			cancel()
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
	err = grp.Wait()
	//if errors.Is(err, context.Canceled) {
	if err != nil && err != context.Canceled {
		c.log.Trace(ctx).Error("micro service run", zap.Error(err))
		return err
	}
	return nil
}

func (c *MSManager) MySQLStartSpan(ctx context.Context, opName string, tags ...map[string]string) (context.Context, opentracing.Span) {
	return tracing.QuickStartSpanWithTracer(ctx, c.mysqlTracer, opName, ext.SpanKindRPCClient, tags...)
}

func (c *MSManager) RedisStartSpan(ctx context.Context, opName string, tags ...map[string]string) (context.Context, opentracing.Span) {
	return tracing.QuickStartSpanWithTracer(ctx, c.redisTracer, opName, ext.SpanKindRPCClient, tags...)
}

func (c *MSManager) MongoStartSpan(ctx context.Context, opName string, tags ...map[string]string) (context.Context, opentracing.Span) {
	return tracing.QuickStartSpanWithTracer(ctx, c.mongoTracer, opName, ext.SpanKindRPCClient, tags...)
}

func (c *MSManager) InfluxStartSpan(ctx context.Context, opName string, tags ...map[string]string) (context.Context, opentracing.Span) {
	return tracing.QuickStartSpanWithTracer(ctx, c.influxTracer, opName, ext.SpanKindRPCClient, tags...)
}

func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		return "", nil
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if localAddr.IP.To4() != nil {
		return localAddr.IP.String(), nil
	}
	return "", errors.New("no local IP")
}

func split2ipport(listen string, discoveryIP string) (string, uint, error) {
	tmp := strings.Split(listen, ":")
	if len(tmp) != 2 {
		return "", 0, fmt.Errorf("incorrect listen %s", listen)
	}
	if discoveryIP != "" {
		tmp[0] = discoveryIP
	}
	tmp[0] = strings.Trim(tmp[0], " ")
	if tmp[0] == "" {
		ip, err := getOutboundIP()
		if err != nil {
			return "", 0, err
		}
		tmp[0] = ip
	}
	port, err := strconv.Atoi(tmp[1])
	if err != nil {
		return "", 0, err
	}
	return tmp[0], uint(port), nil
}
