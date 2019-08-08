package main

import (
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"io/ioutil"
	"testing"
)

func Test_validateTop(t *testing.T) {
	logger := dlog.NewLogger([]dlog.LoggerOption{
		dlog.LoggerSetOutput(ioutil.Discard),
	}...)

	p := Presenter{
		Logger: logger,
	}

	t.Run("should return false for 0", func(t *testing.T) {
		got := p.validateTop(0)
		test_helpers.AssertBoolean(t, got, false)
	})

	t.Run("should return false for a negative value", func(t *testing.T) {
		got := p.validateTop(-1)
		test_helpers.AssertBoolean(t, got, false)
	})

	t.Run("should return true for a positive value", func(t *testing.T) {
		got := p.validateTop(1)
		test_helpers.AssertBoolean(t, got, true)
	})
}
