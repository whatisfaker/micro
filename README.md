# micro

微服务封装

## 初始化

```golang
//初始化
micro.InitMSManager(opts ...Option)
//获取单例子
micro.Manager()
```

InitMSManager 参数

| 参数             | 描述                     |
| ---------------- | ------------------------ |
| NameSpace        | 命名空间(etcd中root dir) |
| FileConfigCenter | 使用本地文件配置         |
| NacosAddr        | 配置Nacos 单体地址       |
| LogLevel         | 日志等级                 |
| Logger           | 自定义日志               |

环境变量（优先级低于参数传入)

```golang
NACOS_ADDR //127.0.0.1:8848
NACOS_CONFIG_KEY //config存储地址默认: go_config
CONFIG_PATH //配置文件路径 conf/test.yaml
LOG_LEVEL //日志等级(debug,info,warn,error) 默认:info
MS_APPLICATION_ID //应用ID 默认:随机UUID
```

注：如果配置了nacos,则配置中心也将使用nacos, 如果配置中心想使用文件，请配置FileConfigCenter或者环境变量CONFIG_PATH

## 配置中心

获取配置中心

```golang
micro.Manager().ConfigCenter()
```

提供方法

```golang
//SetConfig 设置配置
SetConfig(context.Context, interface{}) error
//RemoveConfig 移除配置
RemoveConfig(context.Context, interface{}) error
//GetConfig 获取配置
GetConfig(context.Context, interface{}) error
```

## 依赖项处理

根据配置文件初始化一些依赖项

```golang
micro.ParseConfig(v interface{}, structTag ...string) (*Deps, error)
```

获取依赖项（目前支持）

```golang
GetMySQL(key string) *gorm.DB
GetRedis(key string) redis.Cmdable
GetMongoDB(key string) *mongo.Client
GetInflux(key string) ifxclient.Client
GetRabbitMQ(key string) amqp.Client
```

## 服务中心

### 通用参数

| 参数              | 说明                      |
| ----------------- | ------------------------- |
| ParamEnableTracer | 打开追踪(默认:true)       |
| ParamDiscoveryIP  | 指定发现IP(默认:内网IP)   |
| ParamWeight       | 服务实例权重(默认:50)     |
| ParamMetadata     | 服务实例额外信息(默认:空) |

### 注册gin服务

特有参数

| 参数                 | 说明                            |
| -------------------- | ------------------------------- |
| ParamWebHealthCheck  | 健康检查的路径(/healthz)        |
| ParamWebValidateCN   | 使用中文校验信息(默认:true)     |
| ParamWebGinAuditFunc | 设置审计日志存储(默认:空不存储) |

```golang
//注册gin服务
RegisterGin(name string, listen string, initFunc func(*gin.Engine), params ...Param) error
```

### 注册grpc服务

```golang
//注册gin服务
RegisterGRPC(name string, listen string, initFunc func(*grpc.Server), params ...Param) error
```

### 注册tcp服务

特有参数

| 参数          | 说明     |
| ------------- | -------- |
| ParamTCPCodec | 编码方式 |
| ParamTCPRoute | 路由函数 |

```golang
//注册gin服务
RegisterTCP(name string, listen string, initFunc func(*ms.Server), params ...Param) error
```

### 注册其他服务

只要满足MicroService接口，都可以被注册

```golang
Register(svcs ...MicroService)
```

## 获取服务

获取服务列表

```golang
ServiceInstances(ctx context.Context, name string, group string) ([]*MicroServiceInfo, error)
```

## 日志

获取全局的日志

```golang
mirco.GlobalLogger()
```

## 审计日志(gin)

gin的中间件

```golang
GinAudit(name string) gin.HandlerFunc
```

## 获取GRPC连接池

```golang
//根据服务名获取GRPC连接池
GetGRPCConnPool(name string, opts ...grpc.DialOption) (*grpcpool.Pool, error)
//直接根据dial target获取GRPC连接池
GetGRPCConnPoolDirect(target string, opts ...grpc.DialOption) *grpcpool.Pool
```
