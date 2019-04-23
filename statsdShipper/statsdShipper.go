package statsdShipper

import (
	"fmt"
	"net"

	"github.com/silverstagtech/loggos"
)

const (
	// TCP is a valid value
	TCP = "tcp"
	// UDP is a valid value
	UDP = "udp"
)

// Create statsd metric
// Create a queue
// Send metric to queue
// Consume on queue and send to statsd endpoint

// StatsDShipper will ship statsd metrics to the endpoint contained within it.
// It will process them as fast as it can with the input chan.
type StatsDShipper struct {
	id         string
	host       string
	port       uint16
	transport  string
	input      chan string
	stop       chan bool
	stopped    bool
	StopChan   chan bool
	finished   bool
	connection net.Conn
}

// New will return a *StatsDShipper. Make sure that you call Connect on it before using it.
// Transport must be a string that is either "tcp" or "udp".
func New(id, host string, port uint16, transport string, queueDepth int) *StatsDShipper {
	return &StatsDShipper{
		id:        id,
		host:      host,
		port:      port,
		transport: transport,
		input:     make(chan string, queueDepth),
	}
}

// Ship takes a metric and queues it for transport to the endpoint defined.
func (sd *StatsDShipper) Ship(metric string) error {
	if sd.stopped {
		return fmt.Errorf("input is closed")
	}
	select {
	case sd.input <- metric:
	default:
		jm := loggos.JSONCritln("StatsD connection failed to send metric because buffer is full and message dropped.")
		jm.Add("connection_id", sd.id)
		jm.Add("metric_string", metric)
		loggos.SendJSON(jm)
	}
	sd.input <- metric
	return nil
}

// Start will signal the StatsDShipper to start sending metrics that it gets on the
// input channel. It returns a channel that will signal when closed and flushing is complete.
// When its ready to stop, close the input queue to signal that all metrics have been sent.
func (sd *StatsDShipper) Start() {
	sd.StopChan = make(chan bool, 1)

	go func() {
		for {
			select {
			case metric, ok := <-sd.input:
				if !ok {
					if err := sd.disconnect(); err != nil {
						jm := loggos.JSONWarnln("StatsD connection had an error disconnecting.")
						jm.Add("connection_id", sd.id)
						jm.Error(err)
						loggos.SendJSON(jm)
					}
					sd.finished = true
					sd.StopChan <- true
					return
				}
				err := sd.send(metric)
				if err != nil {
					jm := loggos.JSONWarnln("StatsD connection had an error sending.")
					jm.Add("connection_id", sd.id)
					jm.Add("metric_string", metric)
					jm.Error(err)
					loggos.SendJSON(jm)
				}
			}
		}
	}()
}

// Stop will signal the StatsDShipper to no longer take new metrics and to
// drain any metrics it currently has in its queue.
func (sd *StatsDShipper) Stop() {
	sd.stopped = true
	close(sd.input)
}

// Connect will try to make the connection the StatsDShipper describes within it.
func (sd *StatsDShipper) Connect() error {
	jm := loggos.JSONInfoln("StatsD connection attempting to connect.")
	jm.Add("connection_id", sd.id)
	jm.Add("transport", sd.transport)
	loggos.SendJSON(jm)

	conn, err := net.Dial(sd.transport, fmt.Sprintf("%s:%d", sd.host, sd.port))
	if err != nil {
		return err
	}
	sd.connection = conn
	return nil
}

// send will try to send the metric to the endpoint
func (sd *StatsDShipper) send(metric string) error {
	_, err := sd.connection.Write([]byte(metric + "\n"))
	return err
}

func (sd *StatsDShipper) disconnect() error {
	jm := loggos.JSONInfoln("StatsD connection attempting to shut down")
	jm.Add("connection_id", sd.id)
	jm.Add("transport", sd.transport)
	loggos.SendJSON(jm)

	return sd.connection.Close()
}

// Finished will signal if the shipper is finished sending all the metrics given to it.
// This can be used after the signaling channel has been discarded.
func (sd *StatsDShipper) Finished() bool {
	return sd.finished
}
