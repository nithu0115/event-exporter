package sinks

import (
	"encoding/json"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
)

// StdoutSink is the other basic sink
// By default, Fluentd/ElasticSearch won't index glog formatted lines
// By logging raw JSON to stdout, we will get automated indexing which
// can be queried in Kibana.
type StdoutSink struct {
	// TODO: create a channel and buffer for scaling
	namespace string
}

// NewStdoutSink will create a new StdoutSink with default options, returned as
// an EventSinkInterface
func NewStdoutSink(namespace string) EventSinkInterface {
	return &StdoutSink{
		namespace: namespace}
}

// UpdateEvents implements the EventSinkInterface
func (gs *StdoutSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	eData := NewEventData(eNew, eOld)

	if len(gs.namespace) > 0 {
		namespacedData := map[string]interface{}{}
		namespacedData[gs.namespace] = eData
		if eJSONBytes, err := json.Marshal(namespacedData); err == nil {
			fmt.Println(string(eJSONBytes))
		} else {
			fmt.Fprintf(os.Stderr, "Failed to json serialize event: %v", err)
		}
	} else {
		if eJSONBytes, err := json.arshal(eData); err == nil {
			fmt.Println(string(eJSONBytes))
		} else {
			fmt.Fprintf(os.Stderr, "Failed to json serialize event: %v", err)
		}
	}
}
