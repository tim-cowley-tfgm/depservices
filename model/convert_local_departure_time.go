package model

import (
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ConvertDepartureTime(now *time.Time, localLocation *time.Location, localTime string) (*time.Time, error) {
	if now == nil {
		return nil, errors.New("now must be set")
	}

	re := regexp.MustCompile("^[0-9]{2}:[0-9]{2}$")
	if !re.MatchString(localTime) {
		return nil, errors.New("timeType must be set")
	}

	hm := strings.Split(localTime, ":")
	hours, err := strconv.Atoi(hm[0])
	if err != nil {
		return nil, errors.Wrap(err, "could not parse hours from departure time")
	}

	mins, err := strconv.Atoi(hm[1])
	if err != nil {
		return nil, errors.Wrap(err, "could not parse mins from departure time")
	}

	// National rail provides departures up to two hours into the future.
	// We initially assume that the departure is for the current day
	departingToday := time.Date(now.Year(), now.Month(), now.Day(), hours, mins, 0, 0, localLocation)

	// If subtracting the departure time based on today's date from the
	// current time results in a departure that is more than two hours in the
	// future, it is very likely that the departure is for the following day.

	if now.Sub(departingToday) > 2*time.Hour {
		// We create a new time here rather than adding a duration to prevent
		// issues that may arise with daylight savings time changes.
		departingTomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, hours, mins, 0, 0, localLocation)
		return &departingTomorrow, nil
	}

	// Similarly, if subtracting the current time from the departure time based
	// on today's date results in a duration that is more than two hours into
	// the future, it is very likely that the departure is for the previous day.
	if departingToday.Sub(*now) > 2*time.Hour {
		// Again, look out for daylight savings time changes.
		departingYesterday := time.Date(now.Year(), now.Month(), now.Day()-1, hours, mins, 0, 0, localLocation)
		return &departingYesterday, nil
	}

	return &departingToday, nil
}
