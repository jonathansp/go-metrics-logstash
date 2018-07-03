package logstash

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

// Reporter represents a metrics registry.
type Reporter struct {
	// Registry map is used to hold metrics that will be sent to logstash.
	Registry metrics.Registry
	// Conn is a UDP connection to logstash.
	Conn *net.UDPConn
	// Name of this reporter
	Name    string
	Version string

	percentiles []float64
	udpAddr     *net.UDPAddr
}

// NewReporter creates a new Reporter with a pre-configured statsd client.
func NewReporter(r metrics.Registry, addr string, name string) (*Reporter, error) {
	if r == nil {
		r = metrics.DefaultRegistry
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		Conn:     conn,
		Registry: r,
		Name:     name,
		Version:  "0.1.1",

		udpAddr:     udpAddr,
		percentiles: []float64{0.50, 0.75, 0.95, 0.99, 0.999},
	}, nil
}

// FlushEach is a blocking exporter function which reports metrics in the registry.
// Designed to be used in a goroutine: go reporter.Flush()
func (r *Reporter) FlushEach(interval time.Duration) {
	defer func() {
		if rec := recover(); rec != nil {
			handlePanic(rec)
		}
	}()

	for range time.Tick(interval) {
		if err := r.FlushOnce(); err != nil {
			log.Println(err)
		}
	}
}

// FlushOnce submits a snapshot of the registry.
func (r *Reporter) FlushOnce() error {
	m := NewMetrics(r.Name)

	r.Registry.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			m.Register(fmt.Sprintf("%s.count", name), metric.Count())

		case metrics.Gauge:
			m.Register(name, float64(metric.Value()))

		case metrics.GaugeFloat64:
			m.Register(name, metric.Value())

		case metrics.Histogram:
			ms := metric.Snapshot()
			m.Register(fmt.Sprintf("%s.count", name), float64(ms.Count()))
			m.Register(fmt.Sprintf("%s.max", name), float64(ms.Max()))
			m.Register(fmt.Sprintf("%s.min", name), float64(ms.Min()))
			m.Register(fmt.Sprintf("%s.mean", name), ms.Mean())
			m.Register(fmt.Sprintf("%s.stddev", name), ms.StdDev())
			m.Register(fmt.Sprintf("%s.var", name), ms.Variance())

			for _, p := range r.percentiles {
				pStr := strings.Replace(fmt.Sprintf("p%g", p*100), ".", "_", -1)
				m.Register(fmt.Sprintf("%s.%s", name, pStr), ms.Percentile(p))
			}

		case metrics.Meter:
			ms := metric.Snapshot()
			m.Register(fmt.Sprintf("%s.count", name), float64(ms.Count()))
			m.Register(fmt.Sprintf("%s.rate1", name), ms.Rate1())
			m.Register(fmt.Sprintf("%s.rate5", name), ms.Rate5())
			m.Register(fmt.Sprintf("%s.rate15", name), ms.Rate15())
			m.Register(fmt.Sprintf("%s.mean", name), ms.RateMean())

		case metrics.Timer:
			ms := metric.Snapshot()
			m.Register(fmt.Sprintf("%s.count", name), float64(ms.Count()))
			m.Register(fmt.Sprintf("%s.max", name), time.Duration(ms.Max()).Seconds()*1000)
			m.Register(fmt.Sprintf("%s.min", name), time.Duration(ms.Min()).Seconds()*1000)
			m.Register(fmt.Sprintf("%s.mean", name), time.Duration(ms.Mean()).Seconds()*1000)
			m.Register(fmt.Sprintf("%s.stddev", name), time.Duration(ms.StdDev()).Seconds()*1000)

			for _, p := range r.percentiles {
				duration := time.Duration(ms.Percentile(p)).Seconds() * 1000
				m.Register(fmt.Sprintf("%s.p%g", name, p*100), duration)
			}
		}
	})
	r.Conn.Write(m.ToJSON())
	return nil
}
