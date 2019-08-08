package siri

import "time"

type ServiceDelivery struct {
	*BaseResponse
	ProducerRef                 *string
	Status                      *bool
	MoreData                    *bool
	EstimatedTimetableDelivery  []*EstimatedTimetableDelivery
	ProductionTimetableDelivery []*ProductionTimetableDelivery
	StopMonitoringDelivery      []*StopMonitoringDelivery
	StopTimetableDelivery       []*StopTimetableDelivery
	VehicleMonitoringDelivery   []*VehicleMonitoringDelivery
}

type BaseServiceDelivery struct {
	ResponseTimestamp *time.Time
	SubscriberRef     *string
	SubscriptionRef   *string
	Status            *bool
}

type EstimatedTimetableDelivery struct {
	*BaseServiceDelivery
}

type ProductionTimetableDelivery struct {
	*BaseServiceDelivery
}

type StopMonitoringDelivery struct {
	*BaseServiceDelivery
}

type StopTimetableDelivery struct {
	*BaseServiceDelivery
}

type VehicleMonitoringDelivery struct {
	*BaseServiceDelivery
}
