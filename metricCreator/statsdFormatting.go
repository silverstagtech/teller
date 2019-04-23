package metricCreator

import (
	"fmt"
	"strings"

	ilpo "github.com/morfien101/influxLineProtocolOutput"
)

const (
	magicTagSampleRate = "sample_rate"
	magicTagMetricType = "metric_type"
)

var (
	statsdMetricTypes = map[string]string{"gauge": "g", "set": "s", "counter": "c", "timing": "ms", "histogram": "h"}
)

func digestTags(tags map[string]string) (newTags map[string]string, metricType, sampleRate string) {
	newTags = make(map[string]string)
	for key, value := range tags {
		switch key {
		case magicTagMetricType:
			metricType = statsdMetricTypes[value]
		case magicTagSampleRate:
			sampleRate = value
		default:
			newTags[key] = value
		}
	}
	return
}

func pairTags(t map[string]string, seperator string) []string {
	returnTags := make([]string, len(t))
	index := 0
	for name, value := range t {
		returnTags[index] = fmt.Sprintf("%s%s%s", name, seperator, value)
		index++
	}
	return returnTags
}

func formatSampleRate(sampleRate string) string {
	return fmt.Sprintf("@%v", sampleRate)
}

// DataDog format
// Tags are at the end with a preceding #. Tags are colon pairs and comma seperated.
// Sample rate is optional.
// As a side note telegraf is capable of digesting both in the same metric. They look for datadog
// using the preceding # after splitting on a |
//
// https://docs.datadoghq.com/developers/dogstatsd/datagram_shell/
//
// Example:
// metric_name:value|type[|sample_rate]|#tag:value,tag:value

func statsdWithDDTagging(metric *ilpo.MetricContainer) string {
	// Get tags
	tags, metricType, sampleRate := digestTags(metric.Tags)
	// Build tagging type and sample rate to be attached to metric
	var tagsString string
	if len(tags) > 0 {
		tagsString = fmt.Sprintf("#%s", strings.Join(pairTags(tags, ":"), ","))
	}

	components := []string{metricType}
	if len(sampleRate) > 0 {
		components = append(components, formatSampleRate(sampleRate))
	}
	if len(tagsString) > 0 {
		components = append(components, tagsString)
	}

	metadata := strings.Join(components, "|")

	// for each field attach the value, type, samplerate and tags.
	metrics := make([]string, len(metric.Values))
	index := 0
	for field, value := range metric.Values {
		measurement := fmt.Sprintf("%s_%s:%v", metric.Name, field, value)
		metrics[index] = fmt.Sprintf("%s|%s", measurement, metadata)
		index++
	}
	// return with \n seperation
	return strings.Join(metrics, "\n")
}

// Influx format
// Influx pushed the tags into the middle of the metrics and have a = key value pair and comma seperated.
// Sampling is option here.
//
// https://www.influxdata.com/blog/getting-started-with-sending-statsd-metrics-to-telegraf-influxdb/
//
// Example:
// metric_name[,tag=value,tag=value]:value|type[|sample_rate]

func statsdWithInfluxTagging(metric *ilpo.MetricContainer) string {
	// Get tags
	tags, metricType, sampleRate := digestTags(metric.Tags)
	// Build tagging type and sample rate to be attached to metric
	tagsString := fmt.Sprintf("%s", strings.Join(pairTags(tags, "="), ","))

	var components []string
	if len(sampleRate) > 0 {
		components = []string{metricType, formatSampleRate(sampleRate)}
	} else {
		components = []string{metricType}
	}
	metadata := strings.Join(components, "|")
	// for each field attach the value, type, samplerate and tags.
	metrics := make([]string, len(metric.Values))
	index := 0
	for field, value := range metric.Values {
		measurement := fmt.Sprintf("%s_%v", metric.Name, field)
		if len(tagsString) > 0 {
			// inject tags
			measurement = fmt.Sprintf("%s,%s", measurement, tagsString)
		}
		measurement = fmt.Sprintf("%s:%v", measurement, value)
		metrics[index] = fmt.Sprintf("%s|%s", measurement, metadata)
		index++
	}

	// return with \n seperation
	return strings.Join(metrics, "\n")
}
