package config

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	validEventTypes      = []string{"influx", "statsd", "sleeper"}
	statsdMetricTypes    = []string{"gauge", "set", "counter", "timing", "histogram"}
	validPrecisions      = []string{"h", "m", "s", "ms", "u", "ns"}
	validStatsdTransport = []string{"tcp", "udp"}
)

// ValidationError is a collections of errors found while validation the configuration.
type ValidationError struct {
	errs []string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("The following configuration errors where found.\n%s", strings.Join(ve.errs, "\n"))
}

func (ve *ValidationError) add(errorText string) {
	ve.errs = append(ve.errs, errorText)
}

func (ve *ValidationError) hasErrors() bool {
	return len(ve.errs) > 0
}

func validateEvent(e Event, errorBucket *ValidationError) {
	if e.MetricName == "" {
		errorBucket.add("event metric_name can not be blank.")
	}

	// Checks that are type dependent go in here.
	if e.Type == "" {
		errorBucket.add("event type can not be blank.")
	} else {
		// Is the type valid
		validType := false
		for _, validEvent := range validEventTypes {
			if e.Type == validEvent {
				validType = true
				break
			}
		}
		if !validType {
			errorBucket.add(fmt.Sprintf("Type is not valid. Only %s is allowed.", strings.Join(validEventTypes, ",")))
		} else {
			if e.Type != "sleeper" {
				if e.ConnectionID == "" {
					errorBucket.add("event connection_id must not be blank.")
				}
				if len(e.Fields) < 1 {
					errorBucket.add("event must have at least 1 field.")
				}
			}
		}

		// if type is statsd it must have a metric_type it must also be valid
		if e.Type == "statsd" {
			if e.Tags["metric_type"] == "" {
				errorBucket.add(fmt.Sprintf("Statsd events must have a tag metric_type with values, %s.", strings.Join(statsdMetricTypes, ",")))
			} else {
				validStatsdType := false
				for _, validType := range statsdMetricTypes {
					if validType == e.Tags["metric_type"] {
						validStatsdType = true
					}
				}
				if !validStatsdType {
					errorBucket.add(fmt.Sprintf("Statsd metric type can only be one of the following: %s.", strings.Join(statsdMetricTypes, ",")))
				}
			}
		}
	}
	if e.Repeat < 1 {
		errorBucket.add("event repeat must be a positive number.")
	}
	validateTimeBetween(e.TimeBetween, errorBucket)
}

func validateTimeBetween(t TimeBetween, errorBucket *ValidationError) {
	if t.Static.Time == 0 && t.Dynamic.MinimumTime == 0 {
		errorBucket.add("event time_between must have at least 1 timer.")
	}
	if t.Static.Time > 0 && t.Dynamic.MinimumTime > 0 {
		errorBucket.add("event time_between can not have more than 1 type of timer configured.")
	}
	if t.Static.Time < 0 {
		errorBucket.add("event static timer must have a positive number")
	}
	if t.Dynamic.MinimumTime < 0 || t.Dynamic.Vary < 0 {
		errorBucket.add("event dynamic timer minimum_time and vary must be a positive number")
	}
}

func validateTimeSlice(ts Timeslice, errorBucket *ValidationError) {
	if ts.Name == "" {
		errorBucket.add("Timeslices need to have a name.")
	}
	if ts.Repeat < 0 {
		errorBucket.add("Timeslice should repeat at least once.")
	}
	if len(ts.Events) == 0 {
		errorBucket.add("Timeslices should have at least one event.")
	} else {
		for _, event := range ts.Events {
			validateEvent(*event, errorBucket)
		}
	}
}

func validateTimeLine(tl TimeLine, errorBucket *ValidationError) {
	if tl.Name == "" {
		errorBucket.add("Timelines must have a name.")
	}
	if len(tl.Timeslices) == 0 {
		if tl.Name == "" {
			errorBucket.add("A timeline has no name and no events.")
		} else {
			errorBucket.add(fmt.Sprintf("Timeline %s has no events", tl.Name))
		}
	} else {
		for _, ts := range tl.Timeslices {
			validateTimeSlice(*ts, errorBucket)
		}
	}
}

func validateInfluxConnection(i InfluxConnection, errorBucket *ValidationError) {
	if i.ID == "" {
		errorBucket.add("influx connect id can not be blank.")
	}
	if i.Host == "" {
		errorBucket.add("influx connection host can not be blank.")
	} else {
		re := regexp.MustCompile(`http[s]?\:\/\/[a-z0-9\-\_.]+(?:\:[0-9]+)?`)
		if !re.MatchString(i.Host) {
			errorBucket.add("influx connection hostname is invalid.")
		}
	}
	if i.Username == "" {
		errorBucket.add("influx connection username can not be blank.")
	}
	if i.Password == "" {
		errorBucket.add("influx connection password can not be blank.")
	}
	if i.Database == "" {
		errorBucket.add("influx connection database can not be blank.")
	}
	if i.Precision == "" {
		errorBucket.add("influx connection precision can not be blank.")
	} else {
		precisionValid := false
		for _, precision := range validPrecisions {
			if i.Precision == precision {
				precisionValid = true
			}
		}
		if !precisionValid {
			errorBucket.add(fmt.Sprintf("Precision is not valid. Only %s are valid options.", strings.Join(validPrecisions, ",")))
		}
	}
	if i.BatchSize == 0 {
		errorBucket.add("influx connection batch_size can not be blank.")
	}
	if i.FlushInterval == 0 {
		errorBucket.add("influx connection flush_interval can not be blank.")
	}
	if i.HTTPTimeout == 0 {
		errorBucket.add("influx connection http_timeout can not be blank.")
	}
	if i.NWriters == 0 {
		errorBucket.add("influx connection number_of_writers can not be blank.")
	}
}

func validateStatsd(s StatsDConnection, errorBucket *ValidationError) {
	if s.ID == "" {
		errorBucket.add("statsd id can not be blank.")
	}
	if s.Host == "" {
		errorBucket.add("statsd host can not be blank.")
	}
	if s.Transport == "" {
		errorBucket.add("statsd transport can not be blank.")
	} else {
		validTransport := false
		for _, transport := range validStatsdTransport {
			if s.Transport == transport {
				validTransport = true
			}
		}
		if !validTransport {
			errorBucket.add(fmt.Sprintf("statsd metrics transport %s is invalid. Only %s is valid.", s.Transport, strings.Join(validStatsdTransport, ",")))
		}
	}
	if s.Port < 1 {
		errorBucket.add("statsd port must be a positive number.")
	}
	if s.QueueDepth < 1 {
		errorBucket.add("statsd buffer_depth must be a positive number.")
	}
}

func validateNoDuplicateConnections(s Story, errorBucket *ValidationError) {
	influxIDs := make(map[string]bool)
	statsdIDs := make(map[string]bool)

	for _, i := range s.Influx {
		if influxIDs[i.ID] {
			errorBucket.add(fmt.Sprintf("Influx ID %s is duplicated.", i.ID))
		} else {
			influxIDs[i.ID] = true
		}
	}

	for _, i := range s.StatsD {
		if statsdIDs[i.ID] {
			errorBucket.add(fmt.Sprintf("Influx ID %s is duplicated.", i.ID))
		} else {
			statsdIDs[i.ID] = true
		}
	}
}

func validateEventLinks(s Story, errorBucket *ValidationError) {
	type link struct {
		count int
		used  bool
	}
	influxIds := make(map[string]*link)
	statsdIds := make(map[string]*link)

	// gather IDs
	for _, i := range s.Influx {
		influxIds[i.ID] = new(link)
	}
	for _, s := range s.StatsD {
		statsdIds[s.ID] = new(link)
	}

	for _, timeline := range s.TimeLines {
		for _, timeslice := range timeline.Timeslices {
			for _, event := range timeslice.Events {
				switch event.Type {
				case "sleeper":
					continue
				case "statsd":
					if statsdIds[event.ConnectionID] == nil {
						errorBucket.add(fmt.Sprintf("event %s has an bad id %s", event.MetricName, event.ConnectionID))
					} else {
						statsdIds[event.ConnectionID].count++
						statsdIds[event.ConnectionID].used = true
					}
				case "influx":
					if influxIds[event.ConnectionID] == nil {
						errorBucket.add(fmt.Sprintf("event %s has an bad id %s", event.MetricName, event.ConnectionID))
					} else {
						influxIds[event.ConnectionID].count++
						influxIds[event.ConnectionID].used = true
					}
				}
			}
		}
	}

	for id, links := range influxIds {
		if !links.used {
			errorBucket.add(fmt.Sprintf("influx connection %s has no events linked to it.", id))
		}
	}
	for id, links := range statsdIds {
		if !links.used {
			errorBucket.add(fmt.Sprintf("statsd endpoint %s has no events linked to it.", id))
		}
	}
}

func validateStory(s Story, errorBucket *ValidationError) {
	if s.StoryName == "" {
		errorBucket.add("story_name can not be blank")
	}
	// Check for duplicate connection IDs
	validateNoDuplicateConnections(s, errorBucket)
	// Check that the influx connections are valid
	for _, i := range s.Influx {
		validateInfluxConnection(*i, errorBucket)
	}
	// Check that the statsd connections are valid
	for _, s := range s.StatsD {
		validateStatsd(*s, errorBucket)
	}
	// check that the timelines and events are valid.
	for _, t := range s.TimeLines {
		validateTimeLine(*t, errorBucket)
	}
	// Check that every event has a valid link to a connection
	if len(errorBucket.errs) > 0 {
		validateEventLinks(s, errorBucket)
	}
}
