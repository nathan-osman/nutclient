package nutclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

var statusRegexp = regexp.MustCompile(`^VAR \S+ ups.status "(.*)"$`)
var errInvalidStatus = errors.New("invalid response received from NUT server")

// Client connects to a NUT server and monitors it for events.
type Client struct {
	onBattery  bool
	cfg        *Config
	ctx        context.Context
	cancel     context.CancelFunc
	closedChan chan any
}

func (c *Client) runCommand(conn net.Conn, cmd string) (string, error) {

	// Create a goroutine to monitor the context; if told to shut down, the
	// connection is closed; otherwise use the abortChan to shutdown the
	// monitoring goroutine
	abortChan := make(chan any)
	defer close(abortChan)
	go func() {
		select {
		case <-c.ctx.Done():
			conn.Close()
		case <-abortChan:
		}
	}()

	// Write the command
	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return "", err
	}

	// Read the response
	r := bufio.NewScanner(conn)
	if ok := r.Scan(); !ok {
		return "", r.Err()
	}

	// Return the response
	return r.Text(), nil
}

func (c *Client) getStatus(conn net.Conn) (bool, error) {
	v, err := c.runCommand(conn, fmt.Sprintf("GET VAR %s ups.status", c.cfg.Name))
	if err != nil {
		return false, err
	}
	matches := statusRegexp.FindStringSubmatch(v)
	if len(matches) == 0 {
		return false, errInvalidStatus
	}
	strParts := strings.Split(matches[1], " ")
	switch strParts[0] {
	case "OL":
		return false, nil
	case "OB":
		return true, nil
	default:
		return false, errInvalidStatus
	}
}

func (c *Client) loop() error {

	// Connect to the server
	dialer := &net.Dialer{
		Timeout: c.cfg.ReconnectInterval,
	}
	conn, err := dialer.DialContext(c.ctx, "tcp", c.cfg.Addr)
	if err != nil {
		return err
	}

	// Connected; invoke the callback if specified
	if c.cfg.ConnectedFn != nil {
		c.cfg.ConnectedFn()
	}

	for {

		// Get the current power status
		onBattery, err := c.getStatus(conn)
		if err != nil {
			return err
		}

		// If status != last status, then a power change has occurred
		switch {
		case !c.onBattery && onBattery && c.cfg.PowerLostFn != nil:
			c.cfg.PowerLostFn()
		case c.onBattery && !onBattery && c.cfg.PowerRestoredFn != nil:
			c.cfg.PowerRestoredFn()
		}

		// Store status for next iteration
		c.onBattery = onBattery

		// Wait for next poll interval
		select {
		case <-time.After(c.cfg.PollInterval):
		case <-c.ctx.Done():
			conn.Close()
			return nil
		}
	}
}

func (c *Client) run() {
	// The lifecycle for a NUT client is:
	// - attempt to connect to the server
	// - while connected, poll every [interval]
	// - if disconnected, reconnect after a few seconds

	defer close(c.closedChan)
	for {
		if err := c.loop(); err == nil || err == context.Canceled {
			return
		} else {
			if c.cfg.DisconnectedFn != nil {
				c.cfg.DisconnectedFn()
			}
		}

		// Retry the connection every 30 seconds
		select {
		case <-time.After(c.cfg.ReconnectInterval):
		case <-c.ctx.Done():
			return
		}
	}
}

// New creates a new Client instance for the specified server.
func New(cfg *Config) *Client {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		c           = &Client{
			cfg:        cfg,
			ctx:        ctx,
			cancel:     cancel,
			closedChan: make(chan any),
		}
	)
	go c.run()
	return c
}

// Close shuts down the client. It is guaranteed that no more callbacks will be
// invoked after this method returns.
func (c *Client) Close() {
	c.cancel()
	<-c.closedChan
}
