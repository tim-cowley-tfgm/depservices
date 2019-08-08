package model

import "testing"

func TestGetJourneyType(t *testing.T) {
	tests := make(map[string]JourneyType)

	tests["9400ZZMASPT"] = Tram
	tests["9100MNCRPIC"] = Train
	tests["1800NE43431"] = Bus
	tests["180GWNBS"] = Bus
	tests["2500WOOO"] = Bus

	for atcocode, journeyType := range tests {
		if GetJourneyType(atcocode) != journeyType {
			t.Errorf("got %s, want %s for %s", string(GetJourneyType(atcocode)), string(journeyType), atcocode)
		}
	}
}
