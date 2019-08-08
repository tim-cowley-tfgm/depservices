package test_helpers

import (
	"encoding/json"
	"encoding/xml"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func AssertBoolean(t *testing.T, got bool, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("got '%t' want '%t'\n", got, want)
	}
}

func AssertJSONEquality(t *testing.T, rr *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var got interface{}
	var want interface{}

	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected body: got %#v, wanted %#v\n", rr.Body.String(), expected)
	}
}

func AssertStatusCode(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if status := rr.Code; status != want {
		t.Errorf("wrong status code: got %v, wanted %v\n", status, want)
	}
}

func AssertString(t *testing.T, got string, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got '%s' want '%s'\n", got, want)
	}
}

func AssertXMLEquality(t *testing.T, rr *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var got interface{}
	var want interface{}

	if err := xml.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	if err := xml.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected body: got %#v, wanted %#v\n", rr.Body.String(), expected)
	}
}

func AdjustTime(now time.Time, d string) time.Time {
	duration, _ := time.ParseDuration(d)
	return now.Add(duration)
}
