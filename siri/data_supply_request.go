package siri

type DataSupplyRequest struct {
	*BaseRequest
	ConsumerRef     *string
	NotificationRef *string
	AllData         *bool
}
