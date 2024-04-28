package nutclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

var errNotConnected = errors.New("not connected to the server")

type cmdResponse struct {
	data map[string]string
	err  error
}

// Client connects to a NUT server and monitors it for events.
type Client struct {
	onBattery    bool
	cfg          *Config
	ctx          context.Context
	cancel       context.CancelFunc
	requestChan  chan any
	responseChan chan cmdResponse
}

func (c *Client) runCommand(conn net.Conn, r responseReader) (cErr error) {

	// Create a goroutine to monitor the context; if told to shut down, the
	// connection is closed; otherwise use the abortChan to shutdown the
	// monitoring goroutine
	var (
		abortChan = make(chan any)
		errChan   = make(chan any)
		canceled  = false
	)
	defer func() {
		<-errChan
		if canceled {
			cErr = context.Canceled
		}
	}()
	defer close(abortChan)
	go func() {
		select {
		case <-c.ctx.Done():
			canceled = true
			conn.Close()
		case <-abortChan:
		}
		close(errChan)
	}()

	// Initialize the response reader
	r.init(conn)

	// Write the command
	if _, err := conn.Write(
		[]byte(
			fmt.Sprintf(
				"LIST VAR %s\n",
				c.cfg.getName(),
			),
		),
	); err != nil {
		cErr = err
		return
	}

	// Read the response
	if err := r.parse(); err != nil {
		cErr = err
		return
	}

	return
}

func (c *Client) processResponse(data map[string]string) {

	// Determine if the status is "on battery"
	onBattery := c.cfg.runEvaluateStatusFn(data["ups.status"])

	// If the battery status has changed, invoke the callbacks
	switch {
	case !c.onBattery && onBattery && c.cfg.PowerLostFn != nil:
		c.cfg.PowerLostFn()
	case c.onBattery && !onBattery && c.cfg.PowerRestoredFn != nil:
		c.cfg.PowerRestoredFn()
	}

	// Store status for next iteration
	c.onBattery = onBattery
}

func (c *Client) loop(conn net.Conn) error {
	var (
		now      = time.Now()
		nextPoll = now
		nextChan <-chan time.Time
	)
	for {

		var sendResponse bool

		// If a polling interval was set, initialize the timer channel to the
		// time of the next timer interval
		if c.cfg.PollInterval != 0 {
			time.After(nextPoll.Sub(now))
		}

		// Wait for:
		// - the next poll interval
		// - an explicit request to poll
		// - the client being asked to shut down
		select {
		case <-nextChan:
		case <-c.requestChan:
			sendResponse = true
		case <-c.ctx.Done():
			conn.Close()
			return context.Canceled
		}

		// Initialize the responseReader
		l := &listReader{}

		// Make the request
		if err := c.runCommand(conn, l); err != nil {
			if sendResponse {
				c.responseChan <- cmdResponse{err: err}
			}
			return err
		}

		// Process the response
		c.processResponse(l.variables)

		// Send the response if this was invoked in response to a command
		if sendResponse {
			c.responseChan <- cmdResponse{data: l.variables}
		}

		// Update the current time and next poll interval (if necessary)
		if c.cfg.PollInterval != 0 {
			now = time.Now()
			nextPoll = nextPoll.Add(c.cfg.PollInterval)
		}
	}
}

func (c *Client) lifecycle() error {

	dialer := &net.Dialer{
		Timeout: c.cfg.ReconnectInterval,
	}

	// Connect to the server
	conn, err := dialer.DialContext(c.ctx, "tcp", c.cfg.getAddr())
	if err != nil {
		return err
	}

	// Connected; invoke the callback if specified
	if c.cfg.ConnectedFn != nil {
		c.cfg.ConnectedFn()
	}

	// Run the loop until an error is encountered - either the context is
	// canceled or the client was disconnected
	err = c.loop(conn)
	if err != context.Canceled && c.cfg.DisconnectedFn != nil {
		c.cfg.DisconnectedFn()
	}
	return err
}

func (c *Client) run() {
	// The lifecycle for a NUT client is:
	// - attempt to connect to the server
	// - while connected, poll every [interval]
	// - if disconnected, reconnect after a few seconds

	defer close(c.responseChan)
	for {
		if err := c.lifecycle(); err == context.Canceled {
			return
		}

		// Retry the connection every 30 seconds
		select {
		case <-time.After(c.cfg.getReconnectInterval()):
		case <-c.requestChan:
			c.responseChan <- cmdResponse{err: errNotConnected}
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
			cfg:          cfg,
			ctx:          ctx,
			cancel:       cancel,
			requestChan:  make(chan any),
			responseChan: make(chan cmdResponse),
		}
	)
	go c.run()
	return c
}

// Status returns the current status of the UPS. If the command fails or the
// UPS is not connected, an error is returned. This must not be called after
// Close() is invoked.
func (c *Client) Status() (map[string]string, error) {
	c.requestChan <- nil
	v := <-c.responseChan
	return v.data, v.err
}

// Close shuts down the client. It is guaranteed that no more callbacks will be
// invoked after this method returns.
func (c *Client) Close() {
	c.cancel()
	<-c.responseChan
}
