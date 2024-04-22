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

	// Name specifies the name of the UPS to monitor. If unset, "ups" is used.
	Name string

	// ReconnectInterval specifies the duration between attempts to reconnect
	// to the server when the connection is lost. If unset, the default is 30
	// seconds.
	ReconnectInterval time.Duration

	// PollInterval specifies how often the status of the UPS should be polled.
	// If unset, the default is 5 seconds.
	PollInterval time.Duration

	// ConnectedFn is invoked every time a connection is established with the
	// server.
	ConnectedFn func()

	// DisconnectedFn is invoked every time the connection to the server is
	// lost.
	DisconnectedFn func()

	// PowerLostFn is invoked every time line power is disconnected.
	PowerLostFn func()

	// PowerRestoredFn is invoked every time line power is restored.
	PowerRestoredFn func()
}

func (c *Config) getAddr() string {
	if c.Addr == "" {
		return "localhost:3493"
	}
	return c.Addr
}

func (c *Config) getName() string {
	if c.Name == "" {
		return "ups"
	}
	return c.Name
}

func (c *Config) getReconnectInterval() time.Duration {
	if c.ReconnectInterval == 0 {
		return 30 * time.Second
	}
	return c.ReconnectInterval
}

func (c *Config) getPollInterval() time.Duration {
	if c.PollInterval == 0 {
		return 5 * time.Second
	}
	return c.PollInterval
}
