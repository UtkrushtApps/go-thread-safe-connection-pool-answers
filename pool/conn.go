// pool/conn.go
package pool

import (
	"errors"
	"math/rand"
	"sync/atomic"
	"time"
)

var connIDCounter int64

// Connection represents a mock TCP connection
// In real code, this would wrap the actual net.Conn/other connection
// For healthcheck, we will simulate random failures

// Connection simulates a TCP connection
// It is not exported outside this package

type Connection struct {
	id    int64
	closed atomic.Bool
}

func newConnection() *Connection {
	id := atomic.AddInt64(&connIDCounter, 1)
	return &Connection{id: id}
}

func (c *Connection) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		return nil
	}
	return errors.New("already closed")
}

func (c *Connection) IsClosed() bool {
	return c.closed.Load()
}

// Simulate a health check: randomly unhealthy
func (c *Connection) HealthCheck() bool {
	if c.IsClosed() {
		return false
	}
	// 90% chance healthy (for demo), 10% unhealthy
	return rand.Float32() > 0.10
}

func (c *Connection) DoWork() error {
	if c.IsClosed() {
		return errors.New("cannot use closed connection")
	}
	// Simulate some work
	// (in real code, TCP communication would happen)
	time.Sleep(time.Duration(rand.Intn(40)+10) * time.Millisecond)
	return nil
}

func (c *Connection) ID() int64 {
	return c.id
}
