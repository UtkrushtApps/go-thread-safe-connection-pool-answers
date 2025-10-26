// main.go
package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-thread-safe-connection-pool/pool"
)

func client(id int, cp *pool.Pool, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < 3; i++ {
		conn, err := cp.Acquire()
		if err != nil {
			fmt.Printf("Client %d: failed to acquire connection: %v\n", id, err)
			return
		}
		fmt.Printf("Client %d: acquired conn %d\n", id, conn.ID())
		// Simulate work
		err = conn.DoWork()
		if err != nil {
			fmt.Printf("Client %d: error using conn: %v\n", id, err)
		}
		time.Sleep(time.Duration(rand.Intn(50)+25) * time.Millisecond)
		err = cp.Release(conn)
		if err != nil {
			fmt.Printf("Client %d: release error: %v\n", id, err)
		}
	}
	fmt.Printf("Client %d: done\n", id)
}

func main() {
	cp := pool.NewPool(10)
	var wg sync.WaitGroup
	clientCount := 5
	wg.Add(clientCount)

	for i := 0; i < clientCount; i++ {
		go client(i, cp, &wg)
	}

	// Catch ctrl-c or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("All clients completed.")
	case <-quit:
		fmt.Println("Shutdown signal caught, draining pool...")
	}

	cp.Drain()
	fmt.Printf("Drained pool. Idle count: %d\n", cp.IdleCount())
}