package sinks

import (
	"bytes"

	"github.com/eapache/channels"

	"github.com/eapache/channels"
	log "k8s.io/klog"
)

type StdOutSink struct {
	// eventCh is used to interact eventRouter and the sharedInformer
	eventCh channels.Channel

	//bodyBuf
	bodyBuf *bytes.Buffer
}




// UpdateEvents implements the EventSinkInterface. It really just writes the
// event data to the event OverflowingChannel, which should never block.
// Messages that are buffered beyond the bufferSize specified for this sink
// are discarded.
func (s *S3Sink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
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
	var messageSize int
	for _, evt := range events {
		if eJSONBytes, err := json.Marshal(evt); err != nil {
			glog.Warningf("Failed to serialize json: %v", err)
			return
		}
		log.Printf("%s", string(eJSONBytes))
		messageSize += len(eJSONBytes)
		if messageSize > maxMessageSize {
			h.sendBatch(evts)
			evts = nil
			messageSize = 0
		}
		evts = append(evts, eventhub.NewEvent(eJSONBytes))
	}
	h.sendBatch(evts)
}