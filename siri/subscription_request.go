package siri

import "time"

type SubscriptionRequest struct {
	*BaseRequest
	RequestorRef                            *string
	ConsumerAddress                         *string
	SubscriptionContext                     *SubscriptionContext
	ProductionTimetableSubscriptionRequest  []*ProductionTimetableSubscriptionRequest
	EstimatedTimetableSubscriptionRequest   []*EstimatedTimetableSubscriptionRequest
	StopMonitoringSubscriptionRequest       []*StopMonitoringSubscriptionRequest
	StopTimetableSubscriptionRequest        []*StopTimetableSubscriptionRequest
	VehicleMonitoringSubscriptionRequest    []*VehicleMonitoringSubscriptionRequest
	ConnectionTimetableSubscriptionRequest  []*ConnectionTimetableSubscriptionRequest
	ConnectionMonitoringSubscriptionRequest []*ConnectionMonitoringSubscriptionRequest
	GeneralMessageSubscriptionRequest       []*GeneralMessageSubscriptionRequest
}

type BaseSubscriptionRequest struct {
	SubscriberRef          *string
	SubscriptionIdentifier *string
	InitialTerminationTime *time.Time
}

type SubscriptionContext struct {
	HeartbeatInterval *string
}

type ProductionTimetableSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type EstimatedTimetableSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type StopTimetableSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type StopMonitoringSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type VehicleMonitoringSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type ConnectionTimetableSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type ConnectionMonitoringSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type GeneralMessageSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type SituationExchangeSubscriptionRequest struct {
	*BaseSubscriptionRequest
}

type FacilityMonitoringSubscriptionRequest struct {
	*BaseSubscriptionRequest
}
