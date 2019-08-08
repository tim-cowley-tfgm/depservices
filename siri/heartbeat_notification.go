package siri

import "time"

type HeartbeatNotification struct {
	RequestTimestamp      *time.Time
	ProducerRef           *string
	Status                *bool
	ValidUntil            *time.Time
	ShortestPossibleCycle *string
	ServiceStartedTime    *time.Time
}
