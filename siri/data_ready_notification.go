package siri

type DataReadyNotification struct {
	*BaseResponse
	ProducerRef *string
}
