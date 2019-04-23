package metricCreator

import (
	"fmt"
	"strings"

	ilpo "github.com/morfien101/influxLineProtocolOutput"
)

var (
	statsdTaggingDataDog = "datadog"
	statsdTaggingInflux  = "influx"
	validTaggingFormats  = []string{statsdTaggingDataDog, statsdTaggingInflux}
)

// Metric is a data point that can be represented as a Raw string,
// Influx Line Protocol or as a StatsD metric
type Metric interface {
	InfluxMetric
	StatsdMetric
}

// InfluxMetric is a datapoint that can be formatted as a influx metric
type InfluxMetric interface {
	Influx() string
}

// StatsdMetric is a datapoint that can be formatted as a statsd metric.
// It must also be able to set the tagging format.
type StatsdMetric interface {
	SetTaggingFormat(string) error
	StatsD() string
}

// MetricObject is a Influx Line Protocol version of a metric
type MetricObject struct {
	mc            *ilpo.MetricContainer
	taggingFormat string
}

// NewMetric will return a MetricObject which can output the metric in Influx or Statsd
func NewMetric(name string, tags map[string]string, fields map[string]interface{}) (*MetricObject, error) {
	newMetric := new(MetricObject)
	newMetric.mc = ilpo.New(name)
	newMetric.mc.Add(tags, fields)
	// We should error check the field values and types here.
	// ilpo should be able to do this.
	// Consider pushing a pull request with this functionality

	return newMetric, nil
}

// SetTaggingFormat is used to setup the tagging policy that you would like on the metric. Only used when sending on statsd.
func (m *MetricObject) SetTaggingFormat(requestedFormat string) error {
	for _, validFormat := range validTaggingFormats {
		if validFormat == requestedFormat {
			m.taggingFormat = requestedFormat
			return nil
		}
	}
	return fmt.Errorf("requested format %s is not a valid tagging format. Only %s are valid", requestedFormat, strings.Join(validTaggingFormats, ","))
}

// String outputs the metric as a influx line protocol string
func (m *MetricObject) String() string {
	return m.Influx()
}

// Influx returns the metric in Influx Line Protocol Output
func (m *MetricObject) Influx() string {
	return m.mc.Output()
}

// StatsD returns the metric in StatsD format with the requested tagging format.
func (m *MetricObject) StatsD() string {
	switch m.taggingFormat {
	case statsdTaggingInflux:
		return statsdWithInfluxTagging(m.mc)
	case statsdTaggingDataDog:
		return statsdWithDDTagging(m.mc)
	default:
		return statsdWithDDTagging(m.mc)
	}
}
