package trigger

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/silverstagtech/loggos"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type timer func() chan string

type timeslice struct {
	repeat       int
	name         string
	singleUse    bool
	allowedToRun bool
	timers       []timer
}

func (tl *timeslice) String() string {
	return fmt.Sprintf("Name: %s, Repeat: %d, Number of timers: %d", tl.name, tl.repeat, len(tl.timers))
}

// Trigger has timer trigger functions that will execute and send messages down
// the Ready channel to let the user know when a trigger has been pulled.
type Trigger struct {
	name       string
	Ready      chan string
	timeslices []*timeslice
	shutdown   chan bool
	continuous bool
	stopped    bool
}

// New creates a new Trigger and returns it. You will need to populate it with triggers,
// then call Start() and Stop() when finished.
func New(name string, continuous bool) *Trigger {
	return &Trigger{
		Ready:      make(chan string, 5000),
		shutdown:   make(chan bool, 1),
		timeslices: make([]*timeslice, 0),
		continuous: continuous,
		name:       name,
	}
}

// Start will cycle through the added triggers and fill the Ready channel with the ids as they
// come through. If there are no triggers it returns an error.
func (tr *Trigger) Start() error {
	jm := loggos.JSONDebugln("Starting timeline")
	jm.Add("name", tr.name)
	loggos.SendJSON(jm)

	if len(tr.timeslices) == 0 {
		return fmt.Errorf("There are no timers to run")
	}
	go tr.pullTriggers()
	return nil
}

// Stop will signal that the triggering service needs to stop processing triggers.
// It will return a chan bool that will get a true once completely shutdown.
// The chan is closed after.
func (tr *Trigger) Stop() chan bool {
	jm := loggos.JSONDebugln("Stopping timeline")
	jm.Add("name", tr.name)
	loggos.SendJSON(jm)

	c := make(chan bool)
	go func() {
		tr.teardown()
		c <- true
		close(c)
	}()
	return c
}

func (tr *Trigger) teardown() {
	if !tr.stopped {
		tr.stopped = true
		close(tr.shutdown)
		close(tr.Ready)
	}
}

func (tr *Trigger) hasNothingToDo() bool {
	if tr.stopped {
		return true
	}

	// If any timeline can still run then there is something to do.
	for _, timeline := range tr.timeslices {
		if timeline.allowedToRun {
			return false
		}
	}

	// If we get here there there is nothing to do.
	return true
}

// pullTriggets will go though each of the triggers and execute the function which will
// then feed the Ready chan. If the stopChan is closed then will exit out.
func (tr *Trigger) pullTriggers() {
	for {
		// Is there still work to do?
		if tr.hasNothingToDo() {
			jm := loggos.JSONDebugln("Timeline trigger has nothing to do. Stopping.")
			jm.Add("name", tr.name)
			loggos.SendJSON(jm)

			tr.teardown()
			return
		}

		// Go through the timeslices and check for the ones that have events to send.
	Top:
		for _, timeslice := range tr.timeslices {
			if timeslice.allowedToRun {
				for i := 0; i < timeslice.repeat; i++ {
					for _, timer := range timeslice.timers {
						if tr.stopped {
							break Top
						}
						jm := loggos.JSONDebugln("Starting next timer for time slice")
						jm.Add("name", tr.name)
						jm.Add("timeslice_name", timeslice.name)
						loggos.SendJSON(jm)

						tc := timer()
						tr.consumeFromTimer(tc)
					}
				}
			}

			// If a timeslice is single mark it as used.
			if timeslice.singleUse && timeslice.allowedToRun {
				jm := loggos.JSONDebugln("Time slice is marked as single use. Stopping future runs")
				jm.Add("name", tr.name)
				jm.Add("timeslice_name", timeslice.name)
				loggos.SendJSON(jm)

				timeslice.allowedToRun = false
			}
		}

		if !tr.continuous {
			jm := loggos.JSONInfoln("Timeline is a single run and is now finished")
			jm.Add("name", tr.name)
			loggos.SendJSON(jm)

			tr.teardown()
			return
		}
	}
}

func (tr *Trigger) consumeFromTimer(tc chan string) {
	if tr.stopped {
		return
	}

	for {
		select {
		case _, ok := <-tr.shutdown:
			if !ok {
				jm := loggos.JSONDebugln("Timeline reader is finished and about to stop.")
				jm.Add("name", tr.name)
				loggos.SendJSON(jm)

				tr.teardown()
				return
			}
		case id, ok := <-tc:
			if !ok {
				jm := loggos.JSONDebugln("Finished with trigger.")
				jm.Add("name", tr.name)
				loggos.SendJSON(jm)

				return
			}
			tr.Ready <- id
		}
	}
}

// AddStaticTrigger will created a trigger that always fires at the same time in milliseconds.
func (tr *Trigger) AddStaticTrigger(timesliceIndex int, id string, ms, repeat int) {
	jm := loggos.JSONDebugln("Adding static trigger to the queue")
	jm.Add("id", id)
	jm.Add("name", tr.name)
	jm.Add("milliseconds", ms)
	loggos.SendJSON(jm)

	f := func() chan string {
		c := make(chan string, 1)
		go func() {
			for i := 0; i < repeat; i++ {
				time.Sleep(time.Millisecond * time.Duration(ms))
				c <- id
			}
			jm := loggos.JSONDebugln("Trigger is finished.")
			jm.Add("id", id)
			jm.Add("name", tr.name)
			loggos.SendJSON(jm)

			close(c)
		}()
		return c
	}
	timeslice := tr.timeslices[timesliceIndex]
	timeslice.timers = append(timeslice.timers, f)
}

// AddDynamicTrigger will create a trigger that fires between minMS and varyMS
// milliseconds time.
func (tr *Trigger) AddDynamicTrigger(timesliceIndex int, id string, minMS, varyMS, repeat int) {
	jm := loggos.JSONDebugln("Adding dynamic trigger to the queue")
	jm.Add("name", tr.name)
	jm.Add("id", id)
	jm.Add("minimum_time", minMS)
	jm.Add("maximum_time", minMS+varyMS)
	loggos.SendJSON(jm)

	f := func() chan string {
		c := make(chan string, 1)
		go func() {
			sleeperTime := func() time.Duration {
				return time.Millisecond * time.Duration(minMS+rand.Intn(varyMS))
			}
			for i := 0; i < repeat; i++ {
				time.Sleep(sleeperTime())
				c <- id
			}
			jm := loggos.JSONDebugln("Trigger is finished.")
			jm.Add("id", id)
			jm.Add("name", tr.name)
			loggos.SendJSON(jm)
			close(c)
		}()
		return c
	}
	timeslice := tr.timeslices[timesliceIndex]
	timeslice.timers = append(timeslice.timers, f)
}

// NewTimeSlice creates a new timeslice and adds it to the list of timeslices in the trigger.
// It will return the Index number for the timeslice. To add to this timeslice us the index
// in the add timer functions.
func (tr *Trigger) NewTimeSlice(name string, repeat int, singleUse bool) int {
	tr.timeslices = append(tr.timeslices,
		&timeslice{
			name:         name,
			repeat:       repeat,
			singleUse:    singleUse,
			allowedToRun: true,
		},
	)
	return len(tr.timeslices) - 1
}
