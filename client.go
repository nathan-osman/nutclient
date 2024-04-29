package nutclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

var errNotConnected = errors.New("not connected to the server")

const (
	typeGet = iota
	typeList
)

type cmdRequest struct {
	cmdType int
	cmd     string
}

type cmdResponse struct {
	v   any
	err error
}

// Client connects to a NUT server and monitors it for events.
type Client struct {
	onBattery    bool
	cfg          *Config
	ctx          context.Context
	cancel       context.CancelFunc
	requestChan  chan cmdRequest
	responseChan chan cmdResponse
}

func (c *Client) runCommand(n *nutConn, cmdType int, cmd string) (v any, cErr error) {

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
			n.rwc.Close()
		case <-abortChan:
		}
		close(errChan)
	}()

	// Run the command and return the appropriate response
	switch cmdType {
	case typeGet:
		v, cErr = n.runGet(cmd)
		return
	case typeList:
		v, cErr = n.runListMap(cmd)
		return
	}

	// Should be unreachable
	return
}

func (c *Client) processResponse(v string) {

	// Determine if the status is "on battery"
	onBattery := c.cfg.runEvaluateStatusFn(v)

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

func (c *Client) loop(n *nutConn) error {
	var (
		now      = time.Now()
		nextPoll = now
		nextChan <-chan time.Time
	)
	for {
		// If a polling interval was set, initialize the timer channel to the
		// time of the next timer interval
		if c.cfg.PollInterval != 0 {
			nextChan = time.After(nextPoll.Sub(now))
		}

		// Wait for:
		// - the next poll interval
		// - an explicit request to poll
		// - the client being asked to shut down
		select {
		case <-nextChan:
			v, err := c.runCommand(
				n,
				typeGet,
				fmt.Sprintf("VAR %s ups.status", c.cfg.getName()),
			)
			if err != nil {
				n.rwc.Close()
				return err
			}
			c.processResponse(v.(string))
		case r := <-c.requestChan:
			v, err := c.runCommand(n, r.cmdType, r.cmd)
			c.responseChan <- cmdResponse{
				v:   v,
				err: err,
			}
		case <-c.ctx.Done():
			n.rwc.Close()
			return context.Canceled
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

	n := newNutConn(conn)

	// Run the loop until an error is encountered - either the context is
	// canceled or the client was disconnected
	err = c.loop(n)
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
			requestChan:  make(chan cmdRequest),
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
	c.requestChan <- cmdRequest{
		cmdType: typeList,
		cmd:     fmt.Sprintf("VAR %s", c.cfg.getName()),
	}
	v := <-c.responseChan
	return v.v.(map[string]string), v.err
}

// Close shuts down the client. It is guaranteed that no more callbacks will be
// invoked after this method returns.
func (c *Client) Close() {
	c.cancel()
	<-c.responseChan
}
