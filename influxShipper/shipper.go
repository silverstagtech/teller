package influxShipper

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/silverstagtech/loggos"
)

const (
	// MaxIdleConnections dictates how many connections each Influx writer can have to InfluxDB.
	MaxIdleConnections int = 10
)

type shipper struct {
	mothershipID    string
	id              int
	httpTransport   *http.Transport
	httpClient      *http.Client
	httpTimeout     time.Duration
	address         string
	database        string
	username        string
	password        string
	batchSize       int
	flushInterval   int
	payloads        []string
	finshedChan     chan bool
	running         bool
	influxPrecision string
}

func (ship *shipper) pingURL() string {
	return fmt.Sprintf("%s/ping", ship.address)
}
func (ship *shipper) writeURL() string {
	urlParams := []string{
		fmt.Sprintf("db=%s", ship.database),
		fmt.Sprintf("precision=%s", ship.influxPrecision),
	}
	return fmt.Sprintf("%s/write?%s", ship.address, strings.Join(urlParams, "&"))
}

func (ship *shipper) mergeMetrics() []byte {
	return []byte(strings.Join(ship.payloads, "\n"))
}

func (ship *shipper) clearPayloads() {
	ship.payloads = make([]string, 0)
}

func (ship *shipper) createHTTPClient() {
	ship.httpTransport = &http.Transport{
		MaxIdleConnsPerHost: MaxIdleConnections,
		Dial: (&net.Dialer{
			Timeout: ship.httpTimeout,
		}).Dial,
		TLSHandshakeTimeout: ship.httpTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	ship.httpClient = &http.Client{
		Transport: ship.httpTransport,
		Timeout:   ship.httpTimeout,
	}
}

func (ship *shipper) setCreds(req *http.Request) {
	if ship.username != "" && ship.password != "" {
		req.SetBasicAuth(ship.username, ship.password)
	}
}

func (ship *shipper) connect() error {
	ship.createHTTPClient()
	// We don't yet know if Influx is running or accepting connections.
	// Therefore we will now ping it to see if the connection is successful.
	req, err := http.NewRequest("GET", ship.pingURL(), nil)
	if err != nil {
		return fmt.Errorf("Failed to make request for ping. Error: %s", err)
	}
	ship.setCreds(req)
	resp, err := ship.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to ping Influx. Error: %s", err)
	}
	resp.Body.Close()

	if resp.StatusCode > 299 {
		return fmt.Errorf("Got a bad status code from influx. Code: %d", resp.StatusCode)
	}
	jm := loggos.JSONInfoln("Successful to Ping Influx")
	jm.Add("id", ship.id)
	jm.Add("parent_id", ship.mothershipID)
	jm.Add("shipper_type", "Influx")
	loggos.SendJSON(jm)
	return nil
}

func (ship *shipper) flush() {
	metricsToShip := ship.mergeMetrics()
	ship.clearPayloads()

	req, err := http.NewRequest("POST", ship.writeURL(), bytes.NewReader(metricsToShip))
	if err != nil {
		jm := loggos.JSONCritln("Failed to make a request to send data to InfluxDB.")
		jm.Add("id", ship.id)
		jm.Add("parent_id", ship.mothershipID)
		jm.Add("shipper_type", "Influx")
		jm.Add("url", ship.writeURL())
		jm.Add("data", string(metricsToShip))
		loggos.SendJSON(jm)

		return
	}
	ship.setCreds(req)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// We need to add in some jitter here or all the flushing happens at once
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
	resp, err := ship.httpClient.Do(req)
	if err != nil {
		jm := loggos.JSONCritln("Request to write failed.")
		jm.Add("id", ship.id)
		jm.Add("parent_id", ship.mothershipID)
		jm.Add("shipper_type", "Influx")
		jm.Error(err)
		loggos.SendJSON(jm)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 300 {
		jm := loggos.JSONCritln("Request to write got a bad status code.")
		jm.Add("id", ship.id)
		jm.Add("parent_id", ship.mothershipID)
		jm.Add("shipper_type", "Influx")
		jm.Add("response_code", resp.StatusCode)
		jm.Addf("response_dump", "%v", resp)
		loggos.SendJSON(jm)
		return
	}
	jm := loggos.JSONInfoln("Finished a flush successfully.")
	jm.Add("id", ship.id)
	jm.Add("parent_id", ship.mothershipID)
	jm.Add("shipper_type", "Influx")
	loggos.SendJSON(jm)
}

func (ship *shipper) consume(q chan string) {
	ship.running = true
	ticker := time.NewTicker(time.Second * time.Duration(ship.flushInterval))
	lastFlush := time.Now()
	flush := func() {
		ship.flush()
		lastFlush = time.Now()
	}
	for {
		select {
		case metric, ok := <-q:
			if !ok {
				ticker.Stop()
				flush()
				close(ship.finshedChan)
				return
			}
			ship.payloads = append(ship.payloads, metric)
			jm := loggos.JSONDebugln("Adding metric to queue.")
			jm.Add("queue size", len(ship.payloads))
			jm.Add("id", ship.id)
			jm.Add("parent_id", ship.mothershipID)
			jm.Add("shipper_type", "Influx")
			loggos.SendJSON(jm)

			if len(ship.payloads) == ship.batchSize {
				jm := loggos.JSONDebugln("flushing because queue is too large.")
				jm.Add("queue size", len(ship.payloads))
				jm.Add("id", ship.id)
				jm.Add("parent_id", ship.mothershipID)
				jm.Add("shipper_type", "Influx")
				loggos.SendJSON(jm)

				flush()
			}
		case <-ticker.C:
			if time.Since(lastFlush) >= time.Duration(ship.flushInterval) {
				if len(ship.payloads) > 0 {
					jm := loggos.JSONDebugln("flushing because flush timer hit.")
					jm.Add("queue size", len(ship.payloads))
					jm.Add("id", ship.id)
					jm.Add("parent_id", ship.mothershipID)
					jm.Add("shipper_type", "Influx")
					loggos.SendJSON(jm)
					flush()
				}
			} else {
				jm := loggos.JSONDebugln("flush skipped due to last flush being too close.")
				jm.Add("queue size", len(ship.payloads))
				jm.Add("id", ship.id)
				jm.Add("parent_id", ship.mothershipID)
				jm.Add("shipper_type", "Influx")
				loggos.SendJSON(jm)
			}
		}
	}
}

func (ship *shipper) stop(wg *sync.WaitGroup) {
	if ship.running {
		select {
		case _, ok := <-ship.finshedChan:
			if !ok {
				ship.httpTransport.CloseIdleConnections()
				ship.running = false
			}
		}
	}
	wg.Done()
}
