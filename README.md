# MetricGenerator

![Metrics are coming!](https://media.makeameme.org/created/brace-yourself-metrics.jpg)

## Why do we need it

The metric generator was born when we said the words, "I wish I could send metrics into influx to test my boards and alerts". This tool creates those metrics while trying to also tell the story that the metrics would tell.
With the tool you can simulate spikes, drops and gradual increases in velocity of the metrics you want to send.
You can also send multiple metrics at a time or even tell multiple stories at the same time.

Maybe you want to send 10000 metrics of ok then a annotation for a crash then nothing for 2 minutes then a steady stream of metrics again. Well now you can!

## How do I use it (TL;DR)

You need to create a configuration json file or edit the example one from `metric-generator -e`.
Once you have your configuration that tells your story you execute the app like so.

```bash
./metric-generator -c config.json
```

You will see some logs to tell you whats happening. The generator will exit out if there are any problems and present them to you in the logs.
If everything goes well you will be pushing metrics until you tell it to stop with a `Ctrl+C`.

## Still to come

[] Read metrics from a file and send to influx
[] Mimic time intervals between metrics in file reading

## How to use the metric generator

The metric generator is driven by the stories that the configuration files tell. In the configuration file you create timelines. These tell the generator when it should send metrics and when it should sleep. These stories are meant too reflect the stories that you would get from your metrics in the wild.

The generator can send to the following end points

* InfluxDB with and without authentication over HTTP and HTTPS
* InfluxDB
* StatsD using UDP
* StatsD using TCP

Metrics can have many tags and fields. fields and be strings, ints, floats and bools which is the types supported by Influx. Tags can only be strings.

Global tags can be created which will be added to all metrics.

### Configuration file

The configuration file is a JSON file. It tells the story in timelines which have slices of time which in turn have events that happen in them
. It also describes the endpoints to which you want to send metrics to.

Each section will be laid out below in detail. They will then be put together to form the complete configuration.

Below is the basic structure of the configuration file.

```text
{
  global values,
  influx endpoints: [
    list of influx endpoints
  ]
  statsd endpoints: [
    list of statsd endpoints
  ]
  timelines: [
    list of timelines to read: [
      list of time slices: [
        list of events in the time slice
      ]
    ]
  ]
}
```

#### Global configuration

Global settings are settings that are not part of the story but are still needed for configuration. See the table for further details.

The below is a example of the global configuration.

```json
{
  "story_name": "example",
  "continuous": false,
  "debug_logging": false,
  "global_tags": {
    "global_tag_one": "GValue1",
    "global_tag_two": "GValue2"
  }
}
```

Key | Type | Description
---|---|---
story_name | `string` | Name for the configuration.
continuous | `bool` | Should the times lines be repeated forever or should it finish once the timelines are completed once.
debug_logging | `bool` | Turn on debugging logs.
global_tags | `map[string]string` | A table of key value pairs that have tag names and values.

#### Influx

The `influx` section describes the InfluxDB servers that you are sending metrics to. Each server needs an ID which is unique to it. You need to use the ID in the events to link them to the server.

InfluxDB servers can handle requests from multiple users at the same time. Therefore you can have multiple workers writing metrics at the same time.
InfluxDB servers would rather you batch your requests and write either when you reach maximum batch size or slow enough to still meet your read requirements. This is implemented using the `batch_size` key and `flush_interval`. Exceeding either will trigger a flush on a writer. Bare in mind that each worker has its own `batch_size` so setting 2 workers and a batch_size of 1000 will require 2000 metrics to fill each workers batch. Each writer also maintains its own flush interval with ~200ms of jitter baked in to stop storming.

See table for further details.

The below is an example of a list of InfluxDB servers.

```json
{
  "influx": [
    {
      "id": "influx1",
      "host": "http://localhost:8086",
      "username": "user",
      "password": "password",
      "database": "telegraf",
      "precision": "us",
      "batch_size": 5000,
      "http_timeout": 10,
      "number_of_writers": 2,
      "flush_interval": 2
    },
    {
      "id": "influx2",
      "host": "https://influx.ontheinternet.com:8086",
      "username": "great_username",
      "password": "strong_password",
      "database": "stories",
      "precision": "s",
      "batch_size": 200,
      "http_timeout": 30,
      "number_of_writers": 3,
      "flush_interval": 10
    }
  ],
}
```

Key | Type | Valid values | Description
---|---|---|---
influx | `list` | NA | Contains a list of influx server configuration objects.
influx.id | `string` | anything | A unique string used when sending events to a server. You will need to put this into the event also.
influx.host | `string` | http(s)://something:port | The hostname of the server with either `http://` or `https://`, the hostname and the port. Default port is 8086.
influx.username | `string` | anything | Username used to connect to the InfluxDB server.
influx.password | `string` | anything | Password used to connect to the InfluxDB server.
influx.database | `string` | anything | Name of the database to send metrics to. The database must exist before sending metrics.
influx.precision | `string` | "rfc3339", "h", "m", "s", "ms", "u", "ns" | What precision to send metrics in. See [Influx precision - Does it matter](https://docs.influxdata.com/influxdb/v1.7/troubleshooting/frequently-asked-questions/#does-the-precision-of-the-timestamp-matter)
influx.batch_size | `int` | 1 - 32767 | How many data points to write in each write request. Recommended values are between 1000 and 4000. See [Influx writing](https://docs.influxdata.com/influxdb/v1.7/guides/writing_data/#writing-multiple-points) for further details.
influx.http_timeout | `int` | 1 - 32767 | Number of seconds to give to each stage of the HTTP connection. This should not be too high somewhere between 3 - 6 seconds is recommend. See [Writing multiple points](https://docs.influxdata.com/influxdb/v1.7/guides/writing_data/#writing-points-from-a-file) for further details.
influx.number_of_writers | `int` | 1 - 32767 | Number of workers that send metrics to InfluxDB. Recommended to be between 1 and 5.
influx.flush_interval | `int` | 1 - 32767 | Number of seconds betweens attempted writes on each writer.

#### StatsD

The `statsd` section describes the StatsD endpoints that you are sending metrics to. Each endpoint needs an ID which is unique to it. The various values need to be filled in. You need to use the ID in the events to link them to the endpoint.

Unlike influx metrics, statsd metrics are sent as soon as they can be. There is an internal buffer that stores them in memory and sends them as fast as the endpoint allows. Take care that the internal buffer doesn't fill the memory allowed for the process.

StatsD also supports sending over UDP or TCP. UDP just sends without a care if the message arrives or saturates the endpoint. TCP makes a connection and controls the speed at which you can send but, is vastly more expensive in compute resources.

See table for further details.

The below is an example of a list of StatsD endpoints.

```json
{
  "statsd": [
    {
      "id": "statsd1",
      "host": "localhost",
      "port": 8125,
      "transport": "udp",
      "buffer_depth": 1000
    },
    {
      "id": "statsd2",
      "host": "statsd.online.net",
      "port": 8125,
      "transport": "tcp",
      "buffer_depth": 1000
    }
  ],
}
```

Key | Type | Valid values | Description
---|---|---|---
statsd | `list` | NA | List of endpoint objects with values to describe the StatsD endpoint.
statsd.id | `string` | anything | A unique string used when sending events to a endpoint. You will need to put this into the event also.
statsd.host | `string` | anything | The hostname of the endpoint.
statsd.port | `uint16` | 1 - 65535 | Port number used to connect to the statsd endpoint.
statsd.transport | `string` | "tcp", "udp" | The network transport to use when sending the metrics.
statsd.buffer_depth | `int` | 1 - 32767 | How many metrics to send before slowing down internally to reduce creation speed.

#### Timelines

Time lines are a list of time slices that each have events in them. Each time slice is played out in sequence until the end. If the global configuration states that the time lines are continuous then the time lines start again. Else once complete the metric generator will exit.

```json
"timelines": [
  {
    "timeline_name": "my timeline name",
    "time_slices": [
      //time_slices
    ]
  }
]
```

Key | Type | Description
---|---|---
timelines | `list` | A list of timeline objects.
timelines.timeline_name | `string` | The name of a timeline. Used in the logs to identify events.
timelines.time_slices | `list` | A list of time slices.

#### Time slices

Time slices are like chapters, they are a section of time that has a series of events in it. It has a name, the number of times it must happen before moving onto the next time slice and also a marker to state if it is a single use time slice.

Time slices will play out the events in the order given. The events will repeat as many times as required specified by the repeat option. Once the events have played out the time slice will play them again and again until it has played them `repeat` number of times. If you just want to play them once, then put 1 on the repeat option.

Time slices that are single use will play out the events and repeat as many times as specified but not happen again after completing. This allows you to create startup events such annotations that a service has started.

```json
"timelines": [
  {
    "timeline_name": "my timeline name",
    "time_slices": [
      {
        "time_slice_name": "chapter 1",
          "repeat": 1,
          "single_use": true,
          "events": [
            // start up event
          ]
      },
      {
        "time_slice_name": "chapter 1",
          "repeat": 1,
          "events": [
            // send metrics
          ]
      }
    ]
  }
]
```

Key | Type | Description
---|---|---
time_slice_name | string | A descriptive name for the slice of time.
repeat | int | how many times should the events play out before moving on.
single_use | Should the time slice be used more than once.
events | A list of events that send the metrics.

#### Events

Events are what happens in your story. Each event sends a metric or sleeps for a period of time and repeats itself a number of times. Events have a rate of execution which enables you to make the metrics send fast or slowly. The rate is measured in milliseconds and is controlled by the `time_between` option.

An event has a `repeat` and a `time_between` timer value. Timers are measured in milliseconds. An event will do its action repeatedly until it has done it as many times as `repeat` states, it will sleep `time_between` milliseconds each time before doing the action again.

To make for more realistic graphs we have 2 timers available. Static and Dynamic.

Static timers sleep for the same amount of milliseconds each time.

Dynamic timers will sleep for a minimum time as well as a random amount between 1 and `vary` value milliseconds further. eg. If the minimum was 100 and the vary was 200, you would expect a random time of between 101 to 299ms sleeping to occur.

Event have a type that can be either "influx", "statsd" or "sleep". Influx and statsd correspond to a InfluxDB server or StatsD endpoint. A sleeper is used when you don't want any metrics to be sent for a period of time.

The `connection_id` must correspond with a Influx server or StatsD endpoint.

If your metric is a statsd metric then you MUST have a tag called `metric_type`. See [telegraf - statsd input](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/statsd#measurements) for a good explanation.

Ultimately all metrics will end up in some time series database and therefore will need to conform to its typing. This application was built with InfluxDB in mind. Therefore the tags and fields need to conform to InfluxDB types.

Tags keys and values need to be strings.

Tags need to be formatted correctly for the endpoint that you want to use. Common tagging and thus supported tagging formats are [datadog](https://docs.datadoghq.com/developers/dogstatsd/datagram_shell/) and [influx](https://www.influxdata.com/blog/getting-started-with-sending-statsd-metrics-to-telegraf-influxdb/).

Fields keys need to be strings and the values can only be ints, floats, strings or bools.

```json
{
  "timelines": [
    {
      "timeline_name": "first_timeline",
      "time_slices": [
        {
          "time_slice_name": "example",
          "events": [
            {
              "metric_name": "influx_test1",
              "type": "influx",
              "connection_id": "influx1",
              "tags": {
                "tag1": "value1",
                "tag2": "value2"
              },
              "fields": {
                "counter": 1
              },
              "repeat": 20,
              "time_between": {
                "dynamic": {
                  "minimum_time": 100,
                  "vary": 50
                }
              }
            },
            {
              "metric_name": "statsd_test1",
              "type": "statsd",
              "statsd_tagging_format": "influx",
              "connection_id": "statsd1",
              "tags": {
                "tag1": "value1",
                "tag2": "value2",
                "metric_type": "count"
              },
              "fields": {
                "counter": 1
              },
              "repeat": 20,
              "time_between": {
                "static": {
                  "time": 100
                }
              }
            },
            {
              "metric_name": "sleep1",
              "type": "sleeper",
              "repeat": 1,
              "time_between": {
                "static": {
                  "time": 10000
                }
              }
            }
          ]
        }
      ]
    }
  ]
}
```

Key | Type | Valid values | Description
---|---|---|---
event.metric_name | `string` | anything | The events metric name. This is used to create the metric in the selected system. 
event.type | `string` | "statsd", "influx" or "sleeper" | The type of event you are making. It can be influx or statsd to send a metric and a sleeper if you want to create a gap in time where nothing happens.
event.statsd_tagging_format | `string` | `influx` or `datadog` | The tagging format that you would like to use for statsd. The default is datadog tagging.
event.connection_id | `string` | ID of statsd or influx connection | Links the event to a statsd endpoint or influx server. 
event.tags | `list` | map[string]string | A key value list that has strings as both the keys and values. These are the tags for this metric. If you create a statsd metric you MUST have a "metric_type" key with a valid statsd metric type here. See [telegraf - statsd input](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/statsd#measurements) for metric types.
event.fields | `list` | map[string](ints, floats, string or bool) | A list of key value pairs. The key must be a string and the value can be either ints, floats string or a bool. Multiple values will cause multiple statsd metrics to be fired. Influx will gather them into a single data point.  
event.repeat | `int` | 1 - 32767 | How many times the event should repeat itself. The order is fire, sleep, fire, sleep, etc...
event.time_between | `static timer` or `dynamic timer` | NA | A static or dynamic timer is defined here.
event.time_between.dynamic | `dynamic timer` | NA | A dynamic timer is being defined for this metric. If you define both then the static timer will take precedence.
event.time_between.dynamic.minimum_time | `int` | 1 - 32767 | Number of milliseconds to wait at a minimum.
event.time_between.dynamic.vary | `int` | 1 - 32767 | A random number between 1 and this value will be added to the minimum sleep value of a dynamic timer.
event.time_between.static | `static timer` | NA | A static timer is about to be defined.
event.time_between.static.time | `int` | 1 - 32767 | Number of milliseconds to sleep for.

#### All together

As you can see each section of the configuration controls a aspect of the story that you want your metrics to tell. You need each section to be able to tell your story correctly.

To get started you can used the `-e` flag to get an example configuration file which you can then edit to suit your needs.

## Contributing

Create an issue.

Or if you want to contribute code: create an issue describing your requirements, create a PR and link it to your issue.