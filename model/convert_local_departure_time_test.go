package model

import (
	"testing"
	"time"
)

func TestConvertDepartureTime(t *testing.T) {
	localLocation, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	t.Run("future departure on current day", func(t *testing.T) {
		localTime := "13:20"

		then := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, localLocation)

		got, err := ConvertDepartureTime(&then, localLocation, localTime)
		if err != nil {
			t.Error(err)
		}

		want := then.Add(time.Hour + time.Minute*20).In(localLocation)

		if !got.Equal(want) {
			t.Errorf("got %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
		}
	})

	t.Run("future departure on following day", func(t *testing.T) {
		localTime := "01:20"

		then := time.Date(now.Year(), now.Month(), now.Day(), 23, 30, 0, 0, localLocation)

		got, err := ConvertDepartureTime(&then, localLocation, localTime)
		if err != nil {
			t.Error(err)
		}

		want := then.Add(time.Hour + time.Minute*50).In(localLocation)

		if !got.Equal(want) {
			t.Errorf("got %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
		}
	})

	t.Run("late departure on current day", func(t *testing.T) {
		localTime := "11:20"

		then := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, localLocation)

		got, err := ConvertDepartureTime(&then, localLocation, localTime)
		if err != nil {
			t.Error(err)
		}

		want := then.Add(time.Minute * -40).In(localLocation)

		if !got.Equal(want) {
			t.Errorf("got %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
		}
	})

	t.Run("late departure on previous day", func(t *testing.T) {
		localTime := "23:40"

		then := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 40, 0, 0, localLocation)

		got, err := ConvertDepartureTime(&then, localLocation, localTime)
		if err != nil {
			t.Error(err)
		}

		want := then.Add(time.Hour * -1).In(localLocation)

		if !got.Equal(want) {
			t.Errorf("got %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
		}
	})
}
