package controllers

import "time"

const (
	ShardOwnerLabel    = "starboard-exporter.giantswarm.io/shard-owner"
	DefaultServiceName = "starboard-exporter"
)

var DefaultRequeueDuration = (time.Minute * 5)
