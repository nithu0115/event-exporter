package sinks

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	log "k8s.io/klog"
)

//StdOutSink implements eventch and buffer
type StdOutSink struct {
	// eventCh is used to interact eventRouter and the sharedInformer
	eventCh channels.Channel

	//bodyBuf
	bodyBuf *bytes.Buffer
}

//NewStdOutSink factory sink
func NewStdOutSink(overflow bool, bufferSize int) *StdOutSink {
	s := &StdOutSink{
		bodyBuf: bytes.NewBuffer(make([]byte, 0, 4096)),
	}

	if overflow {
		s.eventCh = channels.NewOverflowingChannel(channels.BufferCap(bufferSize))
	} else {
		s.eventCh = channels.NewNativeChannel(channels.BufferCap(bufferSize))
	}

	return s
}

// UpdateEvents implements the EventSinkInterface. It really just writes the
// event data to the event OverflowingChannel, which should never block.
// Messages that are buffered beyond the bufferSize specified for this sink
// are discarded.
func (s *StdOutSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	s.eventCh.In() <- NewEventData(eNew, eOld)
}

// Run sits in a loop, waiting for data to come in through s.eventCh,
// and forwarding them to the HTTP sink. If multiple events have happened
// between loop iterations, it puts all of them in one request instead of
// making a single request per event.
func (s *StdOutSink) Run(stopCh <-chan bool) {
loop:
	for {
		select {
		case e := <-s.eventCh.Out():
			var evt EventData
			var ok bool
			if evt, ok = e.(EventData); !ok {
				glog.Warningf("Invalid type sent through event channel: %T", e)
				continue loop
			}

			// Start with just this event...
			arr := []EventData{evt}

			// Consume all buffered events into an array, in case more have been written
			// since we last forwarded them
			numEvents := s.eventCh.Len()
			for i := 0; i < numEvents; i++ {
				e := <-s.eventCh.Out()
				if evt, ok = e.(EventData); ok {
					arr = append(arr, evt)
				} else {
					glog.Warningf("Invalid type sent through event channel: %T", e)
				}
			}

			s.drainEvents(arr)
		case <-stopCh:
			break loop
		}
	}
}

// drainEvents takes an array of event data and sends it to the receiving event hub.
func (s *StdOutSink) drainEvents(events []EventData) {
	for _, evt := range events {
		eJSONBytes, err := json.Marshal(evt)
		if err == nil {
			fmt.Println(string(eJSONBytes))
		} else {
			log.Warningf("Failed to json serialize event: %v", err)
		}
	}
}
