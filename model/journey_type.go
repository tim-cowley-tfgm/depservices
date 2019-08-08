package model

type JourneyType string

const (
	Bus   JourneyType = "bus"
	Train JourneyType = "train"
	Tram  JourneyType = "tram"
)

func GetJourneyType(atcocode string) JourneyType {
	atcoRegion := atcocode[0:3]

	if atcoRegion == "910" {
		return Train
	}

	if atcoRegion == "940" {
		return Tram
	}

	return Bus
}
