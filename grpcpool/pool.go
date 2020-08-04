package grpcpool

import (
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
)

var (
	errClosed = errors.New("grpc conn is closed")
)

type Pool struct {
	addr        string
	connCh      chan *ClientConn
	closed      bool
	grpcoptions []grpc.DialOption
	option      Option
	lock        sync.RWMutex
}

type Option struct {
	MaxCap   int
	TTL      time.Duration
	IdleTime time.Duration
}

func NewPool(addr string, poolOption Option, options ...grpc.DialOption) *Pool {
	if poolOption.MaxCap <= 0 {
		poolOption.MaxCap = 0
	}
	if poolOption.TTL <= 0 {
		poolOption.TTL = 0
	}
	if poolOption.IdleTime <= 0 {
		poolOption.IdleTime = 0
	}
	return &Pool{
		addr:        addr,
		connCh:      make(chan *ClientConn, poolOption.MaxCap),
		closed:      false,
		grpcoptions: options,
		option:      poolOption,
	}
}

type ClientConn struct {
	*grpc.ClientConn
	t      time.Time
	u      time.Time
	Closed bool
}

func (c *Pool) Get() (*ClientConn, error) {
	select {
	case conn, ok := <-c.connCh:
		if !ok {
			return nil, errClosed
		}
		c.lock.RLock()
		defer c.lock.RUnlock()
		if c.closed {
			conn.Closed = true
			conn.Close()
			return nil, errClosed
		}
		//如果未超过最大空闲时间
		if !conn.Closed && conn.u.Add(c.option.IdleTime).After(time.Now()) {
			return conn, nil
		}
		//关闭空闲链接
		conn.Closed = true
		conn.Close()
	default:
	}
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}
	return &ClientConn{ClientConn: conn, t: time.Now(), u: time.Now()}, nil
}

func (c *Pool) Put(conn *ClientConn) {
	if conn == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		conn.Close()
		return
	}
	//生命周期未过期
	if conn.t.Add(c.option.TTL).After(time.Now()) {
		//更新空闲开始时间
		conn.u = time.Now()
		select {
		case c.connCh <- conn:
			return
		default:
		}
	}
	conn.Closed = true
	conn.Close()
}

func (c *Pool) Close() {
	if !c.closed {
		c.lock.Lock()
		defer c.lock.Unlock()
		close(c.connCh)
		c.closed = true
	}
}

func (c *Pool) newConn() (*grpc.ClientConn, error) {
	return grpc.Dial(c.addr, c.grpcoptions...)
}
