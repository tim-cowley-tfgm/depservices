package main

import (
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"io/ioutil"
	"testing"
	"time"
)

func TestPresenter_RemoveExpiredDepartures(t *testing.T) {
	logger := dlog.NewLogger([]dlog.LoggerOption{
		dlog.LoggerSetOutput(ioutil.Discard),
	}...)

	p := Presenter{
		Logger: logger,
	}

	t.Run("handles a nil value", func(t *testing.T) {
		now := time.Now()

		got := p.removeExpiredDepartures(now, nil)

		if got != 0 {
			t.Errorf("got `%d`, want `%d`", got, 0)
		}
	})

	t.Run("does not remove anything if all the departures are now or in the future", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		deps := model.Internal{}
		deps.Departures = append(deps.Departures, model.Departure{
			AimedDepartureTime: test_helpers.AdjustTime(now, "10s").Format(time.RFC3339),
		})
		deps.Departures = append(deps.Departures, model.Departure{
			AimedDepartureTime: test_helpers.AdjustTime(now, "20s").Format(time.RFC3339),
		})

		got := p.removeExpiredDepartures(now, &deps)

		if len(deps.Departures) != 2 {
			t.Errorf("Internal struct contains %d departures, should be %d", len(deps.Departures), 2)
			return
		}

		if got != 0 {
			t.Errorf("Removed %d departure(s); should remove %d", got, 0)
		}
	})

	t.Run("removes expired departures", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		deps := model.Internal{}
		deps.Departures = append(deps.Departures, model.Departure{
			AimedDepartureTime: test_helpers.AdjustTime(now, "-10s").Format(time.RFC3339),
		})
		deps.Departures = append(deps.Departures, model.Departure{
			JourneyRef:         "1234",
			AimedDepartureTime: test_helpers.AdjustTime(now, "10s").Format(time.RFC3339),
		})

		got := p.removeExpiredDepartures(now, &deps)

		if len(deps.Departures) != 1 {
			t.Errorf("Internal struct contains %d departures, should be %d", len(deps.Departures), 1)
			return
		}

		if deps.Departures[0].JourneyRef != "1234" {
			t.Error("Wrong departure removed from departures struct")
			return
		}

		if got != 1 {
			t.Errorf("Removed %d departure(s); should remove %d", got, 1)
		}
	})
}
