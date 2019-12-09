package sinks

import (
	"context"
	"fmt"
	"sync"

	jsoniter "github.com/json-iterator/go"

	v1 "k8s.io/api/core/v1"
	log "k8s.io/klog"
)

// let's have few go routines to emit out glog.
const concurrency = 5

// UpdateEvent could be send via channels too
type UpdateEvent struct {
	eNew *v1.Event
	eOld *v1.Event
}

// StdOutSink is the most basic sink
type StdOutSink struct {
	updateChan chan UpdateEvent
}

// NewStdoutSink will create a new
func NewStdoutSink(ctx context.Context) EventSinkInterface {
	ss := &StdOutSink{
		updateChan: make(chan UpdateEvent),
	}

	var wg sync.WaitGroup
	wg.Add(concurrency)
	log.V(3).Infof("Starting glog sink with concurrency=%d", concurrency)
	// let's have couple of parallel routines
	go func() {
		defer wg.Done()
		ss.updateEvents(ctx)
	}()

	// wait
	go func() {
		wg.Wait()
		log.V(3).Info("Stopping glog sink WaitGroup")
		close(ss.updateChan)
	}()

	return ss
}

// UpdateEvents implements the EventSinkInterface.
// This is not a non-blocking call because the channel could get full. But ATM I do not care because
// glog just logs the message. It is CPU heavy (JSON Marshalling) and has no I/O. So the time complexity of the
// blocking call is very minimal. Also we could spawn more routines of updateEvents to make it concurrent.
func (ss *StdOutSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	ss.updateChan <- UpdateEvent{
		eNew: eNew,
		eOld: eOld,
	}
}

func (ss *StdOutSink) updateEvents(ctx context.Context) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	for {
		select {
		case event := <-ss.updateChan:
			eData := NewEventData(event.eNew, event.eOld)
			if eJSONBytes, err := json.Marshal(eData); err == nil {
				fmt.Println(string(eJSONBytes))
			} else {
				log.Warningf("Failed to json serialize event: %v", err)
			}
		}
	}
}
