package sense

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

// CompoundEye scans ports on a target.
type CompoundEye struct {
	Ports []int
}

// CommonPorts is a list of commonly used ports.
var CommonPorts = []int{
	20, 21, 22, 23, 25, 53, 80, 110, 111, 135, 139, 143, 443, 445, 993, 995,
	1723, 3306, 3389, 5900, 8080, 8443, 8000, 8888, 27017, 6379,
}

// NewCompoundEye creates a new port scanner.
// If ports is empty, uses CommonPorts.
func NewCompoundEye(ports []int) *CompoundEye {
	if len(ports) == 0 {
		ports = CommonPorts
	}
	return &CompoundEye{Ports: ports}
}

// ScanResult contains the result of a port scan.
type ScanResult struct {
	Port  int
	Open  bool
	Error error
}

// Scan performs the port scan.
func (c *CompoundEye) Scan(ctx context.Context, target string, timeout time.Duration) ([]int, error) {
	if timeout <= 0 {
		timeout = time.Second
	}

	var openPorts []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	limit := make(chan struct{}, 100) // Concurrency limit

	for _, port := range c.Ports {
		wg.Add(1)
		limit <- struct{}{}

		go func(p int) {
			defer wg.Done()
			defer func() { <-limit }()

			address := fmt.Sprintf("%s:%d", target, p)
			conn, err := net.DialTimeout("tcp", address, timeout)

			if err == nil {
				conn.Close()
				mu.Lock()
				openPorts = append(openPorts, p)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()
	sort.Ints(openPorts)
	return openPorts, nil
}
