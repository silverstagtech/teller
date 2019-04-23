package metricCreator

import (
	"regexp"
	"testing"
)

func TestNewMetric(t *testing.T) {
	tests := []struct {
		testName                    string
		name                        string
		tags                        map[string]string
		fields                      map[string]interface{}
		expectedOutputInflux        string
		expectedOutputStatsDInflux  string
		expectedOutputStatsDDatadog string
	}{
		{
			testName: "both statsd tagging with metric_type only",
			name:     "test_metric_name",
			tags: map[string]string{
				"metric_type": "counter",
			},
			fields: map[string]interface{}{
				"f1": 1,
			},
			expectedOutputStatsDInflux:  "test_metric_name_f1:1|c",
			expectedOutputStatsDDatadog: "test_metric_name_f1:1|c",
		},
		{
			testName: "both statsd tagging with metric_type and sample_rate only",
			name:     "test_metric_name",
			tags: map[string]string{
				"metric_type": "counter",
				"sample_rate": "0.2",
			},
			fields: map[string]interface{}{
				"f1": 1,
			},
			expectedOutputStatsDInflux:  "test_metric_name_f1:1|c|@0.2",
			expectedOutputStatsDDatadog: "test_metric_name_f1:1|c|@0.2",
		},
		{
			testName: "influx",
			name:     "test_metric_name",
			tags: map[string]string{
				"tag1": "tag_value1",
			},
			fields: map[string]interface{}{
				"event": "Potatoes",
			},
			expectedOutputInflux: "test_metric_name,tag1=tag_value1 event=Potatoes",
		},
		{
			testName: "statsd event",
			name:     "test_metric_event",
			tags: map[string]string{
				"metric_type": "counter",
				"event_text":  "Potatoes are starting",
			},
			fields: map[string]interface{}{
				"starting": 1,
			},
			expectedOutputStatsDInflux:  "test_metric_event_starting,event_text=Potatoes are starting:1|c",
			expectedOutputStatsDDatadog: "test_metric_event_starting:1|c|#event_text:Potatoes are starting",
		},
	}

	for _, test := range tests {
		m, _ := NewMetric(test.name, test.tags, test.fields)

		if len(test.expectedOutputInflux) > 0 {
			if m.Influx() != test.expectedOutputInflux {
				t.Logf("Test \"%s\" failed.\nExpected: %s\nGot: %s", test.testName, test.expectedOutputInflux, m.Influx())
				t.Fail()
			}
		}

		if len(test.expectedOutputStatsDInflux) > 0 {
			err := m.SetTaggingFormat(statsdTaggingInflux)
			if err != nil {
				t.Logf("%s failed to set the format type", test.name)
				t.FailNow()
			}

			if m.StatsD() != test.expectedOutputStatsDInflux {
				t.Logf("Test \"%s\" with influx tagging failed.\nExpected: %s\nGot: %s", test.testName, test.expectedOutputStatsDInflux, m.StatsD())
				t.Fail()
			}
		}

		if len(test.expectedOutputStatsDDatadog) > 0 {
			// Datadog is the default format but we need to set it anyway because a previous test could have changed it.
			err := m.SetTaggingFormat(statsdTaggingDataDog)
			if err != nil {
				t.Logf("%s failed to set the format type", test.name)
				t.FailNow()
			}

			if m.StatsD() != test.expectedOutputStatsDDatadog {
				t.Logf("Test \"%s\" with datadog tagging failed.\nExpected: %s\nGot: %s", test.testName, test.expectedOutputStatsDDatadog, m.StatsD())
				t.Fail()
			}
		}
	}
}

func TestMultiTags(t *testing.T) {
	testdata := []struct {
		testName                           string
		metricName                         string
		tags                               map[string]string
		fields                             map[string]interface{}
		expectedOutputInfluxRegex          string
		expectedOutputInfluxMatches        int
		expectedOutputStatsDInfluxRegex    string
		expectedOutputStatsDInfluxMatches  int
		expectedOutputStatsDDatadogRegex   string
		expectedOutputStatsDDatadogMatches int
	}{
		{
			testName:   "test1",
			metricName: "test_metric",
			tags: map[string]string{
				"tag1":        "tag_value_1",
				"tag2":        "tag_value_2",
				"metric_type": "gauge",
			},
			fields: map[string]interface{}{
				"f1": 100,
				"f2": 22,
			},
			expectedOutputInfluxRegex:          `test_metric(?:,(?:tag1=tag_value_1|tag2=tag_value_2|metric_type=gauge))+ (?:f1=100|f2=22)+`,
			expectedOutputInfluxMatches:        1,
			expectedOutputStatsDInfluxRegex:    `test_metric_(?:f1|f2)(?:,(?:tag1=tag_value_1|tag2=tag_value_2))+:(?:(?:100|22)\|g)`,
			expectedOutputStatsDInfluxMatches:  2,
			expectedOutputStatsDDatadogRegex:   `test_metric_(?:f1|f2):(?:(?:100|22)\|g)|#(?:(?:tag1:tag_value_1|tag2:tag_value_2))+`,
			expectedOutputStatsDDatadogMatches: 2,
		},
	}

	for _, test := range testdata {
		metric, _ := NewMetric(test.metricName, test.tags, test.fields)

		if len(test.expectedOutputInfluxRegex) > 0 {
			re := regexp.MustCompile(test.expectedOutputInfluxRegex)
			if len(re.FindAllString(metric.Influx(), -1)) < test.expectedOutputInfluxMatches {
				t.Logf("Test \"%s\" failed.\nExpected regex: %s\nGot: %s", test.testName, test.expectedOutputInfluxRegex, metric.Influx())
				t.Fail()
			}
		}

		if len(test.expectedOutputStatsDInfluxRegex) > 0 {
			metric.SetTaggingFormat(statsdTaggingInflux)
			re := regexp.MustCompile(test.expectedOutputStatsDInfluxRegex)
			if len(re.FindAllString(metric.StatsD(), -1)) < test.expectedOutputStatsDInfluxMatches {
				t.Logf("Test \"%s\" failed.\nExpected regex: %s\nGot: %s", test.testName, test.expectedOutputStatsDInfluxRegex, metric.StatsD())
				t.Fail()
			}
		}
		if len(test.expectedOutputStatsDDatadogRegex) > 0 {
			metric.SetTaggingFormat(statsdTaggingDataDog)
			re := regexp.MustCompile(test.expectedOutputStatsDDatadogRegex)
			if len(re.FindAllString(metric.StatsD(), -1)) < test.expectedOutputStatsDDatadogMatches {
				t.Logf("Test \"%s\" failed.\nExpected regex: %s\nGot: %s", test.testName, test.expectedOutputStatsDDatadogRegex, metric.StatsD())
				t.Fail()
			}
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		testName             string
		name                 string
		tags                 map[string]string
		fields               map[string]interface{}
		expectedOutputInflux string
		expectedOutputStatsD string
	}{
		{
			testName: "Test String",
			name:     "test_metric_name",
			tags: map[string]string{
				"metric_type": "counter",
				"potatoes":    "boil_em",
			},
			fields: map[string]interface{}{
				"potatoes": 4,
			},
			expectedOutputInflux: "test_metric_name,metric_type=counterpotatoes=boil_em potatoes=4",
		},
	}

	for _, test := range tests {
		m, _ := NewMetric(test.name, test.tags, test.fields)
		if m.Influx() != m.String() {
			t.Logf("%s: String() does not match Influx().\nString:%s\nInflux:%s", test.testName, m.String(), m.Influx())
			t.Fail()
		}
	}
}

func TestStatsDMetricTypes(t *testing.T) {
	tests := []struct {
		name          string
		metricName    string
		tags          map[string]string
		fields        map[string]interface{}
		expectedRegex string
	}{
		{
			name:          "Normal gauge",
			metricName:    "test_counter",
			tags:          map[string]string{"metric_type": "counter", "testing_tag": "tag_value"},
			fields:        map[string]interface{}{"test_value": 3},
			expectedRegex: `:3|c`,
		},
		{
			name:          "Adding Gauge",
			metricName:    "additive_gauge",
			tags:          map[string]string{"metric_type": "gauge", "testing_tag": "tag_value"},
			fields:        map[string]interface{}{"test_value": "+5"},
			expectedRegex: `:+5|g`,
		},
		{
			name:          "Subtracting gauge",
			metricName:    "subtracting_gauge",
			tags:          map[string]string{"metric_type": "gauge", "testing_tag": "tag_value"},
			fields:        map[string]interface{}{"test_value": "-3"},
			expectedRegex: `:-3|c`,
		},
		{
			name:          "Normal gauge",
			metricName:    "normal_gauge",
			tags:          map[string]string{"metric_type": "gauge", "testing_tag": "tag_value"},
			fields:        map[string]interface{}{"test_value": 3},
			expectedRegex: `:3|c`,
		},
		{
			name:          "Normal set",
			metricName:    "normal_set",
			tags:          map[string]string{"metric_type": "set", "testing_tag": "tag_value"},
			fields:        map[string]interface{}{"test_value": 3},
			expectedRegex: `:3|s`,
		},
		{
			name:          "Normal timer",
			metricName:    "normal_timer",
			tags:          map[string]string{"metric_type": "histogram", "testing_tag": "tag_value"},
			fields:        map[string]interface{}{"test_value": 300},
			expectedRegex: `:300|h`,
		},
	}

	for _, test := range tests {
		m, err := NewMetric(test.metricName, test.tags, test.fields)
		if err != nil {
			t.Logf("Got an error creating a test metric. Error: %s", err)
			t.FailNow()
		}
		re := regexp.MustCompile(test.expectedRegex)
		for _, format := range validTaggingFormats {
			m.SetTaggingFormat(format)

			if !re.MatchString(m.StatsD()) {
				t.Logf("%s failed. Expected statsd message to match regex %s. Got: %q.", test.name, test.expectedRegex, m.StatsD())
			}
		}
	}
}

func TestBadStatsdFormat(t *testing.T) {
	m, _ := NewMetric("test", nil, nil)
	err := m.SetTaggingFormat("potatoes")
	if err == nil {
		t.Logf("A bad tagging format was given and no error was raised.")
		t.Fail()
	}
}

func TestDefaultTaggingFormat(t *testing.T) {
	m, _ := NewMetric("test", map[string]string{"tag1": "v1", "metric_type": "counter", "sample_rate": "0.9"}, map[string]interface{}{"f1": 1})
	expectedOutput := "test_f1:1|c|@0.9|#tag1:v1"
	if m.StatsD() != expectedOutput {
		t.Logf("Default tagging format didn't give the expected output.\nExpected: %s\nGot: %s", expectedOutput, m.StatsD())
		t.Fail()
	}
}
