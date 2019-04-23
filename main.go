package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/silverstagtech/loggos"
	"github.com/silverstagtech/teller/config"
	"github.com/silverstagtech/teller/orchestrator"
)

const (
	version = "0.0.1"
)

var (
	configLocation    = flag.String("c", "./config.json", "The configuration file for the test. The configuration should tell the story in timelines that oyu want to send to the metric systems.")
	exampleConfigFlag = flag.Bool("e", false, "Print a example json configuration to the terminal.")
	versionFlag       = flag.Bool("v", false, "Shows the version of the application.")
	helpFlag          = flag.Bool("h", false, "Shows this help menu.")
)

func main() {
	// consume flags and test for show stoppers.
	flag.Parse()
	showStoppers()

	// create the logger
	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	decorations := map[string]interface{}{
		"hostname": hostname,
		"version":  version,
	}
	loggos.JSONLoggerAddDecoration(decorations)
	loggos.SendJSON(loggos.JSONInfoln("Starting teller"))

	// Collect configuration file.
	config, err := config.New(*configLocation)
	if err != nil {
		jm := loggos.JSONCritln("Failed to read configuration.")
		jm.Error(err)
		loggos.SendJSON(jm)

		terminate(1)
	}
	// start single run
	// start continuous run
	orchestrator := orchestrator.New(signals, config)
	err = orchestrator.Start()
	if err != nil {
		jm := loggos.JSONCritln("Failed to start the timelines")
		jm.Error(err)
		loggos.SendJSON(jm)
		terminate(1)
	}
	<-orchestrator.StopChan
	loggos.SendJSON(loggos.JSONInfoln("Finished successfully!"))
	terminate(0)
}

func showStoppers() {
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	if *helpFlag {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *exampleConfigFlag {
		fmt.Println(config.ExampleConfig())
		os.Exit(0)
	}
}

// terminate is used to exit but also flush the logger
func terminate(exitNumber int) {
	<-loggos.Flush()
	os.Exit(exitNumber)
}
