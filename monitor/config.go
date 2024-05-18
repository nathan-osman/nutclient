package monitor

import (
	"strings"
	"time"
)

type Config struct {

	// Addr specifies the address passed to nutclient.New().
	Addr string

	// Name specifies the name of the UPS to monitor. If unset, "ups" is used.
	Name string

	// ReconnectInterval specifies the duration passed to nutclient.New().
	ReconnectInterval time.Duration

	// PollInterval specifies how often the status of the UPS should be polled.
	// If unset, polling will be done every 30 seconds.
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

	// EvaluateStatusFn is used to determine if the UPS is on (backup) battery
	// power based on the provided status. If unset, a default algorithm will
	// be used. It is recommended that you observe your UPS under different
	// conditions (line power / on battery) to determine which values your
	// model returns.
	EvaluateStatusFn func(string) bool
}

func (c *Config) getName() string {
	if c.Name == "" {
		return "ups"
	}
	return c.Name
}

func (c *Config) getPollInterval() time.Duration {
	if c.PollInterval == 0 {
		return 30 * time.Second
	}
	return c.PollInterval
}

func (c *Config) runEvaluateStatusFn(v string) bool {
	if c.EvaluateStatusFn != nil {
		return c.EvaluateStatusFn(v)
	}
	for _, p := range strings.Split(v, " ") {
		if p == "OL" {
			return false
		}
	}
	return true
}
