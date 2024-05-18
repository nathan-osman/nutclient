package monitor

import (
	"fmt"
	"time"

	"github.com/nathan-osman/nutclient/v3"
)

// Monitor watches a UPS server for power events.
type Monitor struct {
	onBattery bool
	cfg       *Config
	client    *nutclient.Client
	connChan  chan bool
}

func (m *Monitor) processResponse(v string) {

	// Determine if the status is "on battery"
	onBattery := m.cfg.runEvaluateStatusFn(v)

	// If the battery status has changed, invoke the callbacks
	switch {
	case !m.onBattery && onBattery && m.cfg.PowerLostFn != nil:
		m.cfg.PowerLostFn()
	case m.onBattery && !onBattery && m.cfg.PowerRestoredFn != nil:
		m.cfg.PowerRestoredFn()
	}

	// Store status for next iteration
	m.onBattery = onBattery
}

func (m *Monitor) connected() {
	m.connChan <- true
	if m.cfg.ConnectedFn != nil {
		m.cfg.ConnectedFn()
	}
}

func (m *Monitor) disconnected() {
	m.connChan <- false
	if m.cfg.DisconnectedFn != nil {
		m.cfg.DisconnectedFn()
	}
}

func (m *Monitor) run() {
	var (
		connected bool
		nextChan  <-chan time.Time
	)
	for {
		if connected {
			if v, err := m.client.Get(
				fmt.Sprintf("VAR %s ups.status", m.cfg.getName()),
			); err == nil {
				m.processResponse(v)
			}
			nextChan = time.After(m.cfg.getPollInterval())
		}
		select {
		case <-nextChan:
		case v, ok := <-m.connChan:
			if !ok {
				return
			}
			connected = v
			if !connected {
				nextChan = nil
			}
		}
	}
}

// New creates a new Monitor instance.
func New(cfg *Config) *Monitor {
	if cfg == nil {
		cfg = &Config{}
	}
	m := &Monitor{
		cfg:      cfg,
		connChan: make(chan bool),
	}
	m.client = nutclient.New(
		&nutclient.Config{
			Addr:              cfg.Addr,
			ReconnectInterval: cfg.ReconnectInterval,
			ConnectedFn:       m.connected,
			DisconnectedFn:    m.disconnected,
		},
	)
	go m.run()
	return m
}

// Close shuts down the monitor.
func (m *Monitor) Close() {
	m.client.Close()
	close(m.connChan)
}
