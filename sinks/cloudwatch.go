package sinks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	log "k8s.io/klog"
)

const perEventBytes = 26
const maxMessageSize = 1046528
const logStreamInactivityTimeout = time.Hour

/*
CWLSink is the sink that uploads the kubernetes events as json object stored in a file.
The sinker uploads it to CloudWatch Logs if any of the below criteria gets fullfilled
1) Time(uploadInterval): If the specfied time has passed since the last upload it uploads
2) [TODO] Data size: If the total data getting uploaded becomes greater than N bytes
*/
type CWLSink struct {
	//client from aws which makes the API call to CWL
	client        LogsClient
	logGroupName  string
	logStreamName string
	streams       map[string]*logStream

	// lastUploadTimestamp stores the timestamp when the last upload to CWL happened
	lastUploadTimestamp int64

	// uploadInterval tells after how many seconds the next upload can happen
	// sink waits till this time is passed before next upload can happen
	uploadInterval time.Duration
	// eventCh is used to interact eventRouter and the sharedInformer
	eventCh channels.Channel

	// bodyBuf stores all the event captured data in a buffer before upload
	bodyBuf *bytes.Buffer
}

// LogsClient contains the CloudWatch API calls used by this plugin
type LogsClient interface {
	PutLogEvents(input *cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error)
}

type logStream struct {
	logEvents         []*cloudwatchlogs.InputLogEvent
	currentByteLength int
	nextSequenceToken *string
	logStreamName     string
	expiration        time.Time
}

// NewS3Sink is the factory method constructing a new S3Sink
func NewS3Sink(logGroupName string, logStreamName string, region string, uploadInterval int, overflow bool, bufferSize int) (*CWLSink, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	client := cloudwatchlogs.New(sess)

	cwl := &CWLSink{
		logGroupName:   logGroupName,
		logStreamName:  logStreamName,
		client:         client,
		uploadInterval: time.Second * time.Duration(uploadInterval),
		streams:        make(map[string]*logStream),
		bodyBuf:        bytes.NewBuffer(make([]byte, 0, 4096)),
	}

	if overflow {
		cwl.eventCh = channels.NewOverflowingChannel(channels.BufferCap(bufferSize))
	} else {
		cwl.eventCh = channels.NewNativeChannel(channels.BufferCap(bufferSize))
	}

	return cwl, nil
}

// UpdateEvents implements the EventSinkInterface. It really just writes the
// event data to the event OverflowingChannel, which should never block.
// Messages that are buffered beyond the bufferSize specified for this HTTPSink
// are discarded.
func (cwl *CWLSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	cwl.eventCh.In() <- NewEventData(eNew, eOld)
}

// Run sits in a loop, waiting for data to come in through h.eventCh,
// and forwarding them to the HTTP sink. If multiple events have happened
// between loop iterations, it puts all of them in one request instead of
// making a single request per event.
func (cwl *CWLSink) Run(stopCh <-chan bool) {
loop:
	for {
		select {
		case e := <-cwl.eventCh.Out():
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
			numEvents := cwl.eventCh.Len()
			for i := 0; i < numEvents; i++ {
				e := <-cwl.eventCh.Out()
				if evt, ok = e.(EventData); ok {
					arr = append(arr, evt)
				} else {
					glog.Warningf("Invalid type sent through event channel: %T", e)
				}
			}

			cwl.drainEvents(arr)
		case <-stopCh:
			break loop
		}
	}
}

// drainEvents takes an array of event data and sends it to s3
func (cwl *CWLSink) drainEvents(events []EventData) {

	timestamp := time.Now()
	s := &logStream{
		logStreamName: "testing",
	}

	for _, evt := range events {
		var messageSize int
		eJSONBytes, err := json.Marshal(evt)
		if err != nil {
			glog.Warningf("Failed to flatten json: %v", err)
			return
		}
		messageSize += len(eJSONBytes)
		fmt.Println("size =====", messageSize)
		s.logEvents = append(s.logEvents, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(string(eJSONBytes)),
			Timestamp: aws.Int64(timestamp.UnixNano() / 1e6), // CloudWatch uses milliseconds since epoch
		})
		s.currentByteLength += cloudwatchLen(string(eJSONBytes))
	}

	if cwl.canUpload() == false {
		return
	}

	err := cwl.upload(s)
	if err != nil {
		log.Warning(err)
	}
}

// canUpload verifies the conditions suitable for a new file upload and upload the data
func (cwl *CWLSink) canUpload() bool {
	now := time.Now().UnixNano()
	if (cwl.lastUploadTimestamp + cwl.uploadInterval.Nanoseconds()) < now {
		return true
	}
	return false
}

// upload uploads the events stored in buffer to s3 in the specified key
// and clears the buffer
func (cwl *CWLSink) upload(stream *logStream) error {
	// Reuse the body buffer for each request
	cwl.bodyBuf.Truncate(0)

	now := time.Now()
	stream.updateExpiration()
	// Log events in a single PutLogEvents request must be in chronological order.
	sort.Slice(stream.logEvents, func(i, j int) bool {
		return aws.Int64Value(stream.logEvents[i].Timestamp) < aws.Int64Value(stream.logEvents[j].Timestamp)
	})
	response, err := cwl.client.PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
		LogEvents:     stream.logEvents,
		LogGroupName:  aws.String(cwl.logGroupName),
		LogStreamName: aws.String(stream.logStreamName),
		SequenceToken: stream.nextSequenceToken,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == cloudwatchlogs.ErrCodeDataAlreadyAcceptedException {
				// already submitted, just grab the correct sequence token
				parts := strings.Split(awsErr.Message(), " ")
				stream.nextSequenceToken = &parts[len(parts)-1]
				stream.logEvents = stream.logEvents[:0]
				stream.currentByteLength = 0
				log.Infof("[cloudwatch] Encountered error %v; data already accepted, ignoring error\n", awsErr)
				return nil
			} else if awsErr.Code() == cloudwatchlogs.ErrCodeInvalidSequenceTokenException {
				// sequence code is bad, grab the correct one and retry
				parts := strings.Split(awsErr.Message(), " ")
				stream.nextSequenceToken = &parts[len(parts)-1]

				return cwl.upload(stream)
			} else {
				return err
			}
		} else {
			return err
		}
	}
	cwl.processRejectedEventsInfo(response)
	cwl.lastUploadTimestamp = now.UnixNano()
	stream.logEvents = stream.logEvents[:0]

	return nil
}

func (cwl *CWLSink) processRejectedEventsInfo(response *cloudwatchlogs.PutLogEventsOutput) {
	if response.RejectedLogEventsInfo != nil {
		if response.RejectedLogEventsInfo.ExpiredLogEventEndIndex != nil {
			log.Warning("[cloudwatch] %d log events were marked as expired by CloudWatch\n", aws.Int64Value(response.RejectedLogEventsInfo.ExpiredLogEventEndIndex))
		}
		if response.RejectedLogEventsInfo.TooNewLogEventStartIndex != nil {
			log.Warning("[cloudwatch %d] %d log events were marked as too new by CloudWatch\n", aws.Int64Value(response.RejectedLogEventsInfo.TooNewLogEventStartIndex))
		}
		if response.RejectedLogEventsInfo.TooOldLogEventEndIndex != nil {
			log.Warning("[cloudwatch] %d log events were marked as too old by CloudWatch\n", aws.Int64Value(response.RejectedLogEventsInfo.TooOldLogEventEndIndex))
		}
	}
}

// effectiveLen counts the effective number of bytes in the string, after
// UTF-8 normalization.  UTF-8 normalization includes replacing bytes that do
// not constitute valid UTF-8 encoded Unicode codepoints with the Unicode
// replacement codepoint U+FFFD (a 3-byte UTF-8 sequence, represented in Go as
// utf8.RuneError)
func effectiveLen(line string) int {
	effectiveBytes := 0
	for _, rune := range line {
		effectiveBytes += utf8.RuneLen(rune)
	}
	return effectiveBytes
}

func cloudwatchLen(event string) int {
	return effectiveLen(event) + perEventBytes
}

func (stream *logStream) updateExpiration() {
	stream.expiration = time.Now().Add(logStreamInactivityTimeout)
}
