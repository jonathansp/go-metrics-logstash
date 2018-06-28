package logstash

import (
	"encoding/json"
	"errors"
	"sync"
)

// Metrics represents a metric that will be sent to logstash
type Metrics struct {
	data map[string]interface{}
	name string
	sync.RWMutex
}

// NewMetrics Metric{} constructor
func NewMetrics(name string) *Metrics {
	return &Metrics{
		data: new(name),
	}
}

func new(name string) map[string]interface{} {
	return map[string]interface{}{
		"metric": "doc",
		"client": name,
		"count":  1,
	}
}

func (m *Metrics) register(name string, value interface{}) error {
	m.RLock()
	defer m.RUnlock()

	if name == "" {
		return errors.New("Invalid metric name")
	}
	m.data[name] = value
	return nil
}

// Gauge register a new gauge metric
func (m *Metrics) Gauge(name string, value interface{}) error {
	return m.register(name, value)
}

// Count register a new gaugeFloat64 metric
func (m *Metrics) Count(name string, value int64) error {
	return m.register(name, value)
}

// ToJSON serializes data to json
func (m *Metrics) ToJSON() []byte {
	data, _ := json.Marshal(m.data)
	return data
}
