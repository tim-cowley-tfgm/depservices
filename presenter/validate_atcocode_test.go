package main

import (
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"io/ioutil"
	"testing"
)

func Test_validateAtcocode(t *testing.T) {
	logger := dlog.NewLogger([]dlog.LoggerOption{
		dlog.LoggerSetOutput(ioutil.Discard),
	}...)

	p := Presenter{
		Logger: logger,
	}

	t.Run("valid bus station busStationAtcocode", func(t *testing.T) {
		got := p.validateAtcocode("1800BNIN")
		test_helpers.AssertBoolean(t, got, true)
	})

	t.Run("valid bus station stand busStationAtcocode", func(t *testing.T) {
		got := p.validateAtcocode("1800BNIN0A1")
		test_helpers.AssertBoolean(t, got, true)
	})

	t.Run("valid bus stop busStationAtcocode", func(t *testing.T) {
		got := p.validateAtcocode("1800NE43431")
		test_helpers.AssertBoolean(t, got, true)
	})

	t.Run("invalid bus station busStationAtcocode", func(t *testing.T) {
		got := p.validateAtcocode("foo")
		test_helpers.AssertBoolean(t, got, false)
	})
}
