package main

import (
	"bytes"
	"encoding/xml"
	duration "github.com/ChannelMeter/iso8601duration"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/siri"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type SiriProxy struct {
	DefaultHeartbeatNotificationInterval *duration.Duration
	LatestHeartbeat                      *[]byte
	muxLatestHeartbeat                   sync.Mutex
	HeartbeatSubscriptions               map[string]*SiriHeartbeatSubscription
	muxHeartbeatSubscriptions            sync.Mutex
	HTTPClientTimeout                    *time.Duration
	Logger                               *dlog.Logger
	ServerURL                            *string
	Subscriptions                        []*SiriSubscription
	TargetClient                         *http.Client
	TargetURL                            *string
}

type SiriSubscription struct {
	ConsumerAddress        *string
	HeartbeatInterval      *duration.Duration
	RequestorRef           *string
	InitialTerminationTime *time.Time
	SubscriptionRef        *string
}

type SiriServiceDelivery struct {
	SubscriberRef   string
	SubscriptionRef string
	Header          []byte
	Payload         []byte
	Footer          []byte
}

type SiriHeartbeatSubscription struct {
	Interval        *duration.Duration
	TerminationTime *time.Time
	Ticker          *time.Ticker
	Timer           *time.Timer
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("siri-proxy: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")
	serverURL, exists := os.LookupEnv("SIRI_PROXY_SERVER_URL")
	if !exists || serverURL == "" {
		logger.Fatal("SIRI_PROXY_SERVER_URL not set in environment")
	}

	if _, err := url.Parse(serverURL); err != nil {
		logger.Fatal(errors.Wrapf(err, "SIRI_PROXY_SERVER_URL %s is invalid", serverURL))
	}

	serverPort, exists := os.LookupEnv("SIRI_PROXY_SERVER_PORT")
	if !exists || serverPort == "" {
		logger.Fatal("SIRI_PROXY_SERVER_PORT not set in environment")
	}

	targetURL, exists := os.LookupEnv("SIRI_PROXY_TARGET_URL")
	if !exists || targetURL == "" {
		logger.Fatal("SIRI_PROXY_TARGET_URL not set in environment")
	}

	if _, err := url.Parse(targetURL); err != nil {
		logger.Fatal(errors.Wrapf(err, "SIRI_PROXY_TARGET_URL %s is invalid", targetURL))
	}

	defaultHeartbeatNotificationIntervalString, exists := os.LookupEnv("SIRI_DEFAULT_HEARTBEAT_NOTIFICATION_INTERVAL")
	if !exists || defaultHeartbeatNotificationIntervalString == "" {
		defaultHeartbeatNotificationIntervalString = "PT5M"
		logger.Printf("SIRI_DEFAULT_HEARTBEAT_NOTIFICATION_INTERVAL not set in environment; set to default value of %s", defaultHeartbeatNotificationIntervalString)
	}

	defaultHeartbeatNotificationInterval, err := duration.FromString(defaultHeartbeatNotificationIntervalString)
	if err != nil {
		logger.Fatal(errors.Wrapf(err, "invalid SIRI_DEFAULT_HEARTBEAT_NOTIFICATION_INTERVAL provided: %s", defaultHeartbeatNotificationIntervalString))
	}

	httpClientTimeoutString, exists := os.LookupEnv("HTTP_CLIENT_TIMEOUT")
	if !exists || httpClientTimeoutString == "" {
		httpClientTimeoutString = "10"
		logger.Printf("HTTP_CLIENT_TIMEOUT not set in environment; set to default value of %s seconds", httpClientTimeoutString)
	}

	httpClientTimeoutSecs, err := strconv.Atoi(httpClientTimeoutString)
	if err != nil {
		logger.Fatal(errors.Wrapf(err, "invalid HTTP_CLIENT_TIMEOUT value %s", httpClientTimeoutString))
	}

	httpClientTimeout := time.Duration(httpClientTimeoutSecs) * time.Second

	sp := SiriProxy{
		DefaultHeartbeatNotificationInterval: defaultHeartbeatNotificationInterval,
		HeartbeatSubscriptions:               make(map[string]*SiriHeartbeatSubscription),
		HTTPClientTimeout:                    &httpClientTimeout,
		LatestHeartbeat:                      nil,
		Logger:                               dlog.NewLogger(),
		ServerURL:                            &serverURL,
		Subscriptions:                        []*SiriSubscription{},
		TargetClient: &http.Client{
			Timeout: httpClientTimeout,
		},
		TargetURL: &targetURL,
	}

	http.HandleFunc("/", sp.Handler)

	// Start the server
	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		logger.Fatal(err)
	}
}

func (sp *SiriProxy) Handler(w http.ResponseWriter, r *http.Request) {
	sp.Logger.Debug("Handler")

	if r.Method != "POST" {
		sp.Logger.Debugf("invalid method %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	payload := siri.Siri{}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.Logger.Printf("could not read body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		sp.Logger.Printf("could not close body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := xml.Unmarshal(b, &payload); err != nil {
		sp.Logger.Printf("could not unmarshal request to SIRI: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Reinitialise body buffer
	r.Body = ioutil.NopCloser(bytes.NewBuffer(b))

	switch {
	// Request/Response methodology with not interesting data - handle in one hit
	case payload.CapabilitiesRequest != nil,
		payload.CheckStatusRequest != nil,
		payload.LinesRequest != nil,
		payload.ProductCategoriesRequest != nil,
		payload.ServiceFeaturesRequest != nil,
		payload.StopPointsRequest != nil,
		payload.VehicleFeaturesRequest != nil:
		sp.handleStandardRequestResponse(w, r)
	// Request/Response methodology with interesting data - handle in one hit
	case payload.ServiceRequest != nil,
		payload.DataSupplyRequest != nil:
		sp.handleServiceRequestResponse(w, r)
	// Subscription request - need to store data
	case payload.SubscriptionRequest != nil:
		sp.handleSubscriptionRequestResponse(w, r)
	// Terminate subscription request - need to remove stored data
	case payload.TerminateSubscriptionRequest != nil:
		// TODO: Check response from target server before removing subscription?
		if err := sp.removeSubscription(&payload); err != nil {
			sp.Logger.Printf("could not remove subscription: %s", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sp.handleStandardRequestResponse(w, r)
	// Published data with interesting data - need to forward to stored consumer address
	case payload.ServiceDelivery != nil:
		sp.handleServiceDeliveryPublication(w, r)
		return
	// Published data with not interesting data - forward to stored consumer address
	case payload.HeartbeatNotification != nil:
		sp.handleHeartbeatNotificationPublication(w, r)
		return
	}
}

func (sp *SiriProxy) handleStandardRequestResponse(w http.ResponseWriter, r *http.Request) {
	sp.Logger.Debug("handleStandardRequestResponse")

	rb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.Logger.Printf("could not read body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		sp.Logger.Printf("could not close body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest("POST", *sp.TargetURL, bytes.NewReader(rb))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-type", "application/xml")

	resp, err := sp.TargetClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			sp.Logger.Printf("cannot close response: %s", err.Error())
		}
	}()

	wb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	if _, err := w.Write(wb); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (sp *SiriProxy) handleServiceRequestResponse(w http.ResponseWriter, r *http.Request) {
	sp.Logger.Debug("handleServiceRequestResponse")

	rb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.Logger.Printf("could not read body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		sp.Logger.Printf("could not close body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest("POST", *sp.TargetURL, bytes.NewReader(rb))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-type", "application/xml")

	resp, err := sp.TargetClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			sp.Logger.Printf("cannot close response: %s", err.Error())
		}
	}()

	wb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	if _, err := w.Write(wb); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Do something with the response payload (in a goroutine?)
}

func (sp *SiriProxy) handleServiceDeliveryPublication(w http.ResponseWriter, r *http.Request) {
	sp.Logger.Debug("handleServiceDeliveryPublication")

	rb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.Logger.Printf("could not read body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		sp.Logger.Printf("could not close body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get subscription
	payload := siri.Siri{}

	if err := xml.Unmarshal(rb, &payload); err != nil {
		sp.Logger.Printf("could not unmarshal request to SIRI: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.ServiceDelivery == nil {
		sp.Logger.Printf("")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	reHeaderPayloadBoundary := regexp.MustCompile(`[ \t]*<(EstimatedTimetable|ProductionTimetable|StopMonitoring|StopTimetable|VehicleMonitoring)Delivery(\b|>)`)
	headerEnd := reHeaderPayloadBoundary.FindIndex(rb)
	if headerEnd == nil {
		sp.Logger.Print("no service delivery header in payload")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	header := rb[0:headerEnd[0]]

	payloadStarts := reHeaderPayloadBoundary.FindAllIndex(rb, -1)
	if payloadStarts == nil {
		sp.Logger.Print("no service delivery payloads")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rePayloadEnd := regexp.MustCompile(`</(EstimatedTimetable|ProductionTimetable|StopMonitoring|StopTimetable|VehicleMonitoring)Delivery>[\n\r]*`)
	payloadEnds := rePayloadEnd.FindAllIndex(rb, -1)
	if payloadEnds == nil {
		sp.Logger.Print("no service delivery payload terminators")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(payloadStarts) != len(payloadEnds) {
		sp.Logger.Print("service delivery payload mismatch")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	footer := rb[payloadEnds[len(payloadEnds)-1][1]:]

	var deliveries []*SiriServiceDelivery

	for i := 0; i < len(payloadStarts); i++ {
		sd := SiriServiceDelivery{}

		p := rb[payloadStarts[i][0]:payloadEnds[i][1]]

		if err := xml.Unmarshal(p, &sd); err != nil {
			sp.Logger.Print("could not unmarshal service delivery XML")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		appended := false
		for i, d := range deliveries {
			if sd.SubscriptionRef == d.SubscriptionRef &&
				sd.SubscriberRef == d.SubscriberRef {
				deliveries[i].Payload = append(d.Payload, p...)
				appended = true
				break
			}
		}

		if !appended {
			deliveries = append(deliveries, &SiriServiceDelivery{
				SubscriberRef:   sd.SubscriberRef,
				SubscriptionRef: sd.SubscriptionRef,
				Header:          header,
				Payload:         p,
				Footer:          footer,
			})
		}
	}

	// Get subscription consumer address for each delivery
	// Send to consumer address
	for _, d := range deliveries {
		var body []byte
		body = append(body, d.Header...)
		body = append(body, d.Payload...)
		body = append(body, d.Footer...)

		for _, sub := range sp.Subscriptions {
			if *sub.SubscriptionRef == d.SubscriptionRef &&
				*sub.RequestorRef == d.SubscriberRef {
				req, err := http.NewRequest("POST", *sub.ConsumerAddress, bytes.NewReader(body))
				if err != nil {
					sp.Logger.Printf("could not create HTTP request for %s", *sub.ConsumerAddress)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				req.Header.Set("Content-type", "application/xml")

				subClient := http.Client{
					Timeout: time.Second * 10,
				}

				resp, err := subClient.Do(req)
				if err != nil {
					sp.Logger.Printf("HTTP request to %s failed", *sub.ConsumerAddress)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if resp.StatusCode >= 300 {
					sp.Logger.Printf("Bad response for HTTP request to %s failed", *sub.ConsumerAddress)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

// Gets the heartbeat notification and stores it locally
// It is not passed on directly to subscribers - this is handled separately
func (sp *SiriProxy) handleHeartbeatNotificationPublication(w http.ResponseWriter, r *http.Request) {
	sp.Logger.Debug("handleHeartbeatNotificationPublication")

	rb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.Logger.Printf("could not read body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		sp.Logger.Printf("could not close body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sp.muxLatestHeartbeat.Lock()
	sp.LatestHeartbeat = &rb
	sp.muxLatestHeartbeat.Unlock()
}

func (sp *SiriProxy) handleSubscriptionRequestResponse(w http.ResponseWriter, r *http.Request) {
	sp.Logger.Debug("handleSubscriptionRequestResponse")

	requestPayload := siri.Siri{}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.Logger.Printf("could not read body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		sp.Logger.Printf("could not close body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := xml.Unmarshal(b, &requestPayload); err != nil {
		sp.Logger.Printf("could not unmarshal request to SIRI: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if requestPayload.SubscriptionRequest.RequestorRef == nil {
		sp.Logger.Print("RequestorRef cannot be nil")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if requestPayload.SubscriptionRequest.ConsumerAddress == nil {
		sp.Logger.Print("ConsumerAddress cannot be nil")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var heartbeatInterval *duration.Duration

	if requestPayload.SubscriptionRequest.SubscriptionContext != nil &&
		requestPayload.SubscriptionRequest.SubscriptionContext.HeartbeatInterval != nil {
		heartbeatInterval, err = duration.FromString(*requestPayload.SubscriptionRequest.SubscriptionContext.HeartbeatInterval)
		if err != nil {
			sp.Logger.Print("HeartbeatInterval value is invalid")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		heartbeatInterval = sp.DefaultHeartbeatNotificationInterval
	}

	subscriptionsInRequest := sp.getSubscriptionsFromRequest(&requestPayload)

	var subscriptionsToStore, successfulSubscriptionsToStore []*SiriSubscription

	for _, subs := range subscriptionsInRequest {
		if subs.SubscriptionIdentifier == nil {
			sp.Logger.Print("SubscriptionIdentifier cannot be nil")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if subs.InitialTerminationTime == nil {
			sp.Logger.Print("InitialTerminationTime cannot be nil")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		subscriptionsToStore = append(subscriptionsToStore, &SiriSubscription{
			ConsumerAddress:        requestPayload.SubscriptionRequest.ConsumerAddress,
			RequestorRef:           requestPayload.SubscriptionRequest.RequestorRef,
			SubscriptionRef:        subs.SubscriptionIdentifier,
			InitialTerminationTime: subs.InitialTerminationTime,
			HeartbeatInterval:      heartbeatInterval,
		})
	}

	// Reinitialise body buffer
	r.Body = ioutil.NopCloser(bytes.NewBuffer(b))

	rb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.Logger.Printf("could not read body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		sp.Logger.Printf("could not close body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Replace consumer address with the address of this proxy server
	reConsumerAddress := regexp.MustCompile("<ConsumerAddress>(.+)</ConsumerAddress>")
	consumerAddressB := reConsumerAddress.FindSubmatch(rb)
	if consumerAddressB == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rb = bytes.Replace(rb, []byte(*requestPayload.SubscriptionRequest.ConsumerAddress), []byte(*sp.ServerURL), 1)

	req, err := http.NewRequest("POST", *sp.TargetURL, bytes.NewReader(rb))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-type", "text/xml")

	resp, err := sp.TargetClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			sp.Logger.Printf("cannot close response: %s", err.Error())
		}
	}()

	// Check subscription statuses in response
	wb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := resp.Body.Close(); err != nil {
		sp.Logger.Printf("could not close response body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responsePayload := &siri.Siri{}

	if err := xml.Unmarshal(wb, &responsePayload); err != nil {
		sp.Logger.Printf("could not unmarshal response to SIRI: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if responsePayload.SubscriptionResponse == nil {
		sp.Logger.Printf("SubscriptionRequest failed:\n%s", string(wb))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Remove subscription from subscriptionsToStore if its status is false
	for _, responseStatus := range responsePayload.SubscriptionResponse.ResponseStatus {
		if responseStatus.SubscriptionRef == nil {
			sp.Logger.Print("SubscriptionRef cannot be nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if responseStatus.Status == nil {
			sp.Logger.Print("Status cannot be nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, sub := range subscriptionsToStore {
			if *sub.SubscriptionRef == *responseStatus.SubscriptionRef &&
				*responseStatus.Status {
				successfulSubscriptionsToStore = append(successfulSubscriptionsToStore, sub)
			}
		}
	}

	subscriptionsToStore = successfulSubscriptionsToStore

	// Remove updated subscriptions from those already stored
	i := 0
	for _, sub := range sp.Subscriptions {
		update := false
		for _, toStore := range subscriptionsToStore {
			if *sub.RequestorRef == *toStore.RequestorRef &&
				*sub.SubscriptionRef == *toStore.SubscriptionRef {
				update = true
				break
			}
		}

		if !update {
			sp.Subscriptions[i] = sub
			i++
		}
	}

	sp.Subscriptions = sp.Subscriptions[:i]

	// Remove duplicates from subscriptionsToStore
	seen := make(map[string]struct{}, len(subscriptionsToStore))
	i = 0
	for _, sub := range subscriptionsToStore {
		key := *sub.RequestorRef + *sub.SubscriptionRef
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		subscriptionsToStore[i] = sub
		i++
	}
	subscriptionsToStore = subscriptionsToStore[:i]

	// Store
	sp.Subscriptions = append(sp.Subscriptions, subscriptionsToStore...)

	sp.initialiseHeartbeatNotifications(*requestPayload.SubscriptionRequest.ConsumerAddress)

	// Reinitialise write body buffer
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(wb))

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	if _, err := w.Write(wb); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (sp *SiriProxy) initialiseHeartbeatNotifications(consumerAddress string) {
	var latestTerminationTime *time.Time
	var minimumFrequency *duration.Duration

	for _, sub := range sp.Subscriptions {
		// Skip subscriptions which aren't from this address
		if *sub.ConsumerAddress != consumerAddress {
			continue
		}

		if latestTerminationTime == nil ||
			sub.InitialTerminationTime.After(*latestTerminationTime) {
			latestTerminationTime = sub.InitialTerminationTime
		}

		if minimumFrequency == nil ||
			sub.HeartbeatInterval.ToDuration() < minimumFrequency.ToDuration() {
			minimumFrequency = sub.HeartbeatInterval
		}
	}

	if latestTerminationTime == nil {
		now := time.Now()
		latestTerminationTime = &now
	}

	if minimumFrequency == nil {
		minimumFrequency = sp.DefaultHeartbeatNotificationInterval
	}

	// Check for existing heartbeat subscription and remove it
	sp.muxHeartbeatSubscriptions.Lock()
	if sp.HeartbeatSubscriptions[consumerAddress] != nil {
		sp.HeartbeatSubscriptions[consumerAddress].Ticker.Stop()
		sp.HeartbeatSubscriptions[consumerAddress].Timer.Stop()
		delete(sp.HeartbeatSubscriptions, consumerAddress)
	}
	sp.muxHeartbeatSubscriptions.Unlock()

	// Initialise new heartbeat subscription based on new parameters
	heartbeatSubscription := &SiriHeartbeatSubscription{
		Interval:        minimumFrequency,
		TerminationTime: latestTerminationTime,
		Ticker:          time.NewTicker(minimumFrequency.ToDuration()),
		Timer:           time.NewTimer(time.Until(*latestTerminationTime)),
	}

	go sp.initialiseHeartbeatNotificationTicker(consumerAddress, heartbeatSubscription)

	go sp.initialiseHeartbeatNotificationTimer(consumerAddress, heartbeatSubscription)

	// Store heartbeat subscription
	sp.muxHeartbeatSubscriptions.Lock()
	sp.HeartbeatSubscriptions[consumerAddress] = heartbeatSubscription
	sp.muxHeartbeatSubscriptions.Unlock()
}

func (sp *SiriProxy) removeSubscription(payload *siri.Siri) error {
	sp.Logger.Debug("removeSubscription")

	if payload.TerminateSubscriptionRequest.RequestorRef == nil {
		return errors.New("RequestorRef cannot be nil")
	}

	if payload.TerminateSubscriptionRequest.All != nil {
		i := 0
		for _, sub := range sp.Subscriptions {
			if *sub.RequestorRef != *payload.TerminateSubscriptionRequest.RequestorRef {
				sp.Subscriptions[i] = sub
				i++

			}
		}
		sp.Subscriptions = sp.Subscriptions[:i]
		return nil
	} else {
		if payload.TerminateSubscriptionRequest.SubscriptionRef == nil {
			return errors.New("SubscriptionRef cannot be nil if the All key is not set")
		}

		for _, subscriptionRef := range payload.TerminateSubscriptionRequest.SubscriptionRef {
			if subscriptionRef == nil {
				return errors.New("SubscriptionRef cannot be nil")
			}

			i := 0
			for _, sub := range sp.Subscriptions {
				if *sub.RequestorRef != *payload.TerminateSubscriptionRequest.RequestorRef ||
					(*sub.RequestorRef == *payload.TerminateSubscriptionRequest.RequestorRef &&
						*sub.SubscriptionRef != *subscriptionRef) {
					sp.Subscriptions[i] = sub
					i++
				}
			}

			sp.Subscriptions = sp.Subscriptions[:i]
		}
	}

	// Heartbeat notifications - remove redundant
	for consumerAddress, hs := range sp.HeartbeatSubscriptions {
		var found *SiriSubscription
		for _, sub := range sp.Subscriptions {
			if consumerAddress == *sub.ConsumerAddress {
				found = sub
				break
			}
		}

		if found == nil {
			hs.Ticker.Stop()
			hs.Timer.Stop()
			sp.muxHeartbeatSubscriptions.Lock()
			delete(sp.HeartbeatSubscriptions, consumerAddress)
			sp.muxHeartbeatSubscriptions.Unlock()
			continue
		}

		// Update if interval has changed
		if hs.Interval.ToDuration() != found.HeartbeatInterval.ToDuration() ||
			!hs.TerminationTime.Equal(*found.InitialTerminationTime) {
			hs.Ticker.Stop()
			hs.Timer.Stop()
			go sp.initialiseHeartbeatNotificationTicker(consumerAddress, hs)
			go sp.initialiseHeartbeatNotificationTimer(consumerAddress, hs)
		}
	}

	return nil
}

func (sp *SiriProxy) getSubscriptionsFromRequest(payload *siri.Siri) []*siri.BaseSubscriptionRequest {
	var subscriptions []*siri.BaseSubscriptionRequest

	if payload.SubscriptionRequest.ConnectionMonitoringSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.ConnectionMonitoringSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	if payload.SubscriptionRequest.ConnectionTimetableSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.ConnectionTimetableSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	if payload.SubscriptionRequest.EstimatedTimetableSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.EstimatedTimetableSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	if payload.SubscriptionRequest.GeneralMessageSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.GeneralMessageSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	if payload.SubscriptionRequest.ProductionTimetableSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.ProductionTimetableSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	if payload.SubscriptionRequest.StopMonitoringSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.StopMonitoringSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	if payload.SubscriptionRequest.StopTimetableSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.StopTimetableSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	if payload.SubscriptionRequest.VehicleMonitoringSubscriptionRequest != nil {
		for _, req := range payload.SubscriptionRequest.VehicleMonitoringSubscriptionRequest {
			subscriptions = append(subscriptions, &siri.BaseSubscriptionRequest{
				SubscriberRef:          req.SubscriberRef,
				SubscriptionIdentifier: req.SubscriptionIdentifier,
				InitialTerminationTime: req.InitialTerminationTime,
			})
		}
	}

	return subscriptions
}

func (sp *SiriProxy) sendHeartbeatNotification(consumerAddress string, sub *SiriHeartbeatSubscription) {
	sp.muxLatestHeartbeat.Lock()
	defer sp.muxLatestHeartbeat.Unlock()
	if sp.LatestHeartbeat == nil {
		sp.Logger.Print("No heartbeat notification stored from target server")
		return
	}

	req, err := http.NewRequest("POST", consumerAddress, bytes.NewReader(*sp.LatestHeartbeat))
	if err != nil {
		sp.Logger.Printf("cannot create HeartbeatNotification request to %s", consumerAddress)
		return
	}
	req.Header.Set("Content-type", "application/xml")

	client := &http.Client{
		Timeout: *sp.HTTPClientTimeout,
	}
	if _, err := client.Do(req); err != nil {
		sp.Logger.Printf("cannot send HeartbeatNotification request to client %s", consumerAddress)
		return
	}
}

func (sp *SiriProxy) initialiseHeartbeatNotificationTicker(consumerAddress string, hs *SiriHeartbeatSubscription) {
	sp.Logger.Debugf("Initialise heartbeat notification ticker to %s", consumerAddress)
	defer func() {
		sp.Logger.Debugf("Terminate heartbeat notification ticker to %s", consumerAddress)
	}()
	for range hs.Ticker.C {
		sp.sendHeartbeatNotification(consumerAddress, hs)
	}
}

func (sp *SiriProxy) initialiseHeartbeatNotificationTimer(consumerAddress string, hs *SiriHeartbeatSubscription) {
	sp.Logger.Debugf("Initialise heartbeat notification timer for %s", consumerAddress)
	defer func() {
		sp.Logger.Debugf("Terminate heartbeart notification timer for %s", consumerAddress)
	}()
	<-hs.Timer.C
	hs.Ticker.Stop()
	sp.muxHeartbeatSubscriptions.Lock()
	delete(sp.HeartbeatSubscriptions, consumerAddress)
	sp.muxHeartbeatSubscriptions.Unlock()
}
