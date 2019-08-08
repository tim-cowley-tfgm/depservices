package siri

import "time"

type Siri struct {
	CapabilitiesRequest           *CapabilitiesRequest
	CheckStatusRequest            *CheckStatusRequest
	CheckStatusResponse           *CheckStatusResponse
	DataReadyAcknowledgement      *DataReadyAcknowledgement
	DataReadyNotification         *DataReadyNotification
	DataSupplyRequest             *DataSupplyRequest
	HeartbeatNotification         *HeartbeatNotification
	LinesRequest                  *LinesRequest
	ProductCategoriesRequest      *ProductCategoriesRequest
	ServiceDelivery               *ServiceDelivery
	ServiceFeaturesRequest        *ServiceFeaturesRequest
	ServiceRequest                *ServiceRequest
	StopPointsRequest             *StopPointsRequest
	SubscriptionRequest           *SubscriptionRequest
	SubscriptionResponse          *SubscriptionResponse
	TerminateSubscriptionRequest  *TerminateSubscriptionRequest
	TerminateSubscriptionResponse *TerminateSubscriptionResponse
	VehicleFeaturesRequest        *VehicleFeaturesRequest
}

type BaseRequest struct {
	RequestTimestamp *time.Time
}

type BaseResponse struct {
	ResponseTimestamp *time.Time
}
