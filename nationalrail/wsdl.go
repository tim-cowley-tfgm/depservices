package nationalrail

import (
	"context"
	"encoding/xml"
	"github.com/hooklift/gowsdl/soap"
	"time"
)

// against "unused imports"
var _ time.Time
var _ xml.Name

// Manual additions to work with National Rail Enquiries
type SOAPHeader struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`

	Header interface{} `xml:",omitempty"`
}

type AccessToken struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2010-11-01/ldb/commontypes AccessToken"`

	TokenValue string `xml:"TokenValue"`
}

type GetDepartureBoardRequest GetBoardRequestParams

type GetArrivalBoardRequest GetBoardRequestParams

type GetArrivalDepartureBoardRequest GetBoardRequestParams

type GetServiceDetailsRequest GetServiceDetailsRequestParams

type GetDepBoardWithDetailsRequest GetBoardRequestParams

type GetArrBoardWithDetailsRequest GetBoardRequestParams

type GetArrDepBoardWithDetailsRequest GetBoardRequestParams

type GetNextDeparturesRequest GetDeparturesRequestParams

type GetNextDeparturesWithDetailsRequest GetDeparturesRequestParams

type GetFastestDeparturesRequest GetDeparturesRequestParams

type GetFastestDeparturesWithDetailsRequest GetDeparturesRequestParams

type GetDepartureBoardResponse StationBoardResponseType

type GetArrivalBoardResponse StationBoardResponseType

type GetArrivalDepartureBoardResponse StationBoardResponseType

type GetServiceDetailsResponse ServiceDetailsResponseType

type GetDepBoardWithDetailsResponse StationBoardWithDetailsResponseType

type GetArrBoardWithDetailsResponse StationBoardWithDetailsResponseType

type GetArrDepBoardWithDetailsResponse StationBoardWithDetailsResponseType

type GetNextDeparturesResponse DeparturesBoardResponseType

type GetNextDeparturesWithDetailsResponse DeparturesBoardWithDetailsResponseType

type GetFastestDeparturesResponse DeparturesBoardResponseType

type GetFastestDeparturesWithDetailsResponse DeparturesBoardWithDetailsResponseType

type GetBoardRequestParams struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetDepartureBoardRequest"`

	// The maximum number of services that are required to be returned. This will be limited to a maximum value by the server, which may change according to system load or other factors. Only the minimum required number of services should be requested. For example, if only 10 services are displayed in a user interface, then this parameter should be set to 10.
	NumRows uint16 `xml:"numRows,omitempty"`

	// The CRS code for the station departure board that is required.
	Crs *CRSType `xml:"crs,omitempty"`

	// An optional CRS code that will filter the returned departure board. For example, if crs is set to "MAN", filterCRS is set to "EUS" and filterType is set to "to" then the departure board will return a list of services that depart Manchester Piccadilly and call at London Euston.
	FilterCrs *CRSType `xml:"filterCrs,omitempty"`

	// The type of filter query that is required, either "from" or "to". This parameter is ignored unless filterCrs is also present.
	FilterType *FilterType `xml:"filterType,omitempty"`

	// A time offset that may be applied to the current time to give the base time for the departure board. The value could be negative if the client has suitable permission configured, otherwise the minimun value will be 0. If the client is not configured with suitable permission then upper bound will be 119.
	TimeOffset int32 `xml:"timeOffset,omitempty"`

	// The number of minutes added to the request start time to give the end time. The parameter default value is 120 minutes, if the supplied value is greater than 120 or not supplied. If the supplied pararmeter vaule is less than 0 then an error will return.
	TimeWindow int32 `xml:"timeWindow,omitempty"`
}

type GetServiceDetailsRequestParams struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetServiceDetailsRequest"`

	// The service ID obtained from a departure board response for which full details are required. Note that service details are only available for a short time after a service has arrived/departed from the location in the departure board that the ID was obtained from.
	ServiceID *ServiceIDType `xml:"serviceID,omitempty"`
}

type GetDeparturesRequestParams struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetNextDeparturesRequest"`

	// The CRS code for the station departure board that is required.
	Crs *CRSType `xml:"crs,omitempty"`

	FilterList struct {
		Crs []*CRSType `xml:"crs,omitempty"`
	} `xml:"filterList,omitempty"`

	// A time offset that may be applied to the current time to give the base time for the departure board. The value could be negative if the client has suitable permission configured, otherwise the minimun value will be 0. If the client is not configured with suitable permission then upper bound will be 119.
	TimeOffset int32 `xml:"timeOffset,omitempty"`

	// The number of minutes added to the request start time to give the end time. The parameter default value is 120 minutes, if the supplied value is greater than 120 or not supplied. If the supplied pararmeter vaule is less than 0 then an error will return.
	TimeWindow int32 `xml:"timeWindow,omitempty"`
}

type StationBoardResponseType struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetDepartureBoardResponse"`

	GetStationBoardResult *StationBoard `xml:"GetStationBoardResult,omitempty"`
}

type StationBoardWithDetailsResponseType struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetDepBoardWithDetailsResponse"`

	GetStationBoardResult *StationBoardWithDetails `xml:"GetStationBoardResult,omitempty"`
}

type ServiceDetailsResponseType struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetServiceDetailsResponse"`

	GetServiceDetailsResult *ServiceDetails `xml:"GetServiceDetailsResult,omitempty"`
}

type DeparturesBoardResponseType struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetNextDeparturesResponse"`

	DeparturesBoard *DeparturesBoard `xml:"DeparturesBoard,omitempty"`
}

type DeparturesBoardWithDetailsResponseType struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2017-10-01/ldb/ GetNextDeparturesWithDetailsResponse"`

	DeparturesBoard *DeparturesBoardWithDetails `xml:"DeparturesBoard,omitempty"`
}

// CRS code used to represent a Station location

type CRSType string

// The display name of a Station location

type LocationNameType string

// The display name of a Train Operating Company

type TOCName string

// A Train Operating Company code

type TOCCode string

// A Platform number

type PlatformType string

// Type used to specify which type of service filter to use. This can either be services at a location that have come "from" another location, or services that are going "to" another location

type FilterType string

const (
	FilterTypeTo FilterType = "to"

	FilterTypeFrom FilterType = "from"
)

// Specifies whether a service is a train, a bus or a ferry

type ServiceType string

const (
	ServiceTypeTrain ServiceType = "train"

	ServiceTypeBus ServiceType = "bus"

	ServiceTypeFerry ServiceType = "ferry"
)

// TIPLOC code used to represent an arbitrary schedule location

type TiplocType string

// A TSDB service Unique Identifier

type UIDType string

// An RTTI service identifier

type RIDType string

// A TSDB Train Identifier (headcode)

type TrainIDType string

// Represents a time displayed in a departure board. This will often be a true time in the format HH:MM (possibly with appended characters, such as "*"), but may also be a string, such as "No report" or "cancelled"

// Defines the length of a train

type TrainLength uint16

// A Retail Service Identifier

type RSIDType string

// A value representing the loading of a train coach as a percentage (0-100%).

type LoadingValue uint32

// A Coach number/identifier in a train formation. E.g. "A" or "12".

type CoachNumberType string

// An indication of the class of a coach in a train formation. E.g. "First", "Standard" or "Mixed".

type CoachClassType string

// An indication of the availability of a toilet in a coach in a train formation. E.g. "Unknown", "None" , "Standard" or "Accessible". Note that other values may be supplied in the future without a schema change.

type ToiletType string

// The service status of a toilet in coach formation data.

type ToiletStatus string

const (
	ToiletStatusUnknown ToiletStatus = "Unknown"

	ToiletStatusInService ToiletStatus = "InService"

	ToiletStatusNotInService ToiletStatus = "NotInService"
)

type ToiletAvailabilityType struct {
	Value *ToiletType

	// The service status of this toilet. E.g. "Unknown", "InService" or "NotInService".

	Status *ToiletStatus `xml:"status,attr,omitempty"`
}

// Represents a time displayed in a departure board. This will often be a true time in the format HH:MM (possibly with appended characters, such as "*"), but may also be a string, such as "No report" or "cancelled"

type TimeType string

// Represents an individual service in a departure board and can be used to return details of that service

type ServiceIDType string

// A string to show the Adhoc Alert Text for the locaiton.

type AdhocAlertTextType string

type NRCCMessage struct {
	Value string
}

type ArrayOfNRCCMessages struct {
	Message []*NRCCMessage `xml:"message,omitempty" json:"message,omitempty"`
}

type ArrayOfArrayOfCallingPoints struct {
	CallingPointList []*ArrayOfCallingPoints `xml:"callingPointList,omitempty" json:"callingPointList,omitempty"`
}

type ArrayOfAdhocAlert struct {
	AdhocAlertText []*AdhocAlertTextType `xml:"adhocAlertText,omitempty" json:"adhocAlertText,omitempty"`
}

type StationBoard struct {
	*BaseStationBoard

	// A list of train services for this departure board.
	TrainServices *ArrayOfServiceItems `xml:"trainServices,omitempty" json:"trainServices,omitempty"`

	// A list of scheduled or replacement rail bus services for this departure board.
	BusServices *ArrayOfServiceItems `xml:"busServices,omitempty" json:"busServices,omitempty"`

	// A list of ferry services for this departure board.
	FerryServices *ArrayOfServiceItems `xml:"ferryServices,omitempty" json:"ferryServices,omitempty"`
}

type ServiceDetails struct {

	// A timestamp of the time these service details were generated.
	GeneratedAt time.Time `xml:"generatedAt,omitempty" json:"generatedAt,omitempty"`

	// The Retail Service ID of the service, if known.
	Rsid *RSIDType `xml:"rsid,omitempty" json:"rsid,omitempty"`

	// The type of service (train, bus, ferry) that these details represent. Note that real-time information (e.g. eta, etd, ata, atd, isCancelled, etc.) is only available and present for train services.
	ServiceType *ServiceType `xml:"serviceType,omitempty" json:"serviceType,omitempty"`

	// The display name of the departure board location that these service details were accessed from.
	LocationName *LocationNameType `xml:"locationName,omitempty" json:"locationName,omitempty"`

	// The CRS code of the departure board location that these service details were accessed from.
	Crs *CRSType `xml:"crs,omitempty" json:"crs,omitempty"`

	// The display name of the Train Operating Company that operates this service.
	Operator *TOCName `xml:"operator,omitempty" json:"operator,omitempty"`

	// The code of the Train Operating Company that operates this service.
	OperatorCode *TOCCode `xml:"operatorCode,omitempty" json:"operatorCode,omitempty"`

	// Indicates that the service is cancelled at this location.
	IsCancelled bool `xml:"isCancelled,omitempty" json:"isCancelled,omitempty"`

	// A cancellation reason for this service.
	CancelReason string `xml:"cancelReason,omitempty" json:"cancelReason,omitempty"`

	// A delay reason for this service.
	DelayReason string `xml:"delayReason,omitempty" json:"delayReason,omitempty"`

	// If an expected movement report has been missed, this will contain a message describing the missed movement.
	OverdueMessage string `xml:"overdueMessage,omitempty" json:"overdueMessage,omitempty"`

	// The train length (number of units) at this location. If not supplied, or zero, the length is unknown.
	Length *TrainLength `xml:"length,omitempty" json:"length,omitempty"`

	// True if the service detaches units from the front at this location.
	DetachFront bool `xml:"detachFront,omitempty" json:"detachFront,omitempty"`

	// True if the service is operating in the reverse of its normal formation.
	IsReverseFormation bool `xml:"isReverseFormation,omitempty" json:"isReverseFormation,omitempty"`

	// The platform number that the service is expected to use at this location, if known and available.
	Platform *PlatformType `xml:"platform,omitempty" json:"platform,omitempty"`

	// The scheduled time of arrival of this service at this location. If no sta is present then this is the origin of this service or it does not set down passengers at this location.
	Sta *TimeType `xml:"sta,omitempty" json:"sta,omitempty"`

	// The estimated time of arrival. Will only be present if sta is also present and ata is not present.
	Eta *TimeType `xml:"eta,omitempty" json:"eta,omitempty"`

	// The actual time of arrival. Will only be present if sta is also present and eta is not present.
	Ata *TimeType `xml:"ata,omitempty" json:"ata,omitempty"`

	// The scheduled time of departure of this service at this location. If no std is present then this is the destination of this service or it does not pick up passengers at this location.
	Std *TimeType `xml:"std,omitempty" json:"std,omitempty"`

	// The estimated time of departure. Will only be present if std is also present and atd is not present.
	Etd *TimeType `xml:"etd,omitempty" json:"etd,omitempty"`

	// The actual time of departure. Will only be present if std is also present and etd is not present.
	Atd *TimeType `xml:"atd,omitempty" json:"atd,omitempty"`

	// A list of active Adhoc Alert texts  for to this location.
	AdhocAlerts *ArrayOfAdhocAlert `xml:"adhocAlerts,omitempty" json:"adhocAlerts,omitempty"`

	// A list of lists of the previous calling points in the journey. A separate calling point list will be present for each origin of the service, relative to the current location. Refer to the documentation for how these lists should be handled.
	PreviousCallingPoints *ArrayOfArrayOfCallingPoints `xml:"previousCallingPoints,omitempty" json:"previousCallingPoints,omitempty"`

	// A list of lists of the subsequent calling points in the journey. A separate calling point list will be present for each destination of the service, relative to the current location. Refer to the documentation for how these lists should be handled.
	SubsequentCallingPoints *ArrayOfArrayOfCallingPoints `xml:"subsequentCallingPoints,omitempty" json:"subsequentCallingPoints,omitempty"`
}

type ArrayOfServiceLocations struct {
	Location []*ServiceLocation `xml:"location,omitempty" json:"location,omitempty"`
}

type BaseStationBoard struct {

	// A timestamp of the time this station departure board was generated.
	GeneratedAt time.Time `xml:"generatedAt,omitempty" json:"generatedAt,omitempty"`

	// The display name of the location that this departure board is for.
	LocationName *LocationNameType `xml:"locationName,omitempty" json:"locationName,omitempty"`

	// The CRS code of the location that this departure board is for.
	Crs *CRSType `xml:"crs,omitempty" json:"crs,omitempty"`

	// If a filter was specified in the request, the display name of the location that was specifed as the filter.
	FilterLocationName *LocationNameType `xml:"filterLocationName,omitempty" json:"filterLocationName,omitempty"`

	// If a filter was specified in the request, the CRS code of the filter location.
	Filtercrs *CRSType `xml:"filtercrs,omitempty" json:"filtercrs,omitempty"`

	// If a filter was specified in the request, the type of filter that was requested.
	FilterType *FilterType `xml:"filterType,omitempty" json:"filterType,omitempty"`

	// A list of messages that apply to this departure board.
	NrccMessages *ArrayOfNRCCMessages `xml:"nrccMessages,omitempty" json:"nrccMessages,omitempty"`

	// A flag to indicate whether platform information is available for this departure board. If this flag is false then platforms will not be returned in the service lists and a user interface should not display a platform "heading".
	PlatformAvailable bool `xml:"platformAvailable,omitempty" json:"platformAvailable,omitempty"`

	// If this flag is present with the value of "true" then service data will be unavailable and the service lists will not be returned. This may happen for example if access to a station has been closed to the public at short notice, even though the scheduled services are still running.
	AreServicesAvailable bool `xml:"areServicesAvailable,omitempty" json:"areServicesAvailable,omitempty"`
}

type BaseServiceItem struct {

	// The scheduled time of arrival of this service. If no sta is present then this is the origin of this service or it does not set down passengers at this location.
	Sta *TimeType `xml:"sta,omitempty" json:"sta,omitempty"`

	// The estimated (or actual) time of arrival. Will only be present if sta is also present.
	Eta *TimeType `xml:"eta,omitempty" json:"eta,omitempty"`

	// The scheduled time of departure of this service. If no std is present then this is the destination of this service or it does not pick up passengers at this location.
	Std *TimeType `xml:"std,omitempty" json:"std,omitempty"`

	// The estimated (or actual) time of departure. Will only be present if std is also present.
	Etd *TimeType `xml:"etd,omitempty" json:"etd,omitempty"`

	// The platform number (if known and available).
	Platform *PlatformType `xml:"platform,omitempty" json:"platform,omitempty"`

	// The Train Operating Company of this service.
	Operator *TOCName `xml:"operator,omitempty" json:"operator,omitempty"`

	// The Train Operating Company code of this service.
	OperatorCode *TOCCode `xml:"operatorCode,omitempty" json:"operatorCode,omitempty"`

	// A flag to indicate if this service is running as part of a circular route and will call at this location again later in its journey.
	IsCircularRoute bool `xml:"isCircularRoute,omitempty" json:"isCircularRoute,omitempty"`

	// A flag to indicate that this service is cancelled at this location.
	IsCancelled bool `xml:"isCancelled,omitempty" json:"isCancelled,omitempty"`

	// A flag to indicate that this service is no longer stopping at the requested from/to filter location.
	FilterLocationCancelled bool `xml:"filterLocationCancelled,omitempty" json:"filterLocationCancelled,omitempty"`

	// The type of service (train, bus, ferry) that this item represents. Note that real-time information (e.g. eta, etd, ata, atd, etc.) is only available and present for train services.
	ServiceType *ServiceType `xml:"serviceType,omitempty" json:"serviceType,omitempty"`

	// The train length (number of units) at this location. If not supplied, or zero, the length is unknown.
	Length *TrainLength `xml:"length,omitempty" json:"length,omitempty"`

	// True if the service detaches units from the front at this location.
	DetachFront bool `xml:"detachFront,omitempty" json:"detachFront,omitempty"`

	// True if the service is operating in the reverse of its normal formation.
	IsReverseFormation bool `xml:"isReverseFormation,omitempty" json:"isReverseFormation,omitempty"`

	// A cancellation reason for this service.
	CancelReason string `xml:"cancelReason,omitempty" json:"cancelReason,omitempty"`

	// A delay reason for this service.
	DelayReason string `xml:"delayReason,omitempty" json:"delayReason,omitempty"`

	// A unique ID for this service at this location that can be used to obtain full details of the service.
	ServiceID *ServiceIDType `xml:"serviceID,omitempty" json:"serviceID,omitempty"`

	// A list of Adhoc Alers related to this locationa for this service.
	AdhocAlerts *ArrayOfAdhocAlert `xml:"adhocAlerts,omitempty" json:"adhocAlerts,omitempty"`

	// The Retail Service ID of the service, if known.
	Rsid *RSIDType `xml:"rsid,omitempty" json:"rsid,omitempty"`
}

type ServiceItem struct {
	*BaseServiceItem

	// A list of origins of this service. Note that a service may have more than one origin.
	Origin *ArrayOfServiceLocations `xml:"origin,omitempty" json:"origin,omitempty"`

	// A list of destinations of this service. Note that a service may have more than one destination.
	Destination *ArrayOfServiceLocations `xml:"destination,omitempty" json:"destination,omitempty"`

	// A list of the current origins for this service. Note that a service may have more than one current origin.
	CurrentOrigins *ArrayOfServiceLocations `xml:"currentOrigins,omitempty" json:"currentOrigins,omitempty"`

	// A list of the current destinations for this service. Note that a service may have more than one current destination.
	CurrentDestinations *ArrayOfServiceLocations `xml:"currentDestinations,omitempty" json:"currentDestinations,omitempty"`

	// The formation data of the train at this location (if known).
	Formation *FormationData `xml:"formation,omitempty" json:"formation,omitempty"`
}

type DepartureItemWithCallingPoints struct {

	// The details of the next/fastest service.
	Service *ServiceItemWithCallingPoints `xml:"service,omitempty" json:"service,omitempty"`

	// The CRS code from the requested filterList for which this service is the next/fastest departure.

	Crs *CRSType `xml:"crs,attr,omitempty" json:"crs,attr,omitempty"`
}

type ServiceLocation struct {

	// The display name of this origin or destination location.
	LocationName *LocationNameType `xml:"locationName,omitempty" json:"locationName,omitempty"`

	// The CRS code of this location.
	Crs *CRSType `xml:"crs,omitempty" json:"crs,omitempty"`

	// A text string that disambiguates services that may have more than one possible route to the destination. The format will typically be as in this example: "via Manchester Piccadilly & Wilmslow"
	Via string `xml:"via,omitempty" json:"via,omitempty"`

	// A text string containing the service type (Bus/Ferry/Train) which will be used to get to this future destination if it differs from the current service type.
	FutureChangeTo string `xml:"futureChangeTo,omitempty" json:"futureChangeTo,omitempty"`

	// This origin or destination can no longer be reached because the association has been cancelled.
	AssocIsCancelled bool `xml:"assocIsCancelled,omitempty" json:"assocIsCancelled,omitempty"`
}

type CallingPoint struct {

	// The display name of this location.
	LocationName *LocationNameType `xml:"locationName,omitempty" json:"locationName,omitempty"`

	// The CRS code of this location.
	Crs *CRSType `xml:"crs,omitempty" json:"crs,omitempty"`

	// The scheduled time of the service at this location. The time will be either an arrival or departure time, depending on whether it is in the subsequent or previous calling point list.
	St *TimeType `xml:"st,omitempty" json:"st,omitempty"`

	// The estimated time of the service at this location. The time will be either an arrival or departure time, depending on whether it is in the subsequent or previous calling point list. Will only be present if an actual time (at) is not present.
	Et *TimeType `xml:"et,omitempty" json:"et,omitempty"`

	// The actual time of the service at this location. The time will be either an arrival or departure time, depending on whether it is in the subsequent or previous calling point list. Will only be present if an estimated time (et) is not present.
	At *TimeType `xml:"at,omitempty" json:"at,omitempty"`

	// A flag to indicate that this service is cancelled at this location.
	IsCancelled bool `xml:"isCancelled,omitempty" json:"isCancelled,omitempty"`

	// The train length (number of units) at this location. If not supplied, or zero, the length is unknown.
	Length *TrainLength `xml:"length,omitempty" json:"length,omitempty"`

	// True if the service detaches units from the front at this location.
	DetachFront bool `xml:"detachFront,omitempty" json:"detachFront,omitempty"`

	// A list of active Adhoc Alert texts  for to this location.
	AdhocAlerts *ArrayOfAdhocAlert `xml:"adhocAlerts,omitempty" json:"adhocAlerts,omitempty"`
}

type StationBoardWithDetails struct {
	*BaseStationBoard

	// A list of train services for this departure board.
	TrainServices *ArrayOfServiceItemsWithCallingPoints `xml:"trainServices,omitempty" json:"trainServices,omitempty"`

	// A list of scheduled or replacement rail bus services for this departure board.
	BusServices *ArrayOfServiceItemsWithCallingPoints `xml:"busServices,omitempty" json:"busServices,omitempty"`

	// A list of ferry services for this departure board.
	FerryServices *ArrayOfServiceItemsWithCallingPoints `xml:"ferryServices,omitempty" json:"ferryServices,omitempty"`
}

type ServiceItemWithCallingPoints struct {
	*ServiceItem

	// A list of lists of the previous calling points in the journey. A separate calling point list will be present for each origin of the service, relative to the current location. Refer to the documentation for how these lists should be handled.
	PreviousCallingPoints *ArrayOfArrayOfCallingPoints `xml:"previousCallingPoints,omitempty" json:"previousCallingPoints,omitempty"`

	// A list of lists of the subsequent calling points in the journey. A separate calling point list will be present for each destination of the service, relative to the current location. Refer to the documentation for how these lists should be handled.
	SubsequentCallingPoints *ArrayOfArrayOfCallingPoints `xml:"subsequentCallingPoints,omitempty" json:"subsequentCallingPoints,omitempty"`
}

type DeparturesBoard struct {
	*BaseStationBoard

	// A list of next/fastest services for this departures board.
	Departures *ArrayOfDepartureItems `xml:"departures,omitempty" json:"departures,omitempty"`
}

type DeparturesBoardWithDetails struct {
	*BaseStationBoard

	// A list of next/fastest services for this departures board.
	Departures *ArrayOfDepartureItemsWithCallingPoints `xml:"departures,omitempty" json:"departures,omitempty"`
}

type BaseServiceDetails struct {

	// A timestamp of the time these service details were generated.
	GeneratedAt time.Time `xml:"generatedAt,omitempty" json:"generatedAt,omitempty"`

	// The type of service (train, bus, ferry) that these details represent. Note that real-time information (e.g. eta, etd, ata, atd, isCancelled, etc.) is only available and present for train services.
	ServiceType *ServiceType `xml:"serviceType,omitempty" json:"serviceType,omitempty"`

	// The display name of the departure board location that these service details were accessed from.
	LocationName *LocationNameType `xml:"locationName,omitempty" json:"locationName,omitempty"`

	// The CRS code of the departure board location that these service details were accessed from.
	Crs *CRSType `xml:"crs,omitempty" json:"crs,omitempty"`

	// The display name of the Train Operating Company that operates this service.
	Operator *TOCName `xml:"operator,omitempty" json:"operator,omitempty"`

	// The code of the Train Operating Company that operates this service.
	OperatorCode *TOCCode `xml:"operatorCode,omitempty" json:"operatorCode,omitempty"`

	// The Retail Service ID of the service, if known.
	Rsid *RSIDType `xml:"rsid,omitempty" json:"rsid,omitempty"`

	// Indicates that the service is cancelled at this location.
	IsCancelled bool `xml:"isCancelled,omitempty" json:"isCancelled,omitempty"`

	// A cancellation reason for this service.
	CancelReason string `xml:"cancelReason,omitempty" json:"cancelReason,omitempty"`

	// A delay reason for this service.
	DelayReason string `xml:"delayReason,omitempty" json:"delayReason,omitempty"`

	// If an expected movement report has been missed, this will contain a message describing the missed movement.
	OverdueMessage string `xml:"overdueMessage,omitempty" json:"overdueMessage,omitempty"`

	// The train length (number of units) at this location. If not supplied, or zero, the length is unknown.
	Length *TrainLength `xml:"length,omitempty" json:"length,omitempty"`

	// True if the service detaches units from the front at this location.
	DetachFront bool `xml:"detachFront,omitempty" json:"detachFront,omitempty"`

	// True if the service is operating in the reverse of its normal formation.
	IsReverseFormation bool `xml:"isReverseFormation,omitempty" json:"isReverseFormation,omitempty"`

	// The platform number that the service is expected to use at this location, if known and available.
	Platform *PlatformType `xml:"platform,omitempty" json:"platform,omitempty"`

	// The scheduled time of arrival of this service at this location. If no sta is present then this is the origin of this service or it does not set down passengers at this location.
	Sta *TimeType `xml:"sta,omitempty" json:"sta,omitempty"`

	// The estimated time of arrival. Will only be present if sta is also present and ata is not present.
	Eta *TimeType `xml:"eta,omitempty" json:"eta,omitempty"`

	// The actual time of arrival. Will only be present if sta is also present and eta is not present.
	Ata *TimeType `xml:"ata,omitempty" json:"ata,omitempty"`

	// The scheduled time of departure of this service at this location. If no std is present then this is the destination of this service or it does not pick up passengers at this location.
	Std *TimeType `xml:"std,omitempty" json:"std,omitempty"`

	// The estimated time of departure. Will only be present if std is also present and atd is not present.
	Etd *TimeType `xml:"etd,omitempty" json:"etd,omitempty"`

	// The actual time of departure. Will only be present if std is also present and etd is not present.
	Atd *TimeType `xml:"atd,omitempty" json:"atd,omitempty"`
}

type DepartureItem struct {

	// The details of the next/fastest service.
	Service *ServiceItem `xml:"service,omitempty" json:"service,omitempty"`

	// The CRS code from the requested filterList for which this service is the next/fastest departure.

	Crs *CRSType `xml:"crs,attr,omitempty" json:"crs,attr,omitempty"`
}

type FormationData struct {

	// Average Loading of the train as a whole at this Calling Point. This is a fixed value that is based on long-term averages and does not vary according to real-time actual loading.
	AvgLoading *LoadingValue `xml:"avgLoading,omitempty" json:"avgLoading,omitempty"`

	// A list of coaches that comprise the train formation at this calling point. Will be absent if the formation is unknown.
	Coaches *ArrayOfCoaches `xml:"coaches,omitempty" json:"coaches,omitempty"`
}

type CoachData struct {

	// The class of a coach, where known, e.g. First, Standard, Mixed. Other classes may be introduced in future without a schema change.
	CoachClass *CoachClassType `xml:"coachClass,omitempty" json:"coachClass,omitempty"`

	// The availability of a toilet in this coach. E.g. "Unknown", "None" , "Standard" or "Accessible". Note that other values may be supplied in the future without a schema change. If no toilet availability is supplied then it should be assumed to be "Unknown".
	Toilet *ToiletAvailabilityType `xml:"toilet,omitempty" json:"toilet,omitempty"`

	// The currently estimated passenger loading value for this coach, where known.
	Loading *LoadingValue `xml:"loading,omitempty" json:"loading,omitempty"`

	// The number/identifier for this coach, e.g. "A" or "12".

	Number *CoachNumberType `xml:"number,attr,omitempty" json:"number,attr,omitempty"`
}

type ArrayOfServiceItems struct {
	Service []*ServiceItem `xml:"service,omitempty" json:"service,omitempty"`
}

type ArrayOfServiceItemsWithCallingPoints struct {
	Service []*ServiceItemWithCallingPoints `xml:"service,omitempty" json:"service,omitempty"`
}

type ArrayOfDepartureItems struct {
	Destination []*DepartureItem `xml:"destination,omitempty" json:"destination,omitempty"`
}

type ArrayOfDepartureItemsWithCallingPoints struct {
	Destination []*DepartureItemWithCallingPoints `xml:"destination,omitempty" json:"destination,omitempty"`
}

type ArrayOfCoaches struct {
	Coach []*CoachData `xml:"coach,omitempty" json:"coach,omitempty"`
}

type ArrayOfCallingPoints struct {
	CallingPoint []*CallingPoint `xml:"callingPoint,omitempty" json:"callingPoint,omitempty"`

	// The type of service (train, bus, ferry) of this list of calling points.

	ServiceType *ServiceType `xml:"serviceType,attr,omitempty" json:"serviceType,attr,omitempty"`

	// A boolean to indicate that passenger required to change the service or not.

	ServiceChangeRequired bool `xml:"serviceChangeRequired,attr,omitempty" json:"serviceChangeRequired,attr,omitempty"`

	// A boolean to indicate that this route from the origin or to the destination can no longer be reached because the association has been cancelled.

	AssocIsCancelled bool `xml:"assocIsCancelled,attr,omitempty" json:"assocIsCancelled,attr,omitempty"`
}

type LDBServiceSoap interface {
	GetDepartureBoard(request *GetBoardRequestParams) (*StationBoardResponseType, error)

	GetDepartureBoardContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardResponseType, error)

	GetArrivalBoard(request *GetBoardRequestParams) (*StationBoardResponseType, error)

	GetArrivalBoardContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardResponseType, error)

	GetArrivalDepartureBoard(request *GetBoardRequestParams) (*StationBoardResponseType, error)

	GetArrivalDepartureBoardContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardResponseType, error)

	GetServiceDetails(request *GetServiceDetailsRequestParams) (*ServiceDetailsResponseType, error)

	GetServiceDetailsContext(ctx context.Context, request *GetServiceDetailsRequestParams) (*ServiceDetailsResponseType, error)

	GetDepBoardWithDetails(request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error)

	GetDepBoardWithDetailsContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error)

	GetArrBoardWithDetails(request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error)

	GetArrBoardWithDetailsContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error)

	GetArrDepBoardWithDetails(request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error)

	GetArrDepBoardWithDetailsContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error)

	GetNextDepartures(request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error)

	GetNextDeparturesContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error)

	GetNextDeparturesWithDetails(request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error)

	GetNextDeparturesWithDetailsContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error)

	GetFastestDepartures(request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error)

	GetFastestDeparturesContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error)

	GetFastestDeparturesWithDetails(request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error)

	GetFastestDeparturesWithDetailsContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error)
}

type lDBServiceSoap struct {
	client *soap.Client
}

func NewLDBServiceSoap(client *soap.Client) LDBServiceSoap {
	return &lDBServiceSoap{
		client: client,
	}
}

func (service *lDBServiceSoap) GetDepartureBoardContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardResponseType, error) {
	response := new(StationBoardResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2012-01-13/ldb/GetDepartureBoard", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetDepartureBoard(request *GetBoardRequestParams) (*StationBoardResponseType, error) {
	return service.GetDepartureBoardContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetArrivalBoardContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardResponseType, error) {
	response := new(StationBoardResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2012-01-13/ldb/GetArrivalBoard", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetArrivalBoard(request *GetBoardRequestParams) (*StationBoardResponseType, error) {
	return service.GetArrivalBoardContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetArrivalDepartureBoardContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardResponseType, error) {
	response := new(StationBoardResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2012-01-13/ldb/GetArrivalDepartureBoard", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetArrivalDepartureBoard(request *GetBoardRequestParams) (*StationBoardResponseType, error) {
	return service.GetArrivalDepartureBoardContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetServiceDetailsContext(ctx context.Context, request *GetServiceDetailsRequestParams) (*ServiceDetailsResponseType, error) {
	response := new(ServiceDetailsResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2012-01-13/ldb/GetServiceDetails", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetServiceDetails(request *GetServiceDetailsRequestParams) (*ServiceDetailsResponseType, error) {
	return service.GetServiceDetailsContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetDepBoardWithDetailsContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error) {
	response := new(StationBoardWithDetailsResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2015-05-14/ldb/GetDepBoardWithDetails", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetDepBoardWithDetails(request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error) {
	return service.GetDepBoardWithDetailsContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetArrBoardWithDetailsContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error) {
	response := new(StationBoardWithDetailsResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2015-05-14/ldb/GetArrBoardWithDetails", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetArrBoardWithDetails(request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error) {
	return service.GetArrBoardWithDetailsContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetArrDepBoardWithDetailsContext(ctx context.Context, request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error) {
	response := new(StationBoardWithDetailsResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2015-05-14/ldb/GetArrDepBoardWithDetails", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetArrDepBoardWithDetails(request *GetBoardRequestParams) (*StationBoardWithDetailsResponseType, error) {
	return service.GetArrDepBoardWithDetailsContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetNextDeparturesContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error) {
	response := new(DeparturesBoardResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2015-05-14/ldb/GetNextDepartures", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetNextDepartures(request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error) {
	return service.GetNextDeparturesContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetNextDeparturesWithDetailsContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error) {
	response := new(DeparturesBoardWithDetailsResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2015-05-14/ldb/GetNextDeparturesWithDetails", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetNextDeparturesWithDetails(request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error) {
	return service.GetNextDeparturesWithDetailsContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetFastestDeparturesContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error) {
	response := new(DeparturesBoardResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2015-05-14/ldb/GetFastestDepartures", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetFastestDepartures(request *GetDeparturesRequestParams) (*DeparturesBoardResponseType, error) {
	return service.GetFastestDeparturesContext(
		context.Background(),
		request,
	)
}

func (service *lDBServiceSoap) GetFastestDeparturesWithDetailsContext(ctx context.Context, request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error) {
	response := new(DeparturesBoardWithDetailsResponseType)
	err := service.client.CallContext(ctx, "http://thalesgroup.com/RTTI/2015-05-14/ldb/GetFastestDeparturesWithDetails", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (service *lDBServiceSoap) GetFastestDeparturesWithDetails(request *GetDeparturesRequestParams) (*DeparturesBoardWithDetailsResponseType, error) {
	return service.GetFastestDeparturesWithDetailsContext(
		context.Background(),
		request,
	)
}
