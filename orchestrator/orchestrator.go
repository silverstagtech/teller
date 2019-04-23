package orchestrator

import (
	"os"
	"syscall"

	"github.com/silverstagtech/loggos"
	"github.com/silverstagtech/teller/config"
	"github.com/silverstagtech/teller/influxShipper"
	"github.com/silverstagtech/teller/metricCreator"
	"github.com/silverstagtech/teller/statsdShipper"
	"github.com/silverstagtech/teller/trigger"
)

const (
	influxEvent  = "influx"
	statsdEvent  = "statsd"
	sleeperEvent = "sleeper"
)

// Orchestrator controls the firing of metrics as defined in the configuration.
// It will shutdown the connections to the various systems on a SIGTERM or a SIGHUP.
// The Finished chan will send a true and close once all systems have been shutdown.
type Orchestrator struct {
	signals           chan os.Signal
	config            *config.Config
	Finished          chan bool
	influxConnections map[string]*influxShipper.InfluxShipper
	statsdConnections map[string]*statsdShipper.StatsDShipper
	StopChan          chan error
	timelines         []*timeline
}

// New creates a new Orchestrator and returns it.
func New(signals chan os.Signal, config *config.Config) *Orchestrator {
	return &Orchestrator{
		signals:           signals,
		config:            config,
		Finished:          make(chan bool, 1),
		StopChan:          make(chan error, 1),
		influxConnections: make(map[string]*influxShipper.InfluxShipper),
		statsdConnections: make(map[string]*statsdShipper.StatsDShipper),
		timelines:         make([]*timeline, 0),
	}
}

type eventMetric struct {
	fire func()
}

// Start will start the various connections and start sending metrics.
// An error is returned if any of the systems fail. An attempt to shutdown any started systems
// will be made. The Orchestrator will shutdown on an error and queue a True in the Finished
// chan should there be a failure.
func (o *Orchestrator) Start() error {
	loggos.SendJSON(loggos.JSONDebugln("Orchestrator attempting to start influx connections."))
	err := o.startInflux()
	if err != nil {
		return err
	}
	loggos.SendJSON(loggos.JSONDebugln("Orchestrator attempting to start statsd connections."))
	err = o.startStatsd()
	if err != nil {
		return err
	}
	jm := loggos.JSONDebugln("Orchestrator attempting to start timelines")
	jm.Add("story_name", o.config.Story.StoryName)
	loggos.SendJSON(jm)

	err = o.startTimelines()
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case signal := <-o.signals:
				o.stop(signal)
			}
		}
	}()
	return nil
}

func (o *Orchestrator) stop(signal os.Signal) {
	log := func(sig os.Signal) {
		jm := loggos.JSONInfoln("Got a signale, shutting down.")
		jm.Addf("signal", "%v", sig)
		loggos.SendJSON(jm)
	}
	switch signal {
	case syscall.SIGINT:
		log(signal)
		o.shutdown()
	case syscall.SIGKILL:
		log(signal)
		o.shutdown()
	case syscall.SIGTERM:
		log(signal)
		o.shutdown()
	default:
		jm := loggos.JSONInfoln("Got a signal not linked to an action, ignoring signal.")
		jm.Addf("signal", "%v", signal)
		loggos.SendJSON(jm)
	}
}

func (o *Orchestrator) shutdown() {
	log := func(msg string) {
		loggos.SendJSON(loggos.JSONDebugln(msg))
	}

	log("Orchestrator attempting to stop triggers")
	for _, tl := range o.timelines {
		tl.shutdown()
	}
	for _, tl := range o.timelines {
		<-tl.StopChan
	}
	log("Orchestrator attempting to stop StatsD connections")
	for _, statsdC := range o.statsdConnections {
		statsdC.Stop()
	}
	log("Orchestrator attempting to stop Influx connections")
	for _, influxC := range o.influxConnections {
		influxC.Stop()
		<-influxC.StopChan
	}
	o.StopChan <- nil
}

func (o *Orchestrator) startInflux() error {
	if len(o.config.Story.Influx) == 0 {
		loggos.SendJSON(loggos.JSONDebugln("No Influx connections moving on."))
		return nil
	}
	for _, influxConfig := range o.config.Story.Influx {
		jm := loggos.JSONInfoln("Creating Influx connection")
		jm.Add("connection_id", influxConfig.ID)
		loggos.SendJSON(jm)
		shipper := influxShipper.New(
			influxConfig.ID,
			influxConfig.Host,
			influxConfig.Database,
			influxConfig.Username,
			influxConfig.Password,
			influxConfig.Precision,
			influxConfig.BatchSize,
			influxConfig.FlushInterval,
			influxConfig.NWriters,
			influxConfig.HTTPTimeout,
		)

		jm = loggos.JSONInfoln("Testing Influx connection")
		jm.Add("connection_id", influxConfig.ID)
		loggos.SendJSON(jm)

		err := shipper.Connect()
		if err != nil {
			jm = loggos.JSONCritln("Influx connection failed")
			jm.Add("connection_id", influxConfig.ID)
			loggos.SendJSON(jm)
			return err
		}
		jm = loggos.JSONInfoln("Starting Influx shipper")
		jm.Add("connection_id", influxConfig.ID)
		loggos.SendJSON(jm)

		shipper.Start()
		o.influxConnections[influxConfig.ID] = shipper
	}
	return nil
}

func (o *Orchestrator) startStatsd() error {
	if len(o.config.Story.StatsD) == 0 {
		return nil
	}
	for _, statsdConfig := range o.config.Story.StatsD {
		shipper := statsdShipper.New(
			statsdConfig.ID,
			statsdConfig.Host,
			statsdConfig.Port,
			statsdConfig.Transport,
			statsdConfig.QueueDepth,
		)

		if err := shipper.Connect(); err != nil {
			return err
		}
		shipper.Start()
		o.statsdConnections[statsdConfig.ID] = shipper
	}
	return nil
}

func (o *Orchestrator) startTimelines() error {
	for _, timelineConfig := range o.config.Story.TimeLines {
		jm := loggos.JSONDebugln("Creating Timeline")
		jm.Add("timeline_name", timelineConfig.Name)
		jm.Add("continuous", o.config.Story.Continuous)
		loggos.SendJSON(jm)
		tl := &timeline{
			trigger:  trigger.New(timelineConfig.Name, o.config.Story.Continuous),
			StopChan: make(chan bool, 1),
			events:   make(map[string]*eventMetric),
			Name:     timelineConfig.Name,
		}
		o.timelines = append(o.timelines, tl)
		for _, timeslice := range timelineConfig.Timeslices {
			timesliceIndex := tl.trigger.NewTimeSlice(timeslice.Name, timeslice.Repeat, timeslice.SingleUse)
			for eventIndex, event := range timeslice.Events {
				id := tl.addEventTrigger(timeslice.Name, timesliceIndex, eventIndex, event)
				eventMetric, err := o.newEventMetric(event)
				if err != nil {
					return err
				}
				tl.events[id] = eventMetric
			}
		}
		tl.startFiring()
		err := tl.trigger.Start()
		if err != nil {
			return err
		}
	}
	if !o.config.Story.Continuous {
		go o.waitForSinglesToFinish()
	}
	return nil
}

func (o *Orchestrator) waitForSinglesToFinish() {
	for _, tl := range o.timelines {
		<-tl.StopChan
	}
	o.shutdown()
}

func (o *Orchestrator) newEventMetric(event *config.Event) (*eventMetric, error) {
	switch event.Type {
	case influxEvent:
		return o.createInfluxEventMetric(event)
	case statsdEvent:
		return o.createStatsdEventMetric(event)
	case sleeperEvent:
		return o.createSleeperEvent(event)
	}
	return nil, nil
}

func (o *Orchestrator) createInfluxEventMetric(event *config.Event) (*eventMetric, error) {
	metric, err := o.createMetric(event)
	if err != nil {
		return nil, err
	}
	f := func() {
		jm := loggos.JSONDebugln("Firing event.")
		jm.Add("type", "Influx")
		jm.Add("event_id", event.ConnectionID)
		jm.Add("event_text", metric.Influx())
		loggos.SendJSON(jm)

		o.influxConnections[event.ConnectionID].Ship(metric.Influx())
	}
	return &eventMetric{
		fire: f,
	}, nil
}

func (o *Orchestrator) createStatsdEventMetric(event *config.Event) (*eventMetric, error) {
	metric, err := o.createMetric(event)
	if err != nil {
		return nil, err
	}
	if len(event.StatsDTaggingFormat) > 0 {
		if err := metric.SetTaggingFormat(event.StatsDTaggingFormat); err != nil {
			return nil, err
		}
	}
	f := func() {
		jm := loggos.JSONDebugln("Firing event.")
		jm.Add("type", "StatsD")
		jm.Add("event_id", event.ConnectionID)
		jm.Add("event_text", metric.Influx())
		loggos.SendJSON(jm)
		o.statsdConnections[event.ConnectionID].Ship(metric.StatsD())
	}
	return &eventMetric{
		fire: f,
	}, nil
}

func (o *Orchestrator) createSleeperEvent(event *config.Event) (*eventMetric, error) {
	return &eventMetric{fire: func() {}}, nil
}

func (o *Orchestrator) createMetric(event *config.Event) (metricCreator.Metric, error) {
	metric, err := metricCreator.NewMetric(event.MetricName, event.Tags, event.Fields)
	if err != nil {
		return nil, err
	}
	return metric, nil
}
