# Solution Steps

1. Create a new package directory called 'pool' and within it create two files: conn.go and pool.go.

2. In pool/conn.go, define a Connection struct that simulates a TCP connection: give it a unique ID, a closed flag, HealthCheck, Close, DoWork, IsClosed, and ID methods. HealthCheck should simulate a 10% chance to become unhealthy.

3. In pool/pool.go, define the Pool struct with a semaphore channel to manage the maximum number of live connections, a mutex-protected slice for idle connections, an atomic closed flag, a WaitGroup tracking active borrows, a shutdown channel, and a sync.Once for cleanup.

4. In Pool's NewPool constructor, initialize all fields, set the maxConns (default 10), and start a background goroutine for health checks on idle connections every 5 seconds.

5. Implement the Acquire() method: check if the pool is closed, then try to acquire a slot from the semaphore; if possible, pop an idle connection or create a new one, track in-use connections via WaitGroup.Add(1), and return the connection.

6. Implement the Release() method: always call WaitGroup.Done() on exit; if the connection is closed or fails HealthCheck, close and drop it; otherwise, if the pool is not closed, return to idle (append under lock); always release the semaphore slot.

7. Implement pool's background health checker: every 5 seconds, scan the idle pool for unhealthy/closed connections and close/remove them.

8. Implement the graceful draining logic in Drain(): use sync.Once to guard, set the closed flag, close shutdownCh, wait for ongoing borrows to return (WaitGroup.Wait), then close and clear all idle connections under lock.

9. Add helper method IdleCount() for demo/testing purposes.

10. In main.go, implement a client() function that in a loop tries to acquire, use, and release a connection (simulating work), printing status; run 5 goroutines of clients.

11. In main.go's main(), create a pool, launch 5 client goroutines, wait for all to finish or for SIGINT/SIGTERM, then call pool.Drain() and print idle count.

12. Test the program to observe safe concurrent access, enforcement of 10 connection limit, automatic health cleaning, and clean shutdown/drain.

