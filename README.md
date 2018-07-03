# go-metrics logstash

This package provides a reporter for the [go-metrics](https://github.com/rcrowley/go-metrics) library that will post the metrics to logstash. This library is based on [go-metrics-datadog](https://github.com/syntaqx/go-metrics-datadog).

## Installation

```sh
go get -u github.com/rcrowley/go-metrics
go get -u github.com/jonathansp/go-metrics-logstash
```

## Usage

```golang
package main

import (
	"log"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/jonathansp/go-metrics-logstash"
)

func main() {
	registry := metrics.NewRegistry()

	metrics.RegisterDebugGCStats(registry)
	metrics.RegisterRuntimeMemStats(registry)

	reporter, err := logstash.NewReporter(
		registry,               // go-metrics registry, or nil
		"127.0.0.1:1984",       // logstash UDP address,
		map[string]interface{}{ // default values to be sent at each flush
			"client": "my-app",
		}
	)
	if err != nil {
		log.Fatal(err)
	}

	go metrics.CaptureDebugGCStats(registry, time.Second * 5)
	go metrics.CaptureRuntimeMemStats(registry, time.Second * 5)
	go reporter.FlushEach(time.Second * 10)

}
```

## License

Distributed under the MIT license. See [LICENSE](./LICENSE) file for details.
