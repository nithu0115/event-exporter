/*
Copyright 2017 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sinks

import (
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
func ManufactureSink() (e EventSinkInterface) {
	s := viper.GetString("sink")
	log.Infof("Sink is [%v]", s)
	switch s {
	case "":
		e = NewklogSink()
	case "s3sink":
		region := viper.GetString("s3SinkRegion")
		if region == "" {
			log.Warningf("s3 sink specified and region is not specified, picking default")
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

	// case "logfile"
	default:
		err := errors.New("Invalid Sink Specified")
		panic(err.Error())
	}
	return e
}
