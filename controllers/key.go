package controllers

import "time"

const (
	ShardOwnerLabel = "starboard-exporter.giantswarm.io/shard-owner"
)

var DefaultRequeueDuration = (time.Minute * 5)
