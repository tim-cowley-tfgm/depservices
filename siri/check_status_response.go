package siri

import "time"

type CheckStatusResponse struct {
	*BaseResponse
	Status                *bool
	ValidUntil            *time.Time
	ShortestPossibleCycle *string
	ServiceStartedTime    *time.Time
}
