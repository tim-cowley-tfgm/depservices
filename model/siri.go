package model

import (
	"time"
)

// Siri a representation of SIRI data
type Siri struct {
	ServiceDelivery ServiceDelivery
	ServiceRequest  ServiceRequest
}

// ErrorCondition a representation of a SIRI ErrorCondition item
type ErrorCondition struct {
	Description string
}

// Extensions a representation of a SIRI Extensions item
type Extensions struct {
	NationalOperatorCode string
}

// FramedVehicleJourneyRef a representation of a SIRI FramedVehicleJourneyRef item
type FramedVehicleJourneyRef struct {
	DataFrameRef           string
	DatedVehicleJourneyRef string
}

// MonitoredCall a representation of a SIRI MonitoredCall item
type MonitoredCall struct {
	StopPointRef              string
	Order                     int
	StopPointName             string
	VehicleAtStop             bool
	TimingPoint               bool
	AimedArrivalTime          time.Time
	ExpectedArrivalTime       time.Time
	ArrivalStatus             string
	ArrivalPlatformName       string
	ArrivalBoardingActivity   string
	AimedDepartureTime        time.Time
	ExpectedDepartureTime     time.Time
	DepartureStatus           string
	DeparturePlatformName     string
	DepartureBoardingActivity string
}

// MonitoredStopVisit a representation of a SIRI MonitoredStopVisit item
type MonitoredStopVisit struct {
	RecordedAtTime          time.Time
	MonitoringRef           string
	MonitoredVehicleJourney MonitoredVehicleJourney
	Extensions              Extensions
}

// MonitoredVehichleJourney a representation of a SIRI MonitoredVehicleJourney item
type MonitoredVehicleJourney struct {
	LineRef                     string
	DirectionRef                string
	FramedVehicleJourneyRef     FramedVehicleJourneyRef
	JourneyPatternRef           string
	DirectionName               string
	OperatorRef                 string
	OriginRef                   string
	OriginName                  string
	DestinationRef              string
	DestinationName             string
	VehicleJourneyName          string
	OriginAimedDepartureTime    time.Time
	DestinationAimedArrivalTime time.Time
	Monitored                   bool
	VehicleLocation             VehicleLocation
	BlockRef                    string
	VehicleRef                  string
	MonitoredCall               MonitoredCall
}

// ServiceDelivery a representation of a SIRI ServiceDelivery item
type ServiceDelivery struct {
	ResponseTimestamp      time.Time
	ProducerRef            string
	Status                 bool
	MoreData               bool
	StopMonitoringDelivery StopMonitoringDelivery
	ErrorCondition         ErrorCondition
}

// ServiceRequest a representation of a SIRI ServiceRequest item
type ServiceRequest struct {
	RequestTimestamp      time.Time
	RequestorRef          string
	StopMonitoringRequest StopMonitoringRequest
}

// StopMonitoringDelivery a representation of a SIRI StopMonitoringDelivery item
type StopMonitoringDelivery struct {
	ResponseTimestamp  time.Time
	Status             bool
	ValidUntil         time.Time
	MonitoredStopVisit []MonitoredStopVisit
}

// StopMonitoringRequest a representation of a SIRI StopMonitoringRequest item
type StopMonitoringRequest struct {
	RequestTimestamp  time.Time
	MonitoringRef     string
	PreviewInterval   string
	MaximumStopVisits int
}

// VehicleLocation a representation of a SIRI VehicleLocation item
type VehicleLocation struct {
	Longitude float64
	Latitude  float64
}
