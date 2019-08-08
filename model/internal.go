package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Departure contains a unique identifier for the journey at the location,
// the aimed and expected departure time, the departure location,
// the destination, the bus service number and the operator
type Departure struct {
	RecordedAtTime        string      `json:"recordedAtTime,omitempty"`
	JourneyType           JourneyType `json:"journeyType,omitempty"`
	JourneyRef            string      `json:"journeyRef,omitempty"`
	AimedDepartureTime    string      `json:"aimedDepartureTime,omitempty"`
	ExpectedDepartureTime *string     `json:"expectedDepartureTime,omitempty"`
	DepartureStatus       *string     `json:"departureStatus,omitempty"`
	LocationAtcocode      string      `json:"locationAtcocode,omitempty"`
	Stand                 *string     `json:"stand,omitempty"`
	DestinationAtcocode   string      `json:"destinationAtcocode,omitempty"`
	Destination           string      `json:"destination,omitempty"`
	ServiceNumber         string      `json:"serviceNumber,omitempty"`
	OperatorCode          string      `json:"operatorCode,omitempty"`
}

type DepartureInterface interface {
	DepartureTime() (departureTime time.Time, isRealTime bool, err error)
	IsExpired(now time.Time) bool
}

// Internal contains an array of Departure items
type Internal struct {
	Departures []Departure `json:"departures"`
}

type ByDepartureTime []Departure

type ByServiceNumber []Departure

func (a ByDepartureTime) Len() int {
	return len(a)
}

func (a ByDepartureTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByDepartureTime) Less(i, j int) bool {
	iDepartureTime, _, err := a[i].DepartureTime()
	if err != nil {
		panic(err)
	}
	jDepartureTime, _, err := a[j].DepartureTime()
	if err != nil {
		panic(err)
	}

	if iDepartureTime.Equal(jDepartureTime) {
		result := lessByServiceNumber(a[i], a[j])
		if result == nil {
			return a[i].JourneyRef < a[j].JourneyRef
		}

		return *result
	}
	return iDepartureTime.Before(jDepartureTime)
}

func (a ByServiceNumber) Len() int {
	return len(a)
}

func (a ByServiceNumber) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByServiceNumber) Less(i, j int) bool {
	result := lessByServiceNumber(a[i], a[j])
	if result == nil {
		iDepartureTime, _, err := a[i].DepartureTime()
		if err != nil {
			panic(err)
		}

		jDepartureTime, _, err := a[j].DepartureTime()
		if err != nil {
			panic(err)
		}

		if iDepartureTime.Equal(jDepartureTime) {
			return a[i].JourneyRef < a[j].JourneyRef
		}

		return iDepartureTime.Before(jDepartureTime)
	}

	return *result
}

func lessByServiceNumber(a, b Departure) *bool {
	var result bool
	if a.ServiceNumber == b.ServiceNumber {
		return nil
	}
	iPrefix, iDigits, iSuffix := a.GetServiceNumberParts()
	jPrefix, jDigits, jSuffix := b.GetServiceNumberParts()

	if iPrefix == nil && jPrefix == nil || iPrefix != nil && jPrefix != nil && *iPrefix == *jPrefix {
		if iDigits != nil && jDigits != nil {
			if *iDigits == *jDigits {
				if iSuffix != nil && jSuffix != nil {
					result = *iSuffix < *jSuffix
					return &result
				}

				result = iSuffix == nil

				return &result
			}

			result = *iDigits < *jDigits
			return &result
		}

		result = jDigits == nil

		return &result
	}

	if iPrefix == nil {
		result = true
		return &result
	}

	if jPrefix == nil {
		result = false
		return &result
	}

	result = *iPrefix < *jPrefix
	return &result
}

func (d Departure) DepartureTime() (departureTime time.Time, isRealTime bool, err error) {
	if d.ExpectedDepartureTime != nil {
		departureTime, err = time.Parse(time.RFC3339, *d.ExpectedDepartureTime)
		return departureTime, true, err
	}

	departureTime, err = time.Parse(time.RFC3339, d.AimedDepartureTime)
	return departureTime, false, err
}

func (d Departure) IsExpired(now time.Time) bool {
	depTime, _, err := d.DepartureTime()
	if err != nil {
		panic(err)
	}

	if d.JourneyType == Train {
		if *d.DepartureStatus == "Delayed" {
			return false
		}

		re := regexp.MustCompile("^[0-9]{2}:[0-9]{2}$")
		if re.MatchString(*d.DepartureStatus) {
			expectedDepartureTime, err := ConvertDepartureTime(&now, depTime.Location(), *d.DepartureStatus)
			if err != nil {
				panic(err)
			}

			return expectedDepartureTime.Before(now.Truncate(time.Minute))
		}

		return depTime.Before(now.Truncate(time.Minute))
	}

	return depTime.Before(now)
}

func (d Departure) GetServiceNumberParts() (prefix *string, digits *int, suffix *string) {
	re := regexp.MustCompile(`^([A-Z]+)?(([\d]{1,3})?([A-Z])?)?$`)
	serviceNumber := []byte(strings.ToUpper(d.ServiceNumber))
	matches := re.FindAllSubmatch(serviceNumber, -1)
	if matches == nil || len(matches) > 1 {
		err := fmt.Errorf("invalid service number: %s", d.ServiceNumber)
		panic(err)
	}
	if matches[0][1] != nil {
		prefixVal := string(matches[0][1])
		prefix = &prefixVal
	} else {
		prefix = nil
	}
	if matches[0][4] != nil {
		suffixVal := string(matches[0][4])
		suffix = &suffixVal
	} else {
		suffix = nil
	}
	if matches[0][3] != nil {
		digitsVal, err := strconv.Atoi(string(matches[0][3]))
		if err != nil {
			panic(err)
		}
		digits = &digitsVal
	} else {
		digits = nil
	}
	return
}

func (d Departure) GetStand() *string {
	re := regexp.MustCompile(`^180[A-Z0-9][A-Z]{2}(BS|IC|IN)([A-Z0-9]{2})[0-9]$`)
	locationAtcocode := []byte(strings.ToUpper(d.LocationAtcocode))
	matches := re.FindAllSubmatch(locationAtcocode, -1)

	if matches == nil || len(matches) > 1 {
		return nil
	}

	standPart := matches[0][2]
	if string(standPart[0]) == "0" {
		stand := string(standPart[1])
		return &stand
	}
	stand := string(standPart)
	return &stand
}
