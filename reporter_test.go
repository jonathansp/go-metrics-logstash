package logstash

import (
	"net"
	"strconv"
	"strings"
	"testing"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

type UDPServer struct {
	conn *net.UDPConn
}

func newUDPServer(port int) (*UDPServer, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		return nil, err
	}
	return &UDPServer{conn}, nil
}

func (us *UDPServer) Read() (string, error) {
	buffer := make([]byte, 4096)
	_, _, err := us.conn.ReadFromUDP(buffer)
	if err != nil {
		return "", err
	}
	resizedStr := strings.Trim(string(buffer), "\x00") // Remove the empty chars at the end of the buffer
	return resizedStr, nil
}

func (us *UDPServer) Close() {
	us.conn.Close()
}

func TestFlushOnce(t *testing.T) {
	serverAddr := "localhost:1984"
	server, err := newUDPServer(1984)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	registry := metrics.NewRegistry()
	reporter, err := NewReporter(registry, serverAddr, "dummy-client")

	// Insert metrics
	metrics.GetOrRegisterCounter("test.counter", registry).Inc(6)
	metrics.GetOrRegisterCounter("test.counter", registry).Inc(2)
	metrics.GetOrRegisterGauge("test.gauge", registry).Update(2)
	metrics.GetOrRegisterGauge("test.gauge", registry).Update(3)
	metrics.GetOrRegisterGaugeFloat64("test.gaugeFloat64", registry).Update(4)
	metrics.GetOrRegisterGaugeFloat64("test.gaugeFloat64", registry).Update(5)
	sample := metrics.NewUniformSample(2)
	metrics.GetOrRegisterHistogram("test.histogram", registry, sample).Update(9)
	metrics.GetOrRegisterHistogram("test.histogram", registry, sample).Update(10)
	// TODO test meter and timer
	reporter.FlushOnce()

	received, err := server.Read()
	if err != nil {
		t.Fatal(err)
	}

	expected := `{
		"metric": "doc",
		"client":"dummy-client",
		"count":  1,
		"test.counter.count": 8,
		"test.gauge": 3,
		"test.gaugeFloat64": 5,
		"test.histogram.count": 2,
		"test.histogram.min": 9,
		"test.histogram.max": 10,
		"test.histogram.mean": 9.5,
		"test.histogram.stddev": 0.5,
		"test.histogram.var": 0.25,
		"test.histogram.p50": 9.5,
		"test.histogram.p75": 10,
		"test.histogram.p95": 10,
		"test.histogram.p99": 10,
		"test.histogram.p99_9": 10
	}`
	assert.JSONEq(t, expected, received)
}

func TestFlushOnceKeepsPreviousValues(t *testing.T) {
	serverAddr := "localhost:1984"
	server, err := newUDPServer(1984)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	registry := metrics.NewRegistry()
	reporter, err := NewReporter(registry, serverAddr, "dummy-client")

	// Insert metrics
	sample := metrics.NewUniformSample(3)
	metrics.GetOrRegisterCounter("test.counter", registry).Inc(6)
	metrics.GetOrRegisterCounter("test.counter", registry).Inc(2)
	metrics.GetOrRegisterGauge("test.gauge", registry).Update(2)
	metrics.GetOrRegisterGauge("test.gauge", registry).Update(3)
	metrics.GetOrRegisterGaugeFloat64("test.gaugeFloat64", registry).Update(4)
	metrics.GetOrRegisterGaugeFloat64("test.gaugeFloat64", registry).Update(5)
	metrics.GetOrRegisterHistogram("test.histogram", registry, sample).Update(9)
	metrics.GetOrRegisterHistogram("test.histogram", registry, sample).Update(10)
	reporter.FlushOnce()
	server.Read() // Ignore current values

	metrics.GetOrRegisterCounter("test.counter", registry).Inc(4)
	metrics.GetOrRegisterGauge("test.gauge", registry).Update(8)
	metrics.GetOrRegisterGaugeFloat64("test.gaugeFloat64", registry).Update(9)
	metrics.GetOrRegisterHistogram("test.histogram", registry, sample).Update(12)
	// TODO test meter and timer
	reporter.FlushOnce()

	received, err := server.Read()
	if err != nil {
		t.Fatal(err)
	}

	expected := `{
		"metric": "doc",
		"client":"dummy-client",
		"count":  1,
		"test.counter.count": 12,
		"test.gauge": 8,
		"test.gaugeFloat64": 9,
		"test.histogram.count": 3,
		"test.histogram.min": 9,
		"test.histogram.max": 12,
		"test.histogram.mean": 10.333333333333334,
		"test.histogram.stddev": 1.247219128924647,
		"test.histogram.var": 1.5555555555555556,
		"test.histogram.p50": 10,
		"test.histogram.p75": 12,
		"test.histogram.p95": 12,
		"test.histogram.p99": 12,
		"test.histogram.p99_9": 12
	}`
	assert.JSONEq(t, expected, received)
}
