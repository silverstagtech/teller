// Package influxShipper writes metrics to InfluxDB.
// It is will create as many clients as you tell it to and split the write
// load between all the client.
//
// You can send metrics using no time stamp or with a time stamp written by
// InfluxShipper. The affect of this is more consitant metrics being written as
// you will have metrics that closer represent when the metric was emitted.
//
// You can also set the precision that you want Influx to use on your metrics.
// Only "us" NanoSeconds and "s" Seconds are currently supported.
//
// To start the shipper first, create a new shipper, call Connect() to make sure the
// connections to Influx work, call Start() to signal to the clients they can start
// shipping metrics. Use Ship(metric string) or ShipWithTimeStamp(metric string) to
// submit metrics that need to be written to the Influx Database.
//
// When you call Start() the shipper you will get a channel in return. This channel will
// receive a true bool once the shipper is finished shipping all metrics and closed
// connections to InfluxDB. You need to call Stop() to start this shutdown process.
//
package influxShipper

import (
	"fmt"
	"sync"
	"time"
)

// InfluxShipper will ship metrics to the endpoint given using the bucket size and time
// interval supplied.
type InfluxShipper struct {
	id                  string
	address             string
	database            string
	username            string
	password            string
	influxPrecision     string
	bucket              []string
	batchSize           int
	flushInterval       int
	httpTimeout         time.Duration
	numberOfConnections int
	queue               chan string
	StopChan            chan bool
	stopped             bool
	finished            bool
	shippers            []*shipper
}

// New will return a pointer to a InfluxShipper. You will need to call connect on it to get it ready to
// start sending metrics. httpTimeout is in Second.
func New(id, address, database, username, password, influxPrecision string, batchSize, flushInterval, numberOfConnections, httpTimeout int) *InfluxShipper {
	return &InfluxShipper{
		id:                  id,
		address:             address,
		username:            username,
		password:            password,
		database:            database,
		influxPrecision:     influxPrecision,
		batchSize:           batchSize,
		flushInterval:       flushInterval,
		numberOfConnections: numberOfConnections,
		httpTimeout:         time.Duration(httpTimeout) * time.Second,
		queue:               make(chan string, 1000),
		StopChan:            make(chan bool, 1),
	}
}

func (is *InfluxShipper) influxTimeStamp() string {
	// valid options
	//"h", "m", "s", "ms", "u", "ns"
	// Epoch normally counts to seconds. Use the UnixNano() to get down to
	// nano seconds which also includes ms, u.
	switch is.influxPrecision {
	case "h", "m", "s":
		return fmt.Sprintf("%d", time.Now().Unix())
	case "ms", "u", "ns":
		return fmt.Sprintf("%d", time.Now().UnixNano())
	default:
		return fmt.Sprintf("%d", time.Now().Unix())
	}
}

// newShipper will create a new shipper that will connect and write metrics to InfluxDB
func (is *InfluxShipper) newShipper(id int) *shipper {
	return &shipper{
		mothershipID:    is.id,
		id:              id,
		address:         is.address,
		database:        is.database,
		username:        is.username,
		password:        is.password,
		influxPrecision: is.influxPrecision,
		payloads:        make([]string, 0),
		batchSize:       is.batchSize,
		flushInterval:   is.flushInterval,
		finshedChan:     make(chan bool, 1),
		httpTimeout:     is.httpTimeout,
	}
}

// Connect will create the connections to InfluxDB and ping the
// service on each connection to make sure that it is working.
// If the connection fails a error is returned and no more connections
// are created after the failed connection. All connections that have
// already been established are closed in a best effort approach.
func (is *InfluxShipper) Connect() error {
	for id := 1; id <= is.numberOfConnections; id++ {
		shipper := is.newShipper(id)
		if err := shipper.connect(); err != nil {
			is.Stop()
			return err
		}
		is.shippers = append(is.shippers, shipper)
	}
	return nil
}

// Start signals that the shipper should start sending the metrics. Use Ship(metric string)
// to add metrics that should be shipped to the queue.
func (is *InfluxShipper) Start() {
	for _, ship := range is.shippers {
		go ship.consume(is.queue)
	}
}

// Stop will tell the InfluxDB Clients to flush the queues and then close connections
// to Influx. Once complete the StopChan will get a signal that it is finished.
func (is *InfluxShipper) Stop() {
	close(is.queue)
	is.closeShippers()
	is.stopped = true
	is.StopChan <- true
}

// closeShippers will attempt to stop the connections to influx on the
// shippers using a best effort approach.
func (is *InfluxShipper) closeShippers() {
	wg := &sync.WaitGroup{}
	for _, ship := range is.shippers {
		wg.Add(1)
		ship.stop(wg)
	}
	wg.Wait()
	is.finished = true
}

// ShipWithTimeStamp takes a metric with no time and first attaches the time with
// Nanoseconds when this function is called. It then sends the metric to be
// written on the next flush. Useful if you want your metrics times to be closer
// to when they are produced rather than when written to the database.
func (is *InfluxShipper) ShipWithTimeStamp(metric string) {
	is.queue <- fmt.Sprintf("%s %s", metric, is.influxTimeStamp())
}

// Ship takes a metric as a string and sends it to a Influx connections to be
// written to the database on the next flush.
func (is *InfluxShipper) Ship(metric string) error {
	if is.stopped {
		return fmt.Errorf("input is closed")
	}
	is.queue <- metric
	return nil
}

// Finished will signal if the shipper is finished sending all the metrics given to it.
// This can be used after the signaling channel has been discarded.
func (is *InfluxShipper) Finished() bool {
	return is.finished
}
