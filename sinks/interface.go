package sinks

import (
	"context"
	"errors"

	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	log "k8s.io/klog"
)

// EventSinkInterface is the interface used to shunt events
type EventSinkInterface interface {
	UpdateEvents(eNew *v1.Event, eOld *v1.Event)
}

// ManufactureSink will manufacture a sink according to viper configs
// TODO: Determine if it should return an array of sinks
func ManufactureSink(ctx context.Context) (e EventSinkInterface) {
	viper.SetDefault("sink", "cloudwatchsink")
	s := viper.GetString("sink")
	log.Infof("Sink is [%v]", s)
	switch s {
	case "stdoutsink":
		e = NewStdoutSink(ctx)

	case "cloudwatchsink":
		viper.SetDefault("AWS_REGION", "us-east-1")
		region := viper.GetString("AWS_REGION")
		if region == "" {
			log.Warningf("Region is not specified, picking default")
		}

		viper.SetDefault("logGroupName", "testing-k8s-events")
		logGroup := viper.GetString("logGroupName")
		if logGroup == "" {
			log.Fatal("logGroupName is not specified")
		}

		viper.SetDefault("logStreamName", "testing")
		logStream := viper.GetString("logStreamName")
		if logStream == "" {
			log.Fatal("logStream is not specified")
		}

		// By default we buffer up to 1500 events, and drop messages if more than
		// 1500 have come in without getting consumed
		viper.SetDefault("sinkBufferSize", 1500)
		viper.SetDefault("sinkDiscardMessages", true)

		viper.SetDefault("sinkUploadInterval", 2)
		uploadInterval := viper.GetInt("sinkUploadInterval")

		bufferSize := viper.GetInt("sinkBufferSize")
		overflow := viper.GetBool("sinkDiscardMessages")

		cwl, err := NewS3Sink(logGroup, logStream, region, uploadInterval, overflow, bufferSize)
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
