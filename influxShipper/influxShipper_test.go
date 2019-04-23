package influxShipper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/silverstagtech/loggos"
)

func setupLogger() {
	loggos.JSONLoggerEnableDebugLogging(true)
	loggos.JSONLoggerEnablePrettyPrint(true)
	loggos.JSONLoggerEnableHumanTimestamps(true)
}

// This test can only be run when we have a influx server available
// There is a lot of coverage in here.
//func TestInfluxShipper(t *testing.T) {
//	metricRaw1 := struct {
//		name   string
//		tags   map[string]string
//		fields map[string]interface{}
//	}{
//		name: "test_influx_shipper",
//		tags: map[string]string{
//			"chilli": "hot",
//			"tomato": "cold",
//		},
//		fields: map[string]interface{}{
//			"chocolate": 1,
//		},
//	}
//
//	metricRaw2 := struct {
//		name   string
//		tags   map[string]string
//		fields map[string]interface{}
//	}{
//		name: "test_influx_shipper",
//		tags: map[string]string{
//			"milk":   "hot",
//			"coffee": "cold",
//		},
//		fields: map[string]interface{}{
//			"cheese": 1,
//		},
//	}
//	metric1, err := metricCreator.NewMetric(metricRaw1.name, metricRaw1.tags, metricRaw1.fields)
//	if err != nil {
//		t.Logf("Failed to created metric. Error: %s", err)
//		t.FailNow()
//	}
//	metric2, err := metricCreator.NewMetric(metricRaw2.name, metricRaw2.tags, metricRaw2.fields)
//	if err != nil {
//		t.Logf("Failed to created metric. Error: %s", err)
//		t.FailNow()
//	}
//
//	shipper := New("Full Tester", "http://localhost:8086", "telegraf", "test", "testicles", "ns", 1000, 3, 5, 5000)
//	err = shipper.Connect()
//	if err != nil {
//		t.Logf("Failed to connect to InfluxDB. Error %s", err)
//		t.FailNow()
//	}
//	shipper.Start()
//
//	for i := 0; i < 10000; i++ {
//		shipper.ShipWithTimeStamp(metric1.Influx())
//		shipper.Ship(metric2.Influx())
//	}
//	shipper.Stop()
//	<-shipper.StopChan
//}

func TestPingURL(t *testing.T) {
	setupLogger()
	ship := &shipper{
		address: "http://localhost:8086",
	}
	expectedOutput := "http://localhost:8086/ping"
	if ship.pingURL() != expectedOutput {
		t.Logf("TestPingURL failed to give the correct output.\nGot: %s\nWanted: %s", ship.pingURL(), expectedOutput)
		t.Fail()
	}
}

func TestWriteURL(t *testing.T) {
	setupLogger()
	ship := &shipper{
		address:         "http://localhost:8086",
		database:        "test",
		influxPrecision: "s",
	}
	expectedOutput := "http://localhost:8086/write?db=test&precision=s"
	if ship.writeURL() != expectedOutput {
		t.Logf("TestWriteURL failed to give the correct output.\nGot: %s\nWanted: %s", ship.writeURL(), expectedOutput)
		t.Fail()
	}
}

func TestMergeMetrics(t *testing.T) {
	setupLogger()
	ship := &shipper{
		payloads: []string{"one", "two", "three"},
	}
	expectedOutput := []byte("one\ntwo\nthree")

	if string(ship.mergeMetrics()) != string(expectedOutput) {
		t.Logf("TestMergeMetrics did not get the correct value.\nGot: %v\nWanted: %v", ship.mergeMetrics(), expectedOutput)
		t.Fail()
	}
}

func TestClearPayloads(t *testing.T) {
	setupLogger()
	ship := &shipper{
		payloads: []string{"one", "two", "three"},
	}
	ship.clearPayloads()
	if len(ship.payloads) < 0 {
		t.Logf("TestClearPayloads failed, payloads is larger than 0 after clearing.")
		t.Fail()
	}
}

func TestBadShipperConnect(t *testing.T) {
	setupLogger()
	ship := &shipper{
		address: "http://127.0.0.1:1234",
	}

	err := ship.connect()
	if err == nil {
		t.Logf("TestBadShipperConnect connecting to a non-existant Influx server did not product an error.")
		t.Fail()
	}
}

func TestTimeoutShipConnect(t *testing.T) {
	setupLogger()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 200)
		w.WriteHeader(201)
	})
	server := httptest.NewServer(handler)
	serverTLS := httptest.NewTLSServer(handler)

	ship := &shipper{
		address:     server.URL,
		httpTimeout: time.Duration(10) * time.Millisecond,
	}
	err := ship.connect()
	if err == nil {
		t.Logf("TestTimeoutShipConnect shipper timeout did not work on http server.")
		t.Fail()
	}

	ship = &shipper{
		address:     serverTLS.URL,
		httpTimeout: time.Duration(10) * time.Millisecond,
	}
	err = ship.connect()
	fmt.Println(err)
	if err == nil {
		t.Logf("TestTimeoutShipConnect shipper timeout did not work on http server.")
		t.Fail()
	}
}

func TestGoodShipConnect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(handler)
	serverTLS := httptest.NewTLSServer(handler)

	ship := &shipper{
		address:     server.URL,
		httpTimeout: time.Duration(100) * time.Millisecond,
	}
	err := ship.connect()
	if err != nil {
		t.Logf("TestGoodShipConnect shipper on http failed with error: %s.", err)
		t.Fail()
	}

	ship = &shipper{
		address:     serverTLS.URL,
		httpTimeout: time.Duration(100) * time.Millisecond,
	}
	err = ship.connect()
	fmt.Println(err)
	if err != nil {
		t.Logf("TestGoodShipConnect shipper on https failed with error: %s.", err)
		t.Fail()
	}
}

func TestGoodShipConnectBadStatusCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	server := httptest.NewServer(handler)

	ship := &shipper{
		address:     server.URL,
		httpTimeout: time.Duration(100) * time.Millisecond,
	}
	err := ship.connect()
	if err == nil {
		t.Logf("TestGoodShipConnectBadStatusCode shipper did not error on bad status code.")
		t.Fail()
	}
}

func TestInfluxShipperFinished(t *testing.T) {
	iship := &InfluxShipper{
		id:       "test",
		queue:    make(chan string, 10),
		StopChan: make(chan bool, 1),
	}
	iship.Start()
	iship.Stop()
	select {
	case stoppedSignal := <-iship.StopChan:
		if !stoppedSignal {
			t.Logf("TestInfluxShipperFinished: expected true stop signal.")
			t.Fail()
		}
	default:
		t.Logf("TestInfluxShipperFinished expected stop signal on channel.")
	}

	err := iship.Ship("test")
	if err == nil {
		t.Logf("TestInfluxShipperFinished: expected Ship to return an error after stopping the shipper.")
		t.Fail()
	}

	if !iship.Finished() {
		t.Logf("TestInfluxShipperFinished expected Finished() to return true after stopping the shipper.")
		t.Fail()
	}
}

func TestGoodInfluxShipperConnect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})

	testServers := []*httptest.Server{httptest.NewServer(handler), httptest.NewTLSServer(handler)}

	for _, server := range testServers {
		ishipper := New("test", server.URL, "test", "u", "p", "ns", 1000, 2, 5, 1000)

		err := ishipper.Connect()
		if err != nil {
			t.Logf("Failed to Connect shippers. Errors %s", err)
			t.FailNow()
		}
	}
}

func TestBadInfluxShipperConnect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	testServers := []*httptest.Server{httptest.NewServer(handler), httptest.NewTLSServer(handler)}

	for _, server := range testServers {
		ishipper := New("test", server.URL, "test", "u", "p", "ns", 1000, 2, 5, 1000)

		err := ishipper.Connect()
		if err == nil {
			t.Logf("Failed shippers did not raise and error")
			t.FailNow()
		}
	}
}
