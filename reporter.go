package logstash

import (
	"log"
	"net"
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
	p           []string
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
			m.Count(name, metric.Count())

		case metrics.Gauge:
			m.Gauge(name, float64(metric.Value()))

		case metrics.GaugeFloat64:
			m.Gauge(name, metric.Value())

		case metrics.Histogram:
			ms := metric.Snapshot()
			m.Gauge(name+".count", float64(ms.Count()))
			m.Gauge(name+".max", float64(ms.Max()))
			m.Gauge(name+".min", float64(ms.Min()))
			m.Gauge(name+".mean", ms.Mean())
			m.Gauge(name+".stddev", ms.StdDev())
			m.Gauge(name+".var", ms.Variance())

			if len(r.percentiles) > 0 {
				values := ms.Percentiles(r.percentiles)
				for i, p := range r.p {
					m.Gauge(name+p, values[i])
				}
			}

		case metrics.Meter:
			ms := metric.Snapshot()
			m.Gauge(name+".count", float64(ms.Count()))
			m.Gauge(name+".rate1", ms.Rate1())
			m.Gauge(name+".rate5", ms.Rate5())
			m.Gauge(name+".rate15", ms.Rate15())
			m.Gauge(name+".mean", ms.RateMean())

		case metrics.Timer:
			ms := metric.Snapshot()
			m.Gauge(name+".count", float64(ms.Count()))
			m.Gauge(name+".max", time.Duration(ms.Max()).Seconds()*1000)
			m.Gauge(name+".min", time.Duration(ms.Min()).Seconds()*1000)
			m.Gauge(name+".mean", time.Duration(ms.Mean()).Seconds()*1000)
			m.Gauge(name+".stddev", time.Duration(ms.StdDev()).Seconds()*1000)

			if len(r.percentiles) > 0 {
				values := ms.Percentiles(r.percentiles)
				for i, p := range r.p {
					m.Gauge(name+p, time.Duration(values[i]).Seconds()*1000)
				}
			}
		}
	})
	r.Conn.Write(m.ToJSON())
	return nil
}
