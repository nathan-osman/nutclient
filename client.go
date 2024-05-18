package nutclient

import (
	"context"
	"errors"
	"net"
	"time"
)

var errNotConnected = errors.New("not connected to the server")

const (
	typeGet = iota
	typeList
	typeCmd
)

type cmdRequest struct {
	cmdType int
	cmd     string
	args    []string
}

type cmdResponse struct {
	v   any
	err error
}

// Client connects to a NUT server and monitors it for events.
type Client struct {
	cfg          *Config
	ctx          context.Context
	cancel       context.CancelFunc
	requestChan  chan cmdRequest
	responseChan chan cmdResponse
}

func (c *Client) runCommand(
	conn net.Conn,
	n *nutConn,
	cmdType int,
	cmd string,
	args []string,
) (v any, cErr error) {

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

	// Run the command and return the appropriate response
	switch cmdType {
	case typeGet:
		v, cErr = n.runGet(args)
		return
	case typeList:
		v, cErr = n.runList(args)
		return
	case typeCmd:
		v, cErr = n.runCmd(cmd, args)
		return
	}

	// Should be unreachable
	return
}

func (c *Client) loop(conn net.Conn, n *nutConn) error {
	var keepAliveTicker *time.Ticker
	if c.cfg.KeepAliveInterval != 0 {
		keepAliveTicker = time.NewTicker(c.cfg.KeepAliveInterval)
		defer keepAliveTicker.Stop()
	}
	for {
		var keepAliveChan <-chan time.Time
		if keepAliveTicker != nil {
			keepAliveChan = keepAliveTicker.C
		}
		select {
		case <-keepAliveChan:
			_, err := c.runCommand(conn, n, typeCmd, "HELP", nil)
			if err != nil {
				return err
			}
		case r := <-c.requestChan:
			v, err := c.runCommand(conn, n, r.cmdType, r.cmd, r.args)
			c.responseChan <- cmdResponse{
				v:   v,
				err: err,
			}
			if err != nil {
				return err
			}
			if keepAliveTicker != nil {
				keepAliveTicker.Reset(c.cfg.KeepAliveInterval)
			}
		case <-c.ctx.Done():
			conn.Close()
			return context.Canceled
		}
	}
}

func (c *Client) lifecycle() error {

	dialer := &net.Dialer{
		Timeout: c.cfg.getReconnectInterval(),
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
	err = c.loop(conn, n)
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

		// Retry the connection every [reconnect interval] seconds
		select {
		case <-time.After(c.cfg.getReconnectInterval()):
		case <-c.requestChan:
			c.responseChan <- cmdResponse{err: errNotConnected}
		case <-c.ctx.Done():
			return
		}
	}
}

// New creates a new Client instance for the specified server with the
// specified configuration. If cfg is nil, the default configuration is used.
func New(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}
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

// Get runs a GET command on the server. The provided arguments are appended to
// the GET command.
func (c *Client) Get(args ...string) (string, error) {
	c.requestChan <- cmdRequest{
		cmdType: typeGet,
		args:    args,
	}
	r := <-c.responseChan
	return r.v.(string), r.err
}

// Close shuts down the client. It is guaranteed that no more callbacks will be
// invoked after this method returns.
func (c *Client) Close() {
	c.cancel()
	<-c.responseChan
}
