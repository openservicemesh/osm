package cds

import "github.com/golang/protobuf/ptypes/duration"

func getTimeout() *duration.Duration {
	return &duration.Duration{
		Seconds: 5,
	}
}
