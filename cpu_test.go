package loadshed

import (
	"math"
	"testing"
	"time"

	"bitbucket.org/atlassian/rolling"
)

func TestCPU(t *testing.T) {
	var points = 3
	var w = rolling.NewPointWindow(points)
	var a = rolling.NewAverageAggregator(w)
	var c = &AvgCPU{pollingInterval: time.Second, feeder: w, aggregator: a}

	for x := 0; x < points+1; x = x + 1 {
		c.feed()
	}
	var result = c.Aggregate()
	if result <= 0 || result > 100 {
		t.Fatalf("invalid AvgCPU percentage: %f", result)
	}
}

func TestCPUPolling(t *testing.T) {
	var c = NewAvgCPU(time.Millisecond, 5)
	c.feed()
	var baseline = c.Aggregate()
	var stop = make(chan bool)
	go func(stop chan bool) {
		for {
			select {
			case <-stop:
				return
			default:
				// Run some CPU bound operations to generate data
				for x := 0; x < 100; x = x + 1 {
					math.Pow(10, 1000)
				}
			}
		}
	}(stop)
	time.Sleep(3 * time.Second)
	var result = c.Aggregate()
	close(stop)
	if result <= baseline {
		t.Fatalf("AvgCPU never increased: %f - %f", baseline, result)
	}
}