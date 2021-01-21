// Package prommongo exports pool stats of Mongo Driver as prometheus metrics collector.
package prommongo

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/event"
)

// CommandMonitorCollector extends prometheus.Collector.
type CommandMonitorCollector interface {
	prometheus.Collector

	// CommandMonitor wraps the given *event.CommandMonitor and returns *event.CommandMonitor that collects the command metrics.
	// If nil is passed, it returns an parentless *event.CommandMonitor.
	CommandMonitor(parent *event.CommandMonitor) *event.CommandMonitor
}

type commandMonitorCollector struct {
	commandDurationNs int64 // atomic

	mu sync.RWMutex

	commandDurationNsDesc *prometheus.Desc
}

var _ CommandMonitorCollector = (*commandMonitorCollector)(nil)

func fqName(name string) string {
	return "go_mongo_" + name
}

// NewCommandMonitorCollector returns a new CommandMonitorCollector.
func NewCommandMonitorCollector() CommandMonitorCollector {
	return &commandMonitorCollector{
		commandDurationNsDesc: prometheus.NewDesc(fqName("command_duration_ns"), "Elapsed time of command.", nil, nil)}
}

// Describe implements prometheus.Collector.
func (c *commandMonitorCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.commandDurationNsDesc
}

// Collect implements prometheus.Collector.
func (c *commandMonitorCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ch <- prometheus.MustNewConstMetric(c.commandDurationNsDesc, prometheus.GaugeValue, float64(c.commandDurationNs))
}

// CommandMonitor implements CommandMonitor.
func (c *commandMonitorCollector) CommandMonitor(parent *event.CommandMonitor) *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(ctx context.Context, e *event.CommandStartedEvent) {
			if parent != nil {
				parent.Started(ctx, e)
			}
		},
		Succeeded: func(ctx context.Context, e *event.CommandSucceededEvent) {
			if parent != nil {
				parent.Succeeded(ctx, e)
			}
			c.updateCommandStats(e.CommandFinishedEvent)
		},
		Failed: func(ctx context.Context, e *event.CommandFailedEvent) {
			if parent != nil {
				parent.Failed(ctx, e)
			}
			c.updateCommandStats(e.CommandFinishedEvent)
		},
	}
}

func (c *commandMonitorCollector) updateCommandStats(e event.CommandFinishedEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.commandDurationNs = e.DurationNanos
}

// CommandMonitorCollector extends prometheus.Collector.
type PoolMonitorCollector interface {
	prometheus.Collector

	// PoolMonitor wraps the given *event.PoolMonitor and returns *event.PoolMonitor that collects the command metrics.
	// If nil is passed, it returns an parentless *event.PoolMonitor.
	PoolMonitor(parent *event.PoolMonitor) *event.PoolMonitor
}

type poolMonitorCollector struct {
	connClosed, poolCreated, connCreated, getFailed, getSucceded, connReturned, poolCleard, poolClosed int64

	maxPoolSize, minPoolSize, waitQueueTimeoutMS uint64

	mu sync.RWMutex

	connClosedDesc, poolCreatedDesc, connCreatedDesc, getFailedDesc, getSuccededDesc, connReturnedDesc, poolClearedDesc, poolClosedDec *prometheus.Desc

	maxPoolSizeDesc, minPoolSizeDesc, waitQueueTimeoutMSDesc *prometheus.Desc
}

var _ PoolMonitorCollector = (*poolMonitorCollector)(nil)

// NewCommandMonitorCollector returns a new CommandMonitorCollector.
func NewPoolMonitorCollector() PoolMonitorCollector {
	return &poolMonitorCollector{
		connClosedDesc:         prometheus.NewDesc(fqName("connection_closed"), "The total number of closed connection events.", nil, nil),
		poolCreatedDesc:        prometheus.NewDesc(fqName("pool_created"), "The total number of pool created events.", nil, nil),
		connCreatedDesc:        prometheus.NewDesc(fqName("connection_created"), "The total number of connection created events.", nil, nil),
		getFailedDesc:          prometheus.NewDesc(fqName("get_failed"), "The total number of connection checkout failed events.", nil, nil),
		getSuccededDesc:        prometheus.NewDesc(fqName("get_succeeded"), "The total number of connection checkedout events.", nil, nil),
		connReturnedDesc:       prometheus.NewDesc(fqName("connection_returnd"), "The total number of connection checkedin events.", nil, nil),
		poolClearedDesc:        prometheus.NewDesc(fqName("pool_cleared"), "The total number of connection pool cleared events", nil, nil),
		poolClosedDec:          prometheus.NewDesc(fqName("pool_closed"), "The total number of connection pool closed events", nil, nil),
		maxPoolSizeDesc:        prometheus.NewDesc(fqName("max_pool_size"), "The maximum number of connections allowed in the driver's connection pool to each server.", nil, nil),
		minPoolSizeDesc:        prometheus.NewDesc(fqName("min_pool_size"), "The minimum number of connections allowed in the driver's connection pool to each server.", nil, nil),
		waitQueueTimeoutMSDesc: prometheus.NewDesc(fqName("wait_queue_timeout_ms"), "The maximum amount of time a thread can wait for a connection to become available.", nil, nil),
	}

}

// Describe implements prometheus.Collector.
func (c *poolMonitorCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.connClosedDesc
	ch <- c.poolCreatedDesc
	ch <- c.connCreatedDesc
	ch <- c.getFailedDesc
	ch <- c.getSuccededDesc
	ch <- c.connReturnedDesc
	ch <- c.poolClearedDesc
	ch <- c.poolClosedDec
	ch <- c.maxPoolSizeDesc
	ch <- c.minPoolSizeDesc
	ch <- c.waitQueueTimeoutMSDesc
}

// Collect implements prometheus.Collector.
func (c *poolMonitorCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ch <- prometheus.MustNewConstMetric(c.connClosedDesc, prometheus.CounterValue, float64(c.connClosed))
	ch <- prometheus.MustNewConstMetric(c.poolCreatedDesc, prometheus.CounterValue, float64(c.poolCreated))
	ch <- prometheus.MustNewConstMetric(c.connCreatedDesc, prometheus.CounterValue, float64(c.connCreated))
	ch <- prometheus.MustNewConstMetric(c.getFailedDesc, prometheus.CounterValue, float64(c.getFailed))
	ch <- prometheus.MustNewConstMetric(c.getSuccededDesc, prometheus.CounterValue, float64(c.getSucceded))
	ch <- prometheus.MustNewConstMetric(c.connReturnedDesc, prometheus.CounterValue, float64(c.connReturned))
	ch <- prometheus.MustNewConstMetric(c.poolClearedDesc, prometheus.CounterValue, float64(c.poolCleard))
	ch <- prometheus.MustNewConstMetric(c.poolClosedDec, prometheus.CounterValue, float64(c.poolClosed))

	ch <- prometheus.MustNewConstMetric(c.maxPoolSizeDesc, prometheus.GaugeValue, float64(c.maxPoolSize))
	ch <- prometheus.MustNewConstMetric(c.minPoolSizeDesc, prometheus.GaugeValue, float64(c.minPoolSize))
	ch <- prometheus.MustNewConstMetric(c.waitQueueTimeoutMSDesc, prometheus.GaugeValue, float64(c.waitQueueTimeoutMS))
}

// PoolMonitor implements PoolMonitor.
func (c *poolMonitorCollector) PoolMonitor(parent *event.PoolMonitor) *event.PoolMonitor {
	return &event.PoolMonitor{
		Event: func(e *event.PoolEvent) {
			if parent != nil {
				parent.Event(e)
			}
			c.updateStats(e)
		},
	}
}

func (c *poolMonitorCollector) updateStats(e *event.PoolEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch e.Type {
	case event.ConnectionClosed:
		c.connClosed++
	case event.PoolCreated:
		c.poolCreated++
	case event.ConnectionCreated:
		c.connCreated++
	case event.GetFailed:
		c.getFailed++
	case event.GetSucceeded:
		c.getSucceded++
	case event.ConnectionReturned:
		c.connReturned++
	case event.PoolCleared:
		c.poolCleard++
	case event.PoolClosedEvent:
		c.poolClosed++
	}

	if e.PoolOptions != nil {
		c.maxPoolSize = e.PoolOptions.MaxPoolSize
		c.minPoolSize = e.PoolOptions.MinPoolSize
		c.waitQueueTimeoutMS = e.PoolOptions.WaitQueueTimeoutMS
	}
}
