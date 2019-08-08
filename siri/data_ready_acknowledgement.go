package siri

type DataReadyAcknowledgement struct {
	*BaseResponse
	ConsumerRef *string
	Status      *bool
}
