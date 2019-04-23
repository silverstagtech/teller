package statsdShipper

// The below 2 tests can only be used to test when we have a telegraf and influx instance running.
// Like: https://github.com/samuelebistoletti/docker-statsd-influxdb-grafana
//
//func TestStatsdUDP(t *testing.T) {
//	metricRaw := struct {
//		name   string
//		tags   map[string]string
//		fields map[string]interface{}
//	}{
//		name: "test_metric_name",
//		tags: map[string]string{
//			"metric_type": "counter",
//			"run":         "UDP",
//		},
//		fields: map[string]interface{}{
//			"UDP":  1,
//			"UDP2": 2,
//			"UDP3": 3,
//		},
//	}
//	metric, _ := metricCreator.NewMetric(metricRaw.name, metricRaw.tags, metricRaw.fields)
//	sdShipper, err := New("127.0.0.1", 8125, UDP, 1000)
//	if err != nil {
//		fmt.Println(err)
//	}
//	err = sdShipper.Connect()
//	if err != nil {
//		fmt.Println(err)
//		t.FailNow()
//	}
//	sdShipper.Start()
//	for i := 0; i < 10000; i++ {
//		//time.Sleep(time.Millisecond * 150)
//		err := sdShipper.Ship(metric.StatsD())
//		if err != nil {
//			fmt.Println(err)
//		}
//	}
//	sdShipper.Stop()
//	<-sdShipper.StopChan
//}
//
//func TestStatsdTCP(t *testing.T) {
//	metricRaw := struct {
//		name   string
//		tags   map[string]string
//		fields map[string]interface{}
//	}{
//		name: "test_metric_name",
//		tags: map[string]string{
//			"metric_type": "counter",
//			"run":         "TCP",
//		},
//		fields: map[string]interface{}{
//			"TCP": 1,
//		},
//	}
//	metric, _ := metricCreator.NewMetric(metricRaw.name, metricRaw.tags, metricRaw.fields)
//	sdShipper, err := New("127.0.0.1", 8125, TCP, 100)
//	if err != nil {
//		fmt.Println(err)
//	}
//	err = sdShipper.Connect()
//	if err != nil {
//		fmt.Println(err)
//		t.FailNow()
//	}
//	sdShipper.Start()
//	for i := 0; i < 10000; i++ {
//		//time.Sleep(time.Millisecond * 150)
//		err := sdShipper.Ship(metric.StatsD())
//		if err != nil {
//			fmt.Println(err)
//		}
//	}
//	sdShipper.Stop()
//	<-sdShipper.StopChan
//}
