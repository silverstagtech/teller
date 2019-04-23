package config

// Story is a complete configuration that describes connections and timelines.
// A story basically defines how often things happen when
type Story struct {
	StoryName    string              `json:"story_name"`
	Continuous   bool                `json:"continuous"`
	DebugLogging bool                `json:"debug_logging"`
	GlobalTags   map[string]string   `json:"global_tags"`
	Influx       []*InfluxConnection `json:"influx"`
	StatsD       []*StatsDConnection `json:"statsd"`
	TimeLines    []*TimeLine         `json:"timelines"`
}

// TimeLine defines the expected structure of a list of timelines in a
// story.
type TimeLine struct {
	Name       string       `json:"timeline_name"`
	Timeslices []*Timeslice `json:"time_slices"`
}

// Timeslice is a group of events in a story, they can repeat if needed.
type Timeslice struct {
	Name      string   `json:"time_slice_name"`
	Events    []*Event `json:"events"`
	Repeat    int      `json:"repeat"`
	SingleUse bool     `json:"single_use"`
}

// Event is a timeline event that can be a sleeper or a metric being
// sent to the endpoint of choice.
type Event struct {
	MetricName          string                 `json:"metric_name"`
	Type                string                 `json:"type"`
	ConnectionID        string                 `json:"connection_id"`
	Repeat              int                    `json:"repeat"`
	Fields              map[string]interface{} `json:"fields"`
	Tags                map[string]string      `json:"tags"`
	StatsDTaggingFormat string                 `json:"statsd_tagging_format"`
	TimeBetween         TimeBetween            `json:"time_between"`
}

// TimeBetween defines the expected structure of a metric timing story
type TimeBetween struct {
	Static struct {
		Time int `json:"time"`
	} `json:"static"`
	Dynamic struct {
		MinimumTime int `json:"minimum_time"`
		Vary        int `json:"vary"`
	} `json:"dynamic"`
}

// InfluxConnection defines the expected structure of the Influx connections
// passing in via the configuration
type InfluxConnection struct {
	ID            string `json:"id"`
	Host          string `json:"host"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Database      string `json:"database"`
	Precision     string `json:"precision"`
	BatchSize     int    `json:"batch_size"`
	FlushInterval int    `json:"flush_interval"`
	HTTPTimeout   int    `json:"http_timeout"`
	NWriters      int    `json:"number_of_writers"`
}

// StatsDConnection defines the expected structure of the StatsD connections
// passing in via the configuration
type StatsDConnection struct {
	ID         string `json:"id"`
	Host       string `json:"host"`
	Port       uint16 `json:"port"`
	Transport  string `json:"transport"`
	QueueDepth int    `json:"buffer_depth"`
}
