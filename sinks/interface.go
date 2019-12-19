package sinks

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	log "k8s.io/klog"
)

const (
	sink             string = "SINK"
	logGroupNameEnv  string = "CW_LOG_GROUP_NAME"
	logStreamNameEnv string = "CW_LOG_STREAM_NAME"
)

// EventSinkInterface is the interface used to shunt events
type EventSinkInterface interface {
	UpdateEvents(eNew *v1.Event, eOld *v1.Event)
}

// ManufactureSink will manufacture a sink according to viper configs
func ManufactureSink(ctx context.Context) (e EventSinkInterface) {
	s, ok := os.LookupEnv(sink)
	if !ok || s == "" {
		log.Warningf("SINK is not set! Setting it to CloudWatchLogs")
		viper.SetDefault("SINK", "CWL")
		s = viper.GetString("SINK")
	}
	log.Infof("Sink is [%v]", s)
	switch s {
	case "stdoutsink":
		e = NewStdoutSink(ctx)

	case "CWL":
		logGroupName, ok := os.LookupEnv(logGroupNameEnv)
		if !ok || logGroupName == "" {
			log.Exitf("Missing CWL Log Group, please set CW_LOG_GROUP_NAME Env variable")
		}

		logStreamName, ok := os.LookupEnv(logStreamNameEnv)
		if !ok || logGroupName == "" {
			log.Exitf("Missing CWL Log Stream, please set CW_LOG_STREAM_NAME Env variable")
		}
		// By default we buffer up to 1500 events, and drop messages if more than
		// 1500 have come in without getting consumed
		viper.SetDefault("sinkBufferSize", 1500)
		viper.SetDefault("sinkDiscardMessages", true)

		viper.SetDefault("sinkUploadInterval", 5)
		uploadInterval := viper.GetInt("sinkUploadInterval")

		bufferSize := viper.GetInt("sinkBufferSize")
		overflow := viper.GetBool("sinkDiscardMessages")

		cwl, err := NewCWLSink(logGroupName, logStreamName, uploadInterval, overflow, bufferSize)
		if err != nil {
			log.Fatal(err.Error())
		}

		go cwl.Run(make(chan bool))
		return cwl

	// case "logfile"
	default:
		err := errors.New("Invalid Sink Specified")
		log.Fatalf("%v, Sink variable not set, exiting program...", err.Error())
	}
	return e
}
