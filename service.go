package micro

import "context"

const (
	defaultMSWeight uint32 = 50
)

//MicroService 微服务接口定义
type MicroService interface {
	//Name 服务名
	Name() string
	//Start 启动
	Start(context.Context) error
	//Discovery 服务发现的地址:端口
	Discovery() (string, uint)
	//Group 分组
	Group() string
	//Metadata 元数据
	Metadata() map[string]interface{}
	//Weight 权重
	Weight() uint32
	//Shutdown 关闭
	Shutdown(context.Context)
}

//ServiceCenter 服务注册接口定义
type ServiceCenter interface {
	//Register 注册服务
	Register(context.Context, MicroService) error
	//Deregister 取消注册
	Deregister(context.Context, MicroService) error
}
