package logstash

import (
	"encoding/json"
	"errors"
)

// Metrics represents a metric that will be sent to logstash
type Metrics struct {
	data map[string]interface{}
}

// NewMetrics Metric{} constructor
func NewMetrics(metric string) *Metrics {
	return &Metrics{
		data: map[string]interface{}{
			"metric": metric,
			"count":  1,
		},
	}
}

func (m *Metrics) register(name string, value interface{}) error {
	if name == "" {
		return errors.New("Invalid metric name")
	}
	m.data[name] = value
	return nil
}

// Gauge register a new gauge metric
func (m *Metrics) Gauge(name string, value interface{}) error {
	return m.register(name+".gauge", value)

}

// Count register a new gaugeFloat64 metric
func (m *Metrics) Count(name string, value int64) error {
	return m.register(name, value)

}

// ToJSON serializes data to json
func (m *Metrics) ToJSON() []byte {
	data, _ := json.Marshal(m.data)
	data = append(data, "\n"...)
	return data
}
