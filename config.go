package nutclient

import (
	"time"
)

// Config provides a set of configuration parameters for the client and
// callback functions that can be used for reacting to events.
type Config struct {

	// Addr specifies the address and port of the NUT server. If unset,
	// "localhost:3493" is assumed.
	Addr string

	// ReconnectInterval specifies the duration between attempts to reconnect
	// to the server when the connection is lost. If unset, the default is 5
	// seconds.
	ReconnectInterval time.Duration

	// KeepAliveInterval specifies how often a "keep-alive" command should be
	// sent. If unset, no keep-alive command is sent.
	KeepAliveInterval time.Duration

	// ConnectedFn is invoked every time a connection is established with the
	// server.
	ConnectedFn func()

	// DisconnectedFn is invoked every time the connection to the server is
	// lost.
	DisconnectedFn func()
}

func (c *Config) getAddr() string {
	if c.Addr == "" {
		return "localhost:3493"
	}
	return c.Addr
}

func (c *Config) getReconnectInterval() time.Duration {
	if c.ReconnectInterval == 0 {
		return 5 * time.Second
	}
	return c.ReconnectInterval
}
