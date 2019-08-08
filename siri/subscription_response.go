package siri

import "time"

type SubscriptionResponse struct {
	*BaseResponse
	ResponderRef   *string
	ResponseStatus []*ResponseStatus
}

type ResponseStatus struct {
	ResponseTimestamp     *time.Time
	SubscriptionRef       *string
	Status                *bool
	ValidUntil            *time.Time
	ShortestPossibleCycle *string
}
