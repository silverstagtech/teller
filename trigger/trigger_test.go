package trigger

import (
	"testing"
	"time"

	"github.com/silverstagtech/loggos"
)

func setupLogger() {
	loggos.JSONLoggerEnableDebugLogging(true)
	loggos.JSONLoggerEnablePrettyPrint(true)
	loggos.JSONLoggerEnableHumanTimestamps(true)
}

func TestTriggerStatic(t *testing.T) {
	setupLogger()

	trigger := New("tester", true)
	index := trigger.NewTimeSlice("test", 1, false)
	trigger.AddStaticTrigger(index, "test", 1, 4)
	trigger.Start()
	count := 0
	starttime := time.Now()
	for {
		select {
		case <-trigger.Ready:
			count++
		}
		if count == 4 {
			break
		}
	}
	stoptime := time.Since(starttime)
	<-trigger.Stop()

	if stoptime < time.Duration(time.Millisecond*4) || stoptime > time.Duration(time.Millisecond*10) {
		t.Logf("TestTriggerStatic took %s", stoptime)
		t.Logf("TestTriggerStatic failed because time is not in the sweet spot. This could indicate the computer testing is heavily loaded or a bug in timing.")
		t.Fail()
	}
}

func TestTriggerDynamic(t *testing.T) {
	setupLogger()

	trigger := New("tester", true)
	index := trigger.NewTimeSlice("test", 1, false)
	trigger.AddDynamicTrigger(index, "test", 1, 5, 4)
	trigger.Start()
	count := 0
	starttime := time.Now()
	for {
		select {
		case <-trigger.Ready:
			count++
		}
		if count == 4 {
			break
		}
	}
	stoptime := time.Since(starttime)
	<-trigger.Stop()

	if stoptime < time.Duration(time.Millisecond*8) || stoptime > time.Duration(time.Millisecond*30) {
		t.Logf("TestTriggerDynamic failed because time is not in the sweet spot. This could indicate the computer testing is heavily loaded or a bug in timing.")
		t.Logf("Dynamic Trigger took: %s", stoptime)
		t.Fail()
	}
}

func TestMultipleTriggers(t *testing.T) {
	setupLogger()

	trigger := New("tester", true)
	index := trigger.NewTimeSlice("test", 1, false)

	trigger.AddStaticTrigger(index, "static1", 1, 1)
	trigger.AddStaticTrigger(index, "static2", 5, 1)
	trigger.AddDynamicTrigger(index, "dyn1", 1, 3, 1)
	trigger.Start()

	found := map[string]bool{}
	count := 0
	for {
		select {
		case id := <-trigger.Ready:
			found[id] = true
			count++
		}
		if count == 20 {
			break
		}
	}
	<-trigger.Stop()

	if !found["static1"] || !found["static2"] || !found["dyn1"] {
		// There is a possible race condition in this test.
		// If the trigger never gets a chance to put its value on the queue
		// it would never come through. However this is extremely unlikely
		// and would not be a real sign of failure. Rather we want to know if
		// the value never ever reaches the ready queue. I'm happy to live
		// with this risk as it would be just re-run tests.
		t.Logf("TestMultipleTriggers failed to find one of the ids.")
		t.Fail()
	}
}

func TestNoTriggers(t *testing.T) {
	setupLogger()

	trigger := New("tester", true)
	err := trigger.Start()
	if err == nil {
		t.Logf("TestNoTriggers created a trigger with no timers and didn't get an error.")
		t.Fail()
	}
}
