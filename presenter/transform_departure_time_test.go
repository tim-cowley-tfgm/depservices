package main

import (
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"io/ioutil"
	"testing"
	"time"
)

func Test_TransformDepartureTime(t *testing.T) {
	logger := dlog.NewLogger([]dlog.LoggerOption{
		dlog.LoggerSetOutput(ioutil.Discard),
	}...)

	p := Presenter{
		Logger: logger,
	}

	t.Run("should return the time of day if expected departure time is nil", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		dep := model.Departure{
			AimedDepartureTime: "2019-05-20T12:34:56+01:00",
		}

		got, _ := p.transformDepartureTime(now, dep)

		want := "12:34"

		if got != want {
			t.Errorf("got `%s`, want `%s` for departure time", got, want)
		}
	})

	t.Run("should return `Approaching` if expected departure time is less than one minute away", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		expectedDepartureTime := test_helpers.AdjustTime(now, "59s").Format(time.RFC3339)
		dep := model.Departure{
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		got, _ := p.transformDepartureTime(now, dep)

		want := "Approaching"

		if got != want {
			t.Errorf("got `%s`, want `%s` for departure time", got, want)
		}
	})

	t.Run("should return `1 min` if expected departure time is exactly 1 minute away", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		expectedDepartureTime := test_helpers.AdjustTime(now, "1m").Format(time.RFC3339)
		dep := model.Departure{
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		got, _ := p.transformDepartureTime(now, dep)

		want := "1 min"

		if got != want {
			t.Errorf("got `%s`, want `%s` for departure time", got, want)
		}
	})

	t.Run("should return `1 min` if expected departure time is less than two minutes away", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		expectedDepartureTime := test_helpers.AdjustTime(now, "1m59s").Format(time.RFC3339)
		dep := model.Departure{
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		got, _ := p.transformDepartureTime(now, dep)

		want := "1 min"

		if got != want {
			t.Errorf("got `%s`, want `%s` for departure time", got, want)
		}
	})

	t.Run("should return `2 mins` if expected departure time is exactly 2 minutes away", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		expectedDepartureTime := test_helpers.AdjustTime(now, "2m").Format(time.RFC3339)
		dep := model.Departure{
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		got, _ := p.transformDepartureTime(now, dep)

		want := "2 mins"

		if got != want {
			t.Errorf("got `%s`, want `%s` for departure time", got, want)
		}
	})

	t.Run("should return `2 mins` if expected departure time is less than three minutes away", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		expectedDepartureTime := test_helpers.AdjustTime(now, "2m59s").Format(time.RFC3339)
		dep := model.Departure{
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		got, _ := p.transformDepartureTime(now, dep)

		want := "2 mins"

		if got != want {
			t.Errorf("got `%s`, want `%s` for departure time", got, want)
		}
	})
}
