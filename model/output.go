package model

// Output contains:
// departures - a collection of DepartureDisplay items
type Output struct {
	JourneyType JourneyType        `json:"journeyType"`
	Departures  []DepartureDisplay `json:"departures"`
}

// DepartureDisplay contains:
// departure time - either as a countdown for real-time data, or HH:MM for scheduled data;
// stand - a one- or two-character string for a bus station stand, or nil if there isn't one;
// service number - the bus service number, including prefix and suffix if applicable; and
// destination - where the bus is going: typically would be the end point of the route, but
//   might also be something like "Horwich circular", "Rochdale via Middleton", etc.
// departure status - a text string used for rail departures to represent the status of
//   the departure; e.g. "On time", "Delayed", "Cancelled", or the estimated departure time
//   ("15:04")
type DepartureDisplay struct {
	DepartureTime   string  `json:"departureTime,omitempty"`
	Stand           *string `json:"stand,omitempty"`
	ServiceNumber   string  `json:"serviceNumber,omitempty"`
	Destination     string  `json:"destination,omitempty"`
	DepartureStatus *string `json:"departureStatus,omitempty"`
}
