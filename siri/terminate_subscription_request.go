package siri

type TerminateSubscriptionRequest struct {
	*BaseRequest
	RequestorRef    *string
	All             *bool
	SubscriptionRef []*string
}
