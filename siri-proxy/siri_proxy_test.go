package main

import (
	"bytes"
	"encoding/xml"
	duration "github.com/ChannelMeter/iso8601duration"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/siri"
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"
)

func cleanUpTickers(t *testing.T, sp *SiriProxy) {
	t.Helper()

	sp.muxHeartbeatSubscriptions.Lock()
	for _, sub := range sp.HeartbeatSubscriptions {
		if sub.Ticker != nil {
			sub.Ticker.Stop()
		}

		if sub.Timer != nil {
			sub.Timer.Stop()
		}
	}
	sp.muxHeartbeatSubscriptions.Unlock()
}

func TestHandler(t *testing.T) {
	httpClientTimeout := 10 * time.Second
	serverURL := "http://localhost"

	t.Run("invalid request method", func(t *testing.T) {

	})

	t.Run("invalid request body", func(t *testing.T) {

	})

	t.Run("request-response - CapabilitiesRequest", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.CapabilitiesRequest == nil {
				t.Error("expected payload to contain a CapabilitiesRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableCapabilitiesResponse.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer ts.Close()

		b, err := ioutil.ReadFile("../test_resources/EstimatedTimetableCapabilitiesRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &ts.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for CapabilitiesRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableCapabilitiesResponse.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for CapabilitiesRequest")
		}
	})

	t.Run("request-response - CheckStatusRequest", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.CheckStatusRequest == nil {
				t.Error("expected payload to contain a CheckStatusRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/CheckStatusResponse.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/CheckStatusRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for CheckStatusRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/CheckStatusResponse.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for CheckStatusRequest")
		}
	})

	t.Run("request-response - LinesRequest", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.LinesRequest == nil {
				t.Error("expected payload to contain a LinesRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/LinesDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/LinesRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for LinesRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/LinesDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for LinesRequest")
		}
	})

	t.Run("request-response - ProductCategoriesRequest", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.ProductCategoriesRequest == nil {
				t.Error("expected payload to contain a ProductCategoriesRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/ProductCategoriesDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/ProductCategoriesRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for ProductCategoriesRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/ProductCategoriesDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for ProductCategoriesRequest")
		}
	})

	t.Run("request-response - ServiceFeaturesRequest", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.ServiceFeaturesRequest == nil {
				t.Error("expected payload to contain a ServiceFeaturesRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/ServiceFeaturesDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/ServiceFeaturesRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for ServiceFeaturesRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/ServiceFeaturesDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for ServiceFeaturesRequest")
		}
	})

	t.Run("request-response - StopPointsRequest", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.StopPointsRequest == nil {
				t.Error("expected payload to contain a StopPointsRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/StopPointsDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/StopPointsRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for StopPointsRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/StopPointsDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for StopPointsRequest")
		}
	})

	t.Run("request-response - VehicleFeaturesRequest", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.VehicleFeaturesRequest == nil {
				t.Error("expected payload to contain a VehicleFeaturesRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/VehicleFeaturesDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/VehicleFeaturesRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for VehicleFeaturesRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/VehicleFeaturesDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for VehicleFeaturesRequest")
		}
	})

	t.Run("request-response - ServiceRequest", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.ServiceRequest == nil {
				t.Error("expected payload to contain a ServiceRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/EstimatedTimetableRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for ServiceRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for ServiceRequest")
		}
	})

	t.Run("request-response - DataSupplyRequest", func(t *testing.T) {
		// TODO: Make the unsupported
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.DataSupplyRequest == nil {
				t.Error("expected payload to contain a DataSupplyRequest")
			}

			resp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer s.Close()

		b, err := ioutil.ReadFile("../test_resources/DataSupplyRequest.xml")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", s.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &s.URL,
		}

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for DataSupplyRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for DataSupplyRequest")
		}
	})

	t.Run("publish-subscribe - SubscriptionRequest then ServiceDelivery", func(t *testing.T) {
		// Target server for subscription
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.SubscriptionRequest == nil {
				t.Error("expected payload to contain a SubscriptionRequest")
			}

			if *payload.SubscriptionRequest.ConsumerAddress != serverURL {
				t.Errorf("expected payload ConsumerAddress to equal %s, got %s", serverURL, *payload.SubscriptionRequest.ConsumerAddress)
			}

			resp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableSubscriptionResponse.xml")
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer ts.Close()

		// Client server where data is to be published to
		csCalled := false
		cs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			csCalled = true
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.ServiceDelivery == nil {
				t.Error("expected payload to contain ServiceDelivery")
			}

			exp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableDelivery.xml")
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(b, exp) {
				t.Error("unexpected payload received by ConsumerAddress")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer cs.Close()

		// Subscription request then response
		now := time.Now()
		initialTerminationTime := time.Now().Add(time.Minute)

		subscriptionRequest := siri.Siri{
			SubscriptionRequest: &siri.SubscriptionRequest{
				BaseRequest: &siri.BaseRequest{
					RequestTimestamp: &now,
				},
				RequestorRef:    aws.String("FOO"),
				ConsumerAddress: &cs.URL,
				EstimatedTimetableSubscriptionRequest: []*siri.EstimatedTimetableSubscriptionRequest{
					{
						BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
							SubscriberRef:          aws.String("BAR"),
							SubscriptionIdentifier: aws.String("123"),
							InitialTerminationTime: &initialTerminationTime,
						},
					},
				},
			},
		}

		b, err := xml.Marshal(subscriptionRequest)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &ts.URL,
		}

		defer cleanUpTickers(t, &sp)

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for SubscriptionRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/EstimatedTimetableSubscriptionResponse.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for EstimatedTimetableSubscriptionRequest")
		}

		if *sp.Subscriptions[0].SubscriptionRef != "123" {
			t.Errorf("expected SubscriptionRef %s to be stored", *sp.Subscriptions[0].SubscriptionRef)
		}

		if *sp.Subscriptions[0].ConsumerAddress != cs.URL {
			t.Errorf("expected subscription %s with URL %s to be stored", *sp.Subscriptions[0].SubscriptionRef, cs.URL)
		}

		if sp.HeartbeatSubscriptions[cs.URL] == nil {
			t.Errorf("expected heartbeat ticker to have been set for subscription %s", *sp.Subscriptions[0].SubscriptionRef)
		}

		// ServiceDelivery published
		etd, err := ioutil.ReadFile("../test_resources/EstimatedTimetableDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		pub := httptest.NewRequest("POST", "/", bytes.NewReader(etd))

		crr := httptest.NewRecorder()

		sp.Handler(crr, pub)

		if !csCalled {
			t.Errorf("client server was not called")
			return
		}

		if crr.Code != http.StatusOK {
			t.Errorf("unexpected status code for EstimatedTimetableDelivery: got %d, want %d", crr.Code, http.StatusOK)
			return
		}
	})

	t.Run("publish-subscribe - multiple SubscriptionRequests then multiple ServiceDelivery payloads in one response", func(t *testing.T) {
		// Target server for subscription
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.SubscriptionRequest == nil {
				t.Error("expected payload to contain a SubscriptionRequest")
			}

			if *payload.SubscriptionRequest.ConsumerAddress != serverURL {
				t.Errorf("expected payload ConsumerAddress to equal %s, got %s", serverURL, *payload.SubscriptionRequest.ConsumerAddress)
			}

			filename := "../test_resources/MultipleSubscriptionResponse123.xml"
			if payload.SubscriptionRequest.EstimatedTimetableSubscriptionRequest != nil {
				filename = "../test_resources/MultipleSubscriptionResponse456.xml"
			}

			resp, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-type", "application/xml")
			if _, err := w.Write(resp); err != nil {
				t.Fatalf("cannot output SIRI response: %s", err.Error())
			}
		}))
		defer ts.Close()

		// Client server where data is to be published to
		cs1Called := false
		cs1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cs1Called = true
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.ServiceDelivery == nil {
				t.Error("expected payload to contain ServiceDelivery")
			}

			exp, err := ioutil.ReadFile("../test_resources/MultipleDeliveryExpectation456.xml")
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(b, exp) {
				t.Error("unexpected payload received by ConsumerAddress")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer cs1.Close()

		// Client server where data is to be published to
		cs2Called := false
		cs2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cs2Called = true
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.ServiceDelivery == nil {
				t.Error("expected payload to contain ServiceDelivery")
			}

			exp, err := ioutil.ReadFile("../test_resources/MultipleDeliveryExpectation123.xml")
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(b, exp) {
				t.Error("unexpected payload received by ConsumerAddress")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer cs2.Close()

		// Subscription request then response
		now := time.Now()

		subscriptionRequests := []*siri.Siri{
			{
				SubscriptionRequest: &siri.SubscriptionRequest{
					BaseRequest: &siri.BaseRequest{
						RequestTimestamp: &now,
					},
					RequestorRef:    aws.String("FOO"),
					ConsumerAddress: &cs1.URL,
					EstimatedTimetableSubscriptionRequest: []*siri.EstimatedTimetableSubscriptionRequest{
						{
							BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
								SubscriberRef:          aws.String("BAR"),
								SubscriptionIdentifier: aws.String("456"),
								InitialTerminationTime: &now,
							},
						},
					},
					ProductionTimetableSubscriptionRequest: []*siri.ProductionTimetableSubscriptionRequest{
						{
							BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
								SubscriberRef:          aws.String("BAR"),
								SubscriptionIdentifier: aws.String("456"),
								InitialTerminationTime: &now,
							},
						},
					},
				},
			},
			{
				SubscriptionRequest: &siri.SubscriptionRequest{
					BaseRequest: &siri.BaseRequest{
						RequestTimestamp: &now,
					},
					RequestorRef:    aws.String("FOO"),
					ConsumerAddress: &cs2.URL,
					StopTimetableSubscriptionRequest: []*siri.StopTimetableSubscriptionRequest{
						{
							BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
								SubscriberRef:          aws.String("BAR"),
								SubscriptionIdentifier: aws.String("123"),
								InitialTerminationTime: &now,
							},
						},
					},
					StopMonitoringSubscriptionRequest: []*siri.StopMonitoringSubscriptionRequest{
						{
							BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
								SubscriberRef:          aws.String("BAR"),
								SubscriptionIdentifier: aws.String("123"),
								InitialTerminationTime: &now,
							},
						},
					},
					VehicleMonitoringSubscriptionRequest: []*siri.VehicleMonitoringSubscriptionRequest{
						{
							BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
								SubscriberRef:          aws.String("BAR"),
								SubscriptionIdentifier: aws.String("123"),
								InitialTerminationTime: &now,
							},
						},
					},
				},
			},
		}

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &ts.URL,
		}

		defer cleanUpTickers(t, &sp)

		// Create subscriptions
		for _, subscriptionRequest := range subscriptionRequests {
			b, err := xml.Marshal(subscriptionRequest)
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()

			sp.Handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("unexpected status code for SubscriptionRequest: got %d, want %d", rr.Code, http.StatusOK)
				return
			}
		}

		if len(sp.Subscriptions) != 2 {
			t.Errorf("expected %d subscriptions to have been created, %d found", 2, len(sp.Subscriptions))
		}

		// ServiceDelivery published to client server
		etd, err := ioutil.ReadFile("../test_resources/MultipleDelivery.xml")
		if err != nil {
			t.Fatal(err)
		}

		pub := httptest.NewRequest("POST", "/", bytes.NewReader(etd))

		crr := httptest.NewRecorder()

		sp.Handler(crr, pub)

		if !cs1Called {
			t.Error("Client Server 1 not called")
			return
		}

		if !cs2Called {
			t.Error("Client Server 2 not called")
			return
		}

		if crr.Code != http.StatusOK {
			t.Errorf("unexpected status code for EstimatedTimetableDelivery: got %d, want %d", crr.Code, http.StatusOK)
			return
		}
	})

	t.Run("publish-subscribe - SubscriptionRequest then TerminateSubscriptionRequest All", func(t *testing.T) {
		// Target server for subscription
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.SubscriptionRequest != nil {
				resp, err := ioutil.ReadFile("../test_resources/MultipleSubscriptionResponse_123_456.xml")
				if err != nil {
					t.Fatal(err)
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-type", "application/xml")
				if _, err := w.Write(resp); err != nil {
					t.Fatalf("cannot output SIRI response: %s", err.Error())
				}
			} else if payload.TerminateSubscriptionRequest != nil {
				resp, err := ioutil.ReadFile("../test_resources/TerminateMultipleSubscriptionResponse.xml")
				if err != nil {
					t.Fatal(err)
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-type", "application/xml")
				if _, err := w.Write(resp); err != nil {
					t.Fatalf("cannot output SIRI response: %s", err.Error())
				}
			} else {
				t.Error("expected payload to contain a SubscriptionRequest or a TerminateSubscriptionRequest")
			}
		}))
		defer ts.Close()

		// Subscription request
		initialTerminationTime1 := time.Now().Add(time.Minute)
		initialTerminationTime2 := time.Now().Add(time.Minute * 2)

		subscriptionRequest := siri.Siri{
			SubscriptionRequest: &siri.SubscriptionRequest{
				BaseRequest: &siri.BaseRequest{
					RequestTimestamp: &initialTerminationTime1,
				},
				RequestorRef:    aws.String("FOO"),
				ConsumerAddress: aws.String("http://foo.bar"),
				SubscriptionContext: &siri.SubscriptionContext{
					HeartbeatInterval: aws.String("PT1M"),
				},
				EstimatedTimetableSubscriptionRequest: []*siri.EstimatedTimetableSubscriptionRequest{
					{
						BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
							SubscriberRef:          aws.String("BAR"),
							SubscriptionIdentifier: aws.String("123"),
							InitialTerminationTime: &initialTerminationTime1,
						},
					},
					{
						BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
							SubscriberRef:          aws.String("BAZ"),
							SubscriptionIdentifier: aws.String("456"),
							InitialTerminationTime: &initialTerminationTime2,
						},
					},
				},
			},
		}

		b, err := xml.Marshal(subscriptionRequest)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &ts.URL,
		}

		defer cleanUpTickers(t, &sp)

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for SubscriptionRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/MultipleSubscriptionResponse_123_456.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for MultipleSubscriptionRequest_123_456")
		}

		if *sp.Subscriptions[0].SubscriptionRef != "123" {
			t.Errorf("expected SubscriptionRef %s to be stored", *sp.Subscriptions[0].SubscriptionRef)
		}

		if *sp.Subscriptions[0].ConsumerAddress != "http://foo.bar" {
			t.Errorf("expected subscription %s with URL %s to be stored", *sp.Subscriptions[0].SubscriptionRef, "http://foo.bar")
		}

		if *sp.Subscriptions[1].SubscriptionRef != "456" {
			t.Errorf("expected SubscriptionRef %s to be stored", *sp.Subscriptions[1].SubscriptionRef)
		}

		if *sp.Subscriptions[1].ConsumerAddress != "http://foo.bar" {
			t.Errorf("expected subscription %s with URL %s to be stored", *sp.Subscriptions[1].SubscriptionRef, "http://foo.bar")
		}

		if sp.HeartbeatSubscriptions["http://foo.bar"] == nil {
			t.Fatalf("expected heartbeat ticker to have been set for subscription %s", *sp.Subscriptions[0].SubscriptionRef)
		}

		if sp.HeartbeatSubscriptions["http://foo.bar"].TerminationTime == nil {
			t.Error("expected heartbeat termination time to have been set for subscriptions")
		}

		if !sp.HeartbeatSubscriptions["http://foo.bar"].TerminationTime.Equal(initialTerminationTime2) {
			t.Errorf("expected heartbeat termination time to be %s, got %s", sp.HeartbeatSubscriptions["http://foo.bar"].TerminationTime.Format(time.RFC3339), initialTerminationTime2.Format(time.RFC3339))
		}

		if sp.HeartbeatSubscriptions["http://foo.bar"].Interval == nil {
			t.Error("expected heartbeat interval to have been set for subscriptions")
		}

		expectedDuration, err := duration.FromString("PT1M")
		if err != nil {
			t.Fatal(err)
		}

		if sp.HeartbeatSubscriptions["http://foo.bar"].Interval.ToDuration() != expectedDuration.ToDuration() {
			t.Errorf("expected heartbeat interval to be %s, got %s", expectedDuration.String(), sp.HeartbeatSubscriptions["http://foo.bar"].Interval.String())
		}

		// TerminateSubscriptionRequest
		terminateSubscriptionRequest := siri.Siri{
			TerminateSubscriptionRequest: &siri.TerminateSubscriptionRequest{
				BaseRequest: &siri.BaseRequest{
					RequestTimestamp: &initialTerminationTime1,
				},
				RequestorRef: aws.String("FOO"),
				All:          aws.Bool(false),
			},
		}

		tsrb, err := xml.Marshal(terminateSubscriptionRequest)
		if err != nil {
			t.Fatal(err)
		}

		tsr, err := http.NewRequest("POST", ts.URL, bytes.NewReader(tsrb))
		if err != nil {
			t.Fatal(err)
		}

		trr := httptest.NewRecorder()

		sp.Handler(trr, tsr)

		if trr.Code != http.StatusOK {
			t.Errorf("unexpected status code for EstimatedTimetableDelivery: got %d, want %d", trr.Code, http.StatusOK)
			return
		}

		tsrexp, err := ioutil.ReadFile("../test_resources/TerminateMultipleSubscriptionResponse.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(trr.Body.Bytes(), tsrexp) {
			t.Error("unexpected response for TerminateSubscriptionRequest")
		}

		if len(sp.Subscriptions) > 0 {
			t.Errorf("expected no subscriptions to be stored; %d stored", len(sp.Subscriptions))
		}
	})

	t.Run("publish-subscribe - SubscriptionRequest then TerminateSubscriptionRequest Specific", func(t *testing.T) {
		// Target server for subscription
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.SubscriptionRequest != nil {
				resp, err := ioutil.ReadFile("../test_resources/MultipleSubscriptionResponse_123_456.xml")
				if err != nil {
					t.Fatal(err)
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-type", "application/xml")
				if _, err := w.Write(resp); err != nil {
					t.Fatalf("cannot output SIRI response: %s", err.Error())
				}
			} else if payload.TerminateSubscriptionRequest != nil {
				resp, err := ioutil.ReadFile("../test_resources/TerminateSubscriptionResponse.xml")
				if err != nil {
					t.Fatal(err)
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-type", "application/xml")
				if _, err := w.Write(resp); err != nil {
					t.Fatalf("cannot output SIRI response: %s", err.Error())
				}
			} else {
				t.Error("expected payload to contain a SubscriptionRequest or a TerminateSubscriptionRequest")
			}
		}))
		defer ts.Close()

		// Subscription request
		now := time.Now()
		initialTerminationTime := time.Now().Add(time.Minute)

		subscriptionRequest := siri.Siri{
			SubscriptionRequest: &siri.SubscriptionRequest{
				BaseRequest: &siri.BaseRequest{
					RequestTimestamp: &now,
				},
				RequestorRef:    aws.String("FOO"),
				ConsumerAddress: aws.String("http://foo.bar"),
				EstimatedTimetableSubscriptionRequest: []*siri.EstimatedTimetableSubscriptionRequest{
					{
						BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
							SubscriberRef:          aws.String("BAR"),
							SubscriptionIdentifier: aws.String("123"),
							InitialTerminationTime: &initialTerminationTime,
						},
					},
					{
						BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
							SubscriberRef:          aws.String("BAZ"),
							SubscriptionIdentifier: aws.String("456"),
							InitialTerminationTime: &initialTerminationTime,
						},
					},
				},
			},
		}

		b, err := xml.Marshal(subscriptionRequest)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &ts.URL,
		}

		defer cleanUpTickers(t, &sp)

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for SubscriptionRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/MultipleSubscriptionResponse_123_456.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for EstimatedTimetableSubscriptionRequest")
		}

		if *sp.Subscriptions[0].SubscriptionRef != "123" {
			t.Errorf("expected SubscriptionRef %s to be stored", *sp.Subscriptions[0].SubscriptionRef)
		}

		if *sp.Subscriptions[0].ConsumerAddress != "http://foo.bar" {
			t.Errorf("expected subscription %s with URL %s to be stored", *sp.Subscriptions[0].SubscriptionRef, "http://foo.bar")
		}

		if *sp.Subscriptions[1].SubscriptionRef != "456" {
			t.Errorf("expected SubscriptionRef %s to be stored", *sp.Subscriptions[0].SubscriptionRef)
		}

		if *sp.Subscriptions[1].ConsumerAddress != "http://foo.bar" {
			t.Errorf("expected subscription %s with URL %s to be stored", *sp.Subscriptions[0].SubscriptionRef, "http://foo.bar")
		}

		if sp.HeartbeatSubscriptions["http://foo.bar"] == nil {
			t.Errorf("expected heartbeat ticker to have been set for subscription %s", *sp.Subscriptions[0].SubscriptionRef)
		}

		// TerminateSubscriptionRequest
		terminateSubscriptionRequest := siri.Siri{
			TerminateSubscriptionRequest: &siri.TerminateSubscriptionRequest{
				BaseRequest: &siri.BaseRequest{
					RequestTimestamp: &now,
				},
				RequestorRef: aws.String("FOO"),
				SubscriptionRef: []*string{
					aws.String("456"),
				},
			},
		}

		tsrb, err := xml.Marshal(terminateSubscriptionRequest)
		if err != nil {
			t.Fatal(err)
		}

		tsr, err := http.NewRequest("POST", ts.URL, bytes.NewReader(tsrb))
		if err != nil {
			t.Fatal(err)
		}

		trr := httptest.NewRecorder()

		sp.Handler(trr, tsr)

		if trr.Code != http.StatusOK {
			t.Errorf("unexpected status code for EstimatedTimetableDelivery: got %d, want %d", trr.Code, http.StatusOK)
			return
		}

		tsrexp, err := ioutil.ReadFile("../test_resources/TerminateSubscriptionResponse.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(trr.Body.Bytes(), tsrexp) {
			t.Error("unexpected response for TerminateSubscriptionRequest")
		}

		if *sp.Subscriptions[0].SubscriptionRef != "123" {
			t.Errorf("expected SubscriptionRef %s to be stored", *sp.Subscriptions[0].SubscriptionRef)
		}

		if sp.HeartbeatSubscriptions["http://foo.bar"] == nil {
			t.Errorf("expected HeartbeatTicker to be present for subscription")
		}

		if len(sp.Subscriptions) > 1 {
			t.Errorf("expected subscription %s to have been removed from the map", "456")
		}
	})

	t.Run("publish-subscribe - HeartbeatNotification", func(t *testing.T) {
		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Minutes: 5,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  nil,
			TargetURL:     nil,
		}

		// HeartbeatNotification published
		hbn, err := ioutil.ReadFile("../test_resources/HeartbeatNotification.xml")
		if err != nil {
			t.Fatal(err)
		}

		pub := httptest.NewRequest("POST", "/", bytes.NewReader(hbn))

		rr := httptest.NewRecorder()

		sp.Handler(rr, pub)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for HeartbeatNotification: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		if sp.LatestHeartbeat == nil {
			t.Error("LatestHeartbeat should be stored")
			return
		}

		payload := siri.Siri{}

		if err := xml.Unmarshal(*sp.LatestHeartbeat, &payload); err != nil {
			t.Error(err)
		}

		if *payload.HeartbeatNotification.Status != true {
			t.Error("HeartbeatNotification does not have expected data")
		}
	})

	t.Run("publish-subscribe - Subscription then HeartbeatNotification and initial termination time reached", func(t *testing.T) {
		// Target server for subscription
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.SubscriptionRequest != nil {
				resp, err := ioutil.ReadFile("../test_resources/MultipleSubscriptionResponse_123_456.xml")
				if err != nil {
					t.Fatal(err)
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-type", "application/xml")
				if _, err := w.Write(resp); err != nil {
					t.Fatalf("cannot output SIRI response: %s", err.Error())
				}
			} else if payload.TerminateSubscriptionRequest != nil {
				resp, err := ioutil.ReadFile("../test_resources/TerminateSubscriptionResponse.xml")
				if err != nil {
					t.Fatal(err)
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-type", "application/xml")
				if _, err := w.Write(resp); err != nil {
					t.Fatalf("cannot output SIRI response: %s", err.Error())
				}
			} else {
				t.Error("expected payload to contain a SubscriptionRequest or a TerminateSubscriptionRequest")
			}
		}))
		defer ts.Close()

		// Client server where data is to be published to
		csCallCount := 0
		muxCsCallCount := sync.Mutex{}
		cs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			muxCsCallCount.Lock()
			csCallCount++
			muxCsCallCount.Unlock()
			payload := siri.Siri{}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err := xml.Unmarshal(b, &payload); err != nil {
				t.Fatal(err)
			}

			if payload.HeartbeatNotification == nil {
				t.Error("expected payload to contain HeartbeatNotification")
			}

			exp, err := ioutil.ReadFile("../test_resources/HeartbeatNotification.xml")
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(b, exp) {
				t.Error("unexpected payload received by ConsumerAddress")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer cs.Close()

		// Subscription request
		now := time.Now()
		initialTerminationTime := time.Now().Add(time.Second * 2)

		subscriptionRequest := siri.Siri{
			SubscriptionRequest: &siri.SubscriptionRequest{
				BaseRequest: &siri.BaseRequest{
					RequestTimestamp: &now,
				},
				RequestorRef:    aws.String("FOO"),
				ConsumerAddress: &cs.URL,
				EstimatedTimetableSubscriptionRequest: []*siri.EstimatedTimetableSubscriptionRequest{
					{
						BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
							SubscriberRef:          aws.String("BAR"),
							SubscriptionIdentifier: aws.String("123"),
							InitialTerminationTime: &initialTerminationTime,
						},
					},
					{
						BaseSubscriptionRequest: &siri.BaseSubscriptionRequest{
							SubscriberRef:          aws.String("BAZ"),
							SubscriptionIdentifier: aws.String("456"),
							InitialTerminationTime: &initialTerminationTime,
						},
					},
				},
			},
		}

		b, err := xml.Marshal(subscriptionRequest)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()

		sp := SiriProxy{
			DefaultHeartbeatNotificationInterval: &duration.Duration{
				Seconds: 1,
			},
			HeartbeatSubscriptions: make(map[string]*SiriHeartbeatSubscription),
			HTTPClientTimeout:      &httpClientTimeout,
			LatestHeartbeat:        nil,
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			ServerURL:     &serverURL,
			Subscriptions: []*SiriSubscription{},
			TargetClient:  &http.Client{},
			TargetURL:     &ts.URL,
		}

		defer cleanUpTickers(t, &sp)

		sp.Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("unexpected status code for SubscriptionRequest: got %d, want %d", rr.Code, http.StatusOK)
			return
		}

		exp, err := ioutil.ReadFile("../test_resources/MultipleSubscriptionResponse_123_456.xml")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(rr.Body.Bytes(), exp) {
			t.Error("unexpected response for EstimatedTimetableSubscriptionRequest")
		}

		sp.muxHeartbeatSubscriptions.Lock()
		if sp.HeartbeatSubscriptions[cs.URL] == nil {
			t.Errorf("expected heartbeat ticker to have been set for subscription %s", *sp.Subscriptions[0].SubscriptionRef)
		}
		sp.muxHeartbeatSubscriptions.Unlock()

		// HeartbeatNotification published
		hbn, err := ioutil.ReadFile("../test_resources/HeartbeatNotification.xml")
		if err != nil {
			t.Fatal(err)
		}

		pub := httptest.NewRequest("POST", "/", bytes.NewReader(hbn))

		srr := httptest.NewRecorder()

		sp.Handler(srr, pub)

		if srr.Code != http.StatusOK {
			t.Errorf("unexpected status code for HeartbeatNotification: got %d, want %d", srr.Code, http.StatusOK)
			return
		}

		sp.muxLatestHeartbeat.Lock()
		if sp.LatestHeartbeat == nil {
			t.Error("LatestHeartbeat should be stored")
			sp.muxLatestHeartbeat.Unlock()
			return
		}
		sp.muxLatestHeartbeat.Unlock()

		// HeartbeatNotification sent to client
		done := make(chan struct{}, 1)
		timer := time.NewTimer(time.Second * 3)
		go func() {
			defer close(done)
			<-timer.C
			muxCsCallCount.Lock()
			if csCallCount == 0 {
				t.Error("expected client server to have been called")
			}
			muxCsCallCount.Unlock()

			sp.muxHeartbeatSubscriptions.Lock()
			if sp.HeartbeatSubscriptions[cs.URL] != nil {
				t.Error("expected client subscription to have been deleted after termination time has passed")
			}
			sp.muxHeartbeatSubscriptions.Unlock()
		}()

		<-done
	})
}
