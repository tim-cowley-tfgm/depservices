package main

import (
	"github.com/TfGMEnterprise/departures-service/model"
	"strconv"
	"time"
)

func (p *Presenter) transformDepartureTime(now time.Time, dep model.DepartureInterface) (string, error) {
	p.Logger.Debug("transformDepartureTime")
	depTime, isRealTime, err := dep.DepartureTime()

	if err != nil {
		return "", err
	}

	p.Logger.Debugf("departure time is: %s (real-time: %v)", depTime.Format(time.RFC3339), isRealTime)

	if isRealTime {
		wait := int(depTime.Sub(now).Truncate(time.Minute).Minutes())

		if wait == 0 {
			return "Approaching", nil
		}

		mins := "min"

		if wait != 1 {
			mins += "s"
		}

		return strconv.Itoa(wait) + " " + mins, nil
	}

	return depTime.Format("15:04"), nil
}
