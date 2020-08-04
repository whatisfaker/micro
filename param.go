package micro

import "github.com/whatisfaker/ms/codec"

const (
	defaultHealthzPath = "/healthz"
)

type paramMap struct {
	webHealthCheck  string
	webValidateCN   bool
	enableTracer    bool
	ignoreTracePath []string
	discoveryIP     string
	weight          uint32
	metadata        map[string]interface{}
	tcpCodec        codec.Codec
	tcpRoute        func([]byte) int
}

type Param interface {
	apply(*paramMap)
}

type param struct {
	f func(*paramMap)
}

func (c *param) apply(m *paramMap) {
	c.f(m)
}

func newParam(f func(*paramMap)) Param {
	return &param{
		f: f,
	}
}

//ParamEnableTracer 使用服务追踪(默认开)
func ParamEnableTracer(b bool, ignoreURIs ...string) Param {
	return newParam(func(m *paramMap) {
		m.enableTracer = b
		m.ignoreTracePath = ignoreURIs
	})
}

//ParamDiscoveryIP 指定服务发现要使用的IP(默认内网IP)
func ParamDiscoveryIP(b string) Param {
	return newParam(func(m *paramMap) {
		m.discoveryIP = b
	})
}

//ParamWeight 服务实例的被调用权重（默认50）
func ParamWeight(w uint32) Param {
	return newParam(func(m *paramMap) {
		m.weight = w
	})
}

//ParamMetadata 服务的元数据（默认空）
func ParamMetadata(mm map[string]interface{}) Param {
	return newParam(func(m *paramMap) {
		m.metadata = mm
	})
}

//ParamWebHealthCheck web服务打开健康检查的路由（默认打开，路径为/healthz)
func ParamWebHealthCheck(enable bool, path ...string) Param {
	return newParam(func(m *paramMap) {
		if enable {
			if len(path) > 0 {
				m.webHealthCheck = path[0]
			}
			if m.webHealthCheck == "" {
				m.webHealthCheck = defaultHealthzPath
			}
		} else {
			m.webHealthCheck = ""
		}
	})
}

//ParamWebValidateCN web服务国际化使用中文(默认开)
func ParamWebValidateCN(enable bool) Param {
	return newParam(func(m *paramMap) {
		m.webValidateCN = enable
	})
}

func ParamTCPCodec(codec codec.Codec) Param {
	return newParam(func(m *paramMap) {
		m.tcpCodec = codec
	})
}

func ParamTCPRoute(fn func([]byte) int) Param {
	return newParam(func(m *paramMap) {
		m.tcpRoute = fn
	})
}
