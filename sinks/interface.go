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
	viper.SetDefault("sink", "stdoutsink")
	s := viper.GetString("sink")
	log.Infof("Sink is [%v]", s)
	switch s {
	case "stdoutsink":
		e = NewStdoutSink(ctx)

	/*
		case "cloudwatchsink":
		region := viper.GetString("AWS_REGION")
		if region == "" {
			log.Warningf("Region is not specified, picking default")
		}

		bucket := viper.GetString("s3SinkBucket")
		if bucket == "" {
			log.Fatal("s3 sink specified but s3SinkBucket not specified")
		}

		bucketDir := viper.GetString("s3SinkBucketDir")
		if bucketDir == "" {
			log.Fatal("s3 sink specified but s3SinkBucketDir not specified")
		}

		// By default the json is pushed to s3 in not flatenned rfc5424 write format
		// The option to write to s3 is in the flattened json format which will help in
		// using the data in redshift with least effort
		viper.SetDefault("s3SinkOutputFormat", "rfc5424")
		outputFormat := viper.GetString("s3SinkOutputFormat")
		if outputFormat != "rfc5424" && outputFormat != "flatjson" {
			log.Fatal("s3 sink specified, but incorrect s3SinkOutputFormat specifed. Supported formats are: rfc5424 (default) and flatjson")
		}

		// By default we buffer up to 1500 events, and drop messages if more than
		// 1500 have come in without getting consumed
		viper.SetDefault("s3SinkBufferSize", 1500)
		viper.SetDefault("s3SinkDiscardMessages", true)

		viper.SetDefault("s3SinkUploadInterval", 120)
		uploadInterval := viper.GetInt("s3SinkUploadInterval")

		bufferSize := viper.GetInt("s3SinkBufferSize")
		overflow := viper.GetBool("s3SinkDiscardMessages")

		s3, err := NewS3Sink(region, bucket, bucketDir, uploadInterval, overflow, bufferSize, outputFormat)
		if err != nil {
			log.Fatal(err.Error())
		}

		go s3.Run(make(chan bool))
		return s3
	*/

	// case "logfile"
	default:
		err := errors.New("Invalid Sink Specified")
		log.Fatalf("%v, Sink variable not set, exiting program...", err.Error())
	}
	return e
}
