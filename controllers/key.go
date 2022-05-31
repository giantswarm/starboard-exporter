package controllers

import "time"

const (
	DefaultServiceName = "starboard-exporter"
	ShardOwnerLabel    = "starboard-exporter.giantswarm.io/shard-owner"
)

var DefaultRequeueDuration = (time.Minute * 5)
