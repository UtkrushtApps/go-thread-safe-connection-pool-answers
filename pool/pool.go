// pool/pool.go
package pool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var ErrPoolClosed = errors.New("pool closed for new acquires")
var ErrTimeout = errors.New("timeout acquiring connection")

// Pool manages a pool of reusable mock TCP Connections

const (
	defaultMaxConnections = 10
	idleCheckInterval     = 5 * time.Second
)

type Pool struct {
	maxConns   int
	sem        chan struct{} // semaphore to limit live conns
	idleMu     sync.Mutex    // for idle slice
	idle       []*Connection
	closed     atomic.Bool
	wg         sync.WaitGroup // tracks in-use conns
	shutdownCh chan struct{}
	once       sync.Once
}

func NewPool(maxConns int) *Pool {
	if maxConns <= 0 {
		maxConns = defaultMaxConnections
	}
	p := &Pool{
		maxConns:   maxConns,
		sem:        make(chan struct{}, maxConns),
		idle:       []*Connection{},
		shutdownCh: make(chan struct{}),
	}
	go p.backgroundHealthCheck()
	return p
}

// Acquire returns a connection from idle pool or creates new if under limit.
// Returns error if pool is draining/drained.
func (p *Pool) Acquire() (*Connection, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}
	// Semaphore: block until slot available OR closed
	select {
	case p.sem <- struct{}{}:
		// Successfully acquired slot
		break
	case <-p.shutdownCh:
		return nil, ErrPoolClosed
	}

	// Try to get from idle
	p.idleMu.Lock()
	var conn *Connection
	if len(p.idle) > 0 {
		conn = p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
	}
	p.idleMu.Unlock()

	if conn == nil {
		conn = newConnection()
	}

	p.wg.Add(1)
	return conn, nil
}

// Release returns the connection to the pool if healthy, else closes.
func (p *Pool) Release(conn *Connection) error {
	defer p.wg.Done()
	if conn == nil {
		<-p.sem
		return nil
	}
	if conn.IsClosed() {
		<-p.sem
		return nil
	}
	if !conn.HealthCheck() {
		conn.Close()
		<-p.sem
		return nil
	}
	if p.closed.Load() {
		// Pool is draining: do not add to idle, just close
		conn.Close()
		<-p.sem
		return nil
	}
	// Return to idle pool
	p.idleMu.Lock()
	p.idle = append(p.idle, conn)
	p.idleMu.Unlock()
	<-p.sem
	return nil
}

func (p *Pool) backgroundHealthCheck() {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.cleanIdle()
		case <-p.shutdownCh:
			return
		}
	}
}

func (p *Pool) cleanIdle() {
	p.idleMu.Lock()
	var live []*Connection
	for _, conn := range p.idle {
		if conn.HealthCheck() && !conn.IsClosed() {
			live = append(live, conn)
		} else {
			conn.Close()
		}
	}
	p.idle = live
	p.idleMu.Unlock()
}

// Drain gracefully shuts down the pool:
// - reject new acquires
// - wait for all borrowed connections to be released
// - close all idle connections
func (p *Pool) Drain() {
	p.once.Do(func() {
		p.closed.Store(true)
		close(p.shutdownCh)
		p.wg.Wait()
		// Close all idle conns
		p.idleMu.Lock()
		for _, c := range p.idle {
			c.Close()
		}
		p.idle = nil
		p.idleMu.Unlock()
	})
}

// For demonstrating/test only
func (p *Pool) IdleCount() int {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()
	return len(p.idle)
}
