package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/silverstagtech/loggos"
)

// Config contains a logger and a story. Use the story to run the shippers.
type Config struct {
	configPath string
	Story      *Story
}

// New creates a configuration that contains the story of the configuration file define by
// the user. It will return a *Config and a list of errors that it found along the way.
// These could be failed to open file or configuration validation failures.
func New(filepath string) (*Config, error) {
	c := &Config{
		configPath: filepath,
	}

	file, err := c.readConfigFile()
	if err != nil {
		return nil, err
	}

	story, err := c.newStory(file)
	if err != nil {
		return nil, err
	}

	loggos.JSONLoggerEnableDebugLogging(story.DebugLogging)
	loggos.JSONLoggerEnablePrettyPrint(true)
	loggos.JSONLoggerEnableHumanTimestamps(true)

	err = c.validate(story)
	if err != nil {
		return nil, err
	}

	c.Story = story
	c.shimGlobalTags()
	return c, nil
}

func (cf *Config) readConfigFile() ([]byte, error) {
	file, err := ioutil.ReadFile(cf.configPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open the supplied configuration file. Error %s", err)
	}
	if len(file) == 0 {
		return nil, fmt.Errorf("configuration file was empty")
	}
	return file, nil
}

// NewStory takes a new
func (cf *Config) newStory(config []byte) (*Story, error) {
	story := new(Story)
	err := json.Unmarshal(config, story)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a story. Error %s", err)
	}
	return story, nil
}

func (cf *Config) shimGlobalTags() {
	if len(cf.Story.GlobalTags) > 0 {
		// Go over each global flag and add it to each event.
		// This includes the sleepers. However since sleepers also have tags we don't really
		// need to worry about it.
		for GlobalTagKey, GlobalTagValue := range cf.Story.GlobalTags {
			// Go to each event in the timelines and shim in the extra tag
			for timelineIndex, timeline := range cf.Story.TimeLines {
				for timesliceIndex, timeslice := range timeline.Timeslices {
					for eventIndex := range timeslice.Events {
						if cf.Story.TimeLines[timelineIndex].Timeslices[timesliceIndex].Events[eventIndex].Tags != nil {
							cf.Story.TimeLines[timelineIndex].Timeslices[timesliceIndex].Events[eventIndex].Tags[GlobalTagKey] = GlobalTagValue
						}
					}
				}
			}
		}
	}
}

func (cf *Config) validate(story *Story) error {
	errorBucket := new(ValidationError)
	validateStory(*story, errorBucket)

	if len(errorBucket.errs) > 0 {
		return errorBucket
	}

	return nil
}

func (cf *Config) String() string {
	b, err := json.MarshalIndent(cf.Story, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error %s", err)
	}
	return string(b)
}
