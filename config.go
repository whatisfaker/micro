package micro

import (
	"context"
	"errors"
)

var ErrNotApplicable = errors.New("this function is not applicable")

type ConfigCenter interface {
	//SetConfig 设置配置
	SetConfig(context.Context, interface{}) error
	//RemoveConfig 移除配置
	RemoveConfig(context.Context, interface{}) error
	//GetConfig 获取配置
	GetConfig(context.Context, interface{}) error
	//GetConfigAndWatch 获取配置并监听
	//GetConfigAndWatch(context.Context, string, interface{}, func(string, string, interface{}, error)) error
}
