package micro

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	validator "github.com/go-playground/validator/v10"
	"github.com/opentracing/opentracing-go"
	"github.com/whatisfaker/gin-contrib/ginzap"
	"github.com/whatisfaker/gin-contrib/nethttp"
	"github.com/whatisfaker/gin-contrib/validatoroverriding"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

type msGin struct {
	params      *paramMap
	srv         *gin.Engine
	listen      string
	discoveryIP string
	port        uint
	name        string
	log         *log.Factory
	initFunc    func(context.Context, *gin.Engine)
	httpSrv     *http.Server
}

var _ MicroService = (*msGin)(nil)

func newGinMicroService(name string, listen string, initFunc func(context.Context, *gin.Engine), log *log.Factory, params ...Param) (*msGin, error) {
	p := &paramMap{
		webHealthCheck:  defaultHealthzPath,
		webValidateCN:   true,
		enableTracer:    true,
		ignoreTracePath: []string{defaultHealthzPath},
		metadata:        map[string]interface{}{},
		weight:          defaultMSWeight,
	}
	for _, v := range params {
		v.apply(p)
	}
	c := &msGin{
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

func (c *msGin) Start(ctx context.Context) error {
	if c.log.Level() == "debug" {
		gin.SetMode(gin.DebugMode)
		c.srv = gin.New()
		c.srv.Use(ginzap.GinzapWithConfig(c.log.ZapLogger, &ginzap.GinzapConfig{
			TimeFormat: time.RFC3339,
			UTC:        true,
			SkipPath:   c.params.ignoreTracePath,
		}))
	} else {
		gin.SetMode(gin.ReleaseMode)
		c.srv = gin.New()
	}
	if c.params.webValidateCN {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			_ = validatoroverriding.BindValidator(v)
		}
	}
	c.srv.Use(gin.Recovery())
	if c.params.enableTracer {
		tracer := opentracing.GlobalTracer()
		if len(c.params.ignoreTracePath) > 0 {
			c.srv.Use(nethttp.Middleware(tracer, nethttp.MWOmitURI(c.params.ignoreTracePath...)))
		} else {
			c.srv.Use(nethttp.Middleware(tracer))
		}
	}
	if c.params.webHealthCheck != "" {
		c.srv.GET(c.params.webHealthCheck, func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "ok")
		})
	}
	if c.initFunc != nil {
		c.initFunc(ctx, c.srv)
	}
	c.httpSrv = &http.Server{
		Addr:    c.listen,
		Handler: c.srv,
	}
	if err := c.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (c *msGin) Name() string {
	return c.name
}

func (c *msGin) Discovery() (string, uint) {
	return c.discoveryIP, c.port
}

func (c *msGin) Weight() uint32 {
	return c.params.weight
}

func (c *msGin) Group() string {
	return MSGroupWeb
}

func (c *msGin) Metadata() map[string]interface{} {
	return c.params.metadata
}

func (c *msGin) Shutdown(ctx context.Context) {
	if c.httpSrv != nil {
		_ = c.httpSrv.Shutdown(ctx)
	}
}

//RegisterGin 注册gin的http微服务
func (c *MSManager) RegisterGin(name string, listen string, initFunc func(context.Context, *gin.Engine), params ...Param) error {
	svc, err := newGinMicroService(name, listen, initFunc, c.log.With(zap.String("srv_gin", name)), params...)
	if err != nil {
		c.log.Normal().Error("register gin", zap.Error(err), zap.String("name", name), zap.String("listen", listen))
		return err
	}
	c.svcs = append(c.svcs, svc)
	return nil
}
