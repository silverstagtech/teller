package orchestrator

import (
	"fmt"

	"github.com/silverstagtech/teller/config"
	"github.com/silverstagtech/teller/trigger"
)

type timeline struct {
	Name     string
	trigger  *trigger.Trigger
	StopChan chan bool
	events   map[string]*eventMetric
}

// eventName return the next index number in the events slice.
func (tl *timeline) eventName(timesliceName string, eventNumber int) string {
	return fmt.Sprintf("%q timeslice %q event %d", tl.Name, timesliceName, eventNumber)
}

func (tl *timeline) addEventTrigger(timesliceName string, timesliceIndex, eventIndex int, event *config.Event) string {
	id := tl.eventName(timesliceName, eventIndex)
	if event.TimeBetween.Static.Time > 0 {
		tl.trigger.AddStaticTrigger(timesliceIndex, id, event.TimeBetween.Static.Time, event.Repeat)
	} else if event.TimeBetween.Dynamic.MinimumTime > 0 && event.TimeBetween.Dynamic.Vary > 0 {
		tl.trigger.AddDynamicTrigger(timesliceIndex, id, event.TimeBetween.Dynamic.MinimumTime, event.TimeBetween.Dynamic.Vary, event.Repeat)
	}
	return id
}

func (tl *timeline) startFiring() {
	go func() {
		for {
			select {
			case id, ok := <-tl.trigger.Ready:
				if !ok {
					tl.StopChan <- true
					close(tl.StopChan)
					return
				}
				tl.events[id].fire()
			}
		}
	}()
}

func (tl *timeline) shutdown() {
	<-tl.trigger.Stop()
}
