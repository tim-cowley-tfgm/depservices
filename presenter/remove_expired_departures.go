package main

import (
	"github.com/TfGMEnterprise/departures-service/model"
	"time"
)

// We trust that the data is sorted by ascending departure time
func (p *Presenter) removeExpiredDepartures(now time.Time, deps *model.Internal) int64 {
	p.Logger.Debug("removeExpiredDepartures")

	if deps == nil {
		return 0
	}

	var i = 0
	for _, dep := range deps.Departures {
		if dep.IsExpired(now) {
			i++
			continue
		}
		break
	}

	p.Logger.Debugf("removed %d expired departure(s)", i)

	deps.Departures = deps.Departures[i:]

	return int64(i)
}
