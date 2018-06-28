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

// Register registers a new metric
func (m *Metrics) Register(name string, value interface{}) error {
	m.RLock()
	defer m.RUnlock()

	if name == "" {
		return errors.New("Invalid metric name")
	}
	m.data[name] = value
	return nil
}

// ToJSON serializes data to json
func (m *Metrics) ToJSON() []byte {
	data, _ := json.Marshal(m.data)
	return data
}
