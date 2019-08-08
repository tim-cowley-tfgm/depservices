package model

import (
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"sort"
	"testing"
	"time"
)

func TestDepartures_SortByDepartureTime(t *testing.T) {
	t.Run("should sort the departures in ascending time order", func(t *testing.T) {
		now := time.Now()

		expectedDepartureTime1 := test_helpers.AdjustTime(now, "40m").Format(time.RFC3339)
		expectedDepartureTime2 := test_helpers.AdjustTime(now, "10m").Format(time.RFC3339)

		departures := Internal{
			[]Departure{
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1234",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "20m").Format(time.RFC3339),
					ExpectedDepartureTime: &expectedDepartureTime1,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1235",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "30m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "456",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1236",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "35m").Format(time.RFC3339),
					ExpectedDepartureTime: &expectedDepartureTime2,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "789",
					OperatorCode:          "ANWE",
				},
			},
		}

		sort.Sort(ByDepartureTime(departures.Departures))

		if departures.Departures[0].JourneyRef != now.Format("2006-01-02")+"_1236" {
			t.Errorf("Expected first departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1236", departures.Departures[0].JourneyRef)
		}

		if departures.Departures[1].JourneyRef != now.Format("2006-01-02")+"_1235" {
			t.Errorf("Expected second departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1235", departures.Departures[1].JourneyRef)
		}

		if departures.Departures[2].JourneyRef != now.Format("2006-01-02")+"_1234" {
			t.Errorf("Expected third departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1234", departures.Departures[2].JourneyRef)
		}
	})

	t.Run("should sort departures with the same departure time by service number, service suffix then service prefix", func(t *testing.T) {
		now := time.Now()

		departures := Internal{
			[]Departure{
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1234",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "B12A",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1235",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1236",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "A12",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1237",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "12",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1238",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "A12B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "A12A",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "B12B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "B12",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "12B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "12A",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123A",
					OperatorCode:          "ANWE",
				},
			},
		}

		sort.Sort(ByDepartureTime(departures.Departures))

		if departures.Departures[0].ServiceNumber != "12" {
			t.Errorf("Expected first departure to have ServiceNumber `%s`; got `%s`", "12", departures.Departures[0].ServiceNumber)
		}

		if departures.Departures[1].ServiceNumber != "12A" {
			t.Errorf("Expected second departure to have ServiceNumber `%s`; got `%s`", "12A", departures.Departures[1].ServiceNumber)
		}

		if departures.Departures[2].ServiceNumber != "12B" {
			t.Errorf("Expected third departure to have ServiceNumber `%s`; got `%s`", "12B", departures.Departures[2].ServiceNumber)
		}

		if departures.Departures[3].ServiceNumber != "123" {
			t.Errorf("Expected fourth departure to have ServiceNumber `%s`; got `%s`", "123", departures.Departures[3].ServiceNumber)
		}

		if departures.Departures[4].ServiceNumber != "123A" {
			t.Errorf("Expected fifth departure to have ServiceNumber `%s`; got `%s`", "123A", departures.Departures[4].ServiceNumber)
		}

		if departures.Departures[5].ServiceNumber != "123B" {
			t.Errorf("Expected sixth departure to have ServiceNumber `%s`; got `%s`", "123B", departures.Departures[5].ServiceNumber)
		}

		if departures.Departures[6].ServiceNumber != "A12" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "A12", departures.Departures[6].ServiceNumber)
		}

		if departures.Departures[7].ServiceNumber != "A12A" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "A12A", departures.Departures[7].ServiceNumber)
		}

		if departures.Departures[8].ServiceNumber != "A12B" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "A12B", departures.Departures[8].ServiceNumber)
		}

		if departures.Departures[9].ServiceNumber != "B12" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "B12", departures.Departures[9].ServiceNumber)
		}

		if departures.Departures[10].ServiceNumber != "B12A" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "B12A", departures.Departures[10].ServiceNumber)
		}

		if departures.Departures[11].ServiceNumber != "B12B" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "B12B", departures.Departures[11].ServiceNumber)
		}
	})

	t.Run("should sort departures with the same departure time and service number by journey reference", func(t *testing.T) {
		now := time.Now()

		departures := Internal{
			[]Departure{
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1235",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "534",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1236",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "534",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1234",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "534",
					OperatorCode:          "ANWE",
				},
			},
		}

		sort.Sort(ByDepartureTime(departures.Departures))

		if departures.Departures[0].JourneyRef != now.Format("2006-01-02")+"_1234" {
			t.Errorf("Expected first departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1234", departures.Departures[0].JourneyRef)
		}

		if departures.Departures[1].JourneyRef != now.Format("2006-01-02")+"_1235" {
			t.Errorf("Expected second departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1235", departures.Departures[1].JourneyRef)
		}

		if departures.Departures[2].JourneyRef != now.Format("2006-01-02")+"_1236" {
			t.Errorf("Expected third departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1236", departures.Departures[2].JourneyRef)
		}
	})
}

func TestDepartures_SortByServiceNumber(t *testing.T) {
	t.Run("should sort departures by service number, service suffix and then service prefix", func(t *testing.T) {
		now := time.Now()

		departures := Internal{
			[]Departure{
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1234",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "B12A",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1235",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1236",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "A12",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1237",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "12",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1238",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "A12B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "A12A",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "B12B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "B12",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "12B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123B",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "12A",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1239",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123A",
					OperatorCode:          "ANWE",
				},
			},
		}

		sort.Sort(ByServiceNumber(departures.Departures))

		if departures.Departures[0].ServiceNumber != "12" {
			t.Errorf("Expected first departure to have ServiceNumber `%s`; got `%s`", "12", departures.Departures[0].ServiceNumber)
		}

		if departures.Departures[1].ServiceNumber != "12A" {
			t.Errorf("Expected second departure to have ServiceNumber `%s`; got `%s`", "12A", departures.Departures[1].ServiceNumber)
		}

		if departures.Departures[2].ServiceNumber != "12B" {
			t.Errorf("Expected third departure to have ServiceNumber `%s`; got `%s`", "12B", departures.Departures[2].ServiceNumber)
		}

		if departures.Departures[3].ServiceNumber != "123" {
			t.Errorf("Expected fourth departure to have ServiceNumber `%s`; got `%s`", "123", departures.Departures[3].ServiceNumber)
		}

		if departures.Departures[4].ServiceNumber != "123A" {
			t.Errorf("Expected fifth departure to have ServiceNumber `%s`; got `%s`", "123A", departures.Departures[4].ServiceNumber)
		}

		if departures.Departures[5].ServiceNumber != "123B" {
			t.Errorf("Expected sixth departure to have ServiceNumber `%s`; got `%s`", "123B", departures.Departures[5].ServiceNumber)
		}

		if departures.Departures[6].ServiceNumber != "A12" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "A12", departures.Departures[6].ServiceNumber)
		}

		if departures.Departures[7].ServiceNumber != "A12A" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "A12A", departures.Departures[7].ServiceNumber)
		}

		if departures.Departures[8].ServiceNumber != "A12B" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "A12B", departures.Departures[8].ServiceNumber)
		}

		if departures.Departures[9].ServiceNumber != "B12" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "B12", departures.Departures[9].ServiceNumber)
		}

		if departures.Departures[10].ServiceNumber != "B12A" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "B12A", departures.Departures[10].ServiceNumber)
		}

		if departures.Departures[11].ServiceNumber != "B12B" {
			t.Errorf("Expected seventh departure to have ServiceNumber `%s`; got `%s`", "B12B", departures.Departures[11].ServiceNumber)
		}
	})

	t.Run("should sort the departures with the same service number by departure time", func(t *testing.T) {
		now := time.Now()

		expectedDepartureTime1 := test_helpers.AdjustTime(now, "40m").Format(time.RFC3339)
		expectedDepartureTime2 := test_helpers.AdjustTime(now, "10m").Format(time.RFC3339)

		departures := Internal{
			[]Departure{
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1234",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "20m").Format(time.RFC3339),
					ExpectedDepartureTime: &expectedDepartureTime1,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1235",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "30m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1236",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "35m").Format(time.RFC3339),
					ExpectedDepartureTime: &expectedDepartureTime2,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "123",
					OperatorCode:          "ANWE",
				},
			},
		}

		sort.Sort(ByServiceNumber(departures.Departures))

		if departures.Departures[0].JourneyRef != now.Format("2006-01-02")+"_1236" {
			t.Errorf("Expected first departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1236", departures.Departures[0].JourneyRef)
		}

		if departures.Departures[1].JourneyRef != now.Format("2006-01-02")+"_1235" {
			t.Errorf("Expected second departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1235", departures.Departures[1].JourneyRef)
		}

		if departures.Departures[2].JourneyRef != now.Format("2006-01-02")+"_1234" {
			t.Errorf("Expected third departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1234", departures.Departures[2].JourneyRef)
		}
	})

	t.Run("should sort departures with the same service number and departure time by journey reference", func(t *testing.T) {
		now := time.Now()

		departures := Internal{
			[]Departure{
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1235",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "534",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1236",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "534",
					OperatorCode:          "ANWE",
				},
				{
					RecordedAtTime:        now.Format(time.RFC3339),
					JourneyRef:            now.Format("2006-01-02") + "_1234",
					AimedDepartureTime:    test_helpers.AdjustTime(now, "10m").Format(time.RFC3339),
					ExpectedDepartureTime: nil,
					LocationAtcocode:      "1800BNIN0C1",
					DestinationAtcocode:   "1800WA12481",
					ServiceNumber:         "534",
					OperatorCode:          "ANWE",
				},
			},
		}

		sort.Sort(ByServiceNumber(departures.Departures))

		if departures.Departures[0].JourneyRef != now.Format("2006-01-02")+"_1234" {
			t.Errorf("Expected first departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1234", departures.Departures[0].JourneyRef)
		}

		if departures.Departures[1].JourneyRef != now.Format("2006-01-02")+"_1235" {
			t.Errorf("Expected second departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1235", departures.Departures[1].JourneyRef)
		}

		if departures.Departures[2].JourneyRef != now.Format("2006-01-02")+"_1236" {
			t.Errorf("Expected third departure to have JourneyRef `%s`; got `%s`", now.Format("2006-01-02")+"_1236", departures.Departures[2].JourneyRef)
		}
	})
}

func TestDeparture_DepartureTime(t *testing.T) {
	t.Run("expected departure time", func(t *testing.T) {
		expectedDepartureTime := "2019-05-20T11:22:33+01:00"
		dep := Departure{
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		depTime, isRealTime, err := dep.DepartureTime()
		if err != nil {
			t.Error(err)
		}

		wantTime, err := time.Parse(time.RFC3339, expectedDepartureTime)
		if err != nil {
			t.Error(err)
		}

		if !depTime.Equal(wantTime) {
			t.Errorf("got `%s`, want `%s` for departure time", depTime.Format(time.RFC3339), wantTime.Format(time.RFC3339))
		}

		if isRealTime != true {
			t.Errorf("got `%v`, want `%v` for is real time", isRealTime, true)
		}
	})

	t.Run("returns the aimed departure time if the expected departure time is nil", func(t *testing.T) {
		dep := Departure{
			AimedDepartureTime: "2019-05-20T12:34:56+01:00",
		}

		depTime, isRealTime, err := dep.DepartureTime()
		if err != nil {
			t.Error(err)
		}

		wantTime, err := time.Parse(time.RFC3339, "2019-05-20T12:34:56+01:00")
		if err != nil {
			t.Error(err)
		}

		if !depTime.Equal(wantTime) {
			t.Errorf("got `%s`, want `%s` for departure time", depTime.Format(time.RFC3339), wantTime.Format(time.RFC3339))
		}

		if isRealTime != false {
			t.Errorf("got `%v`, want `%v` for is real time", isRealTime, false)
		}
	})
}

func TestDeparture_IsExpired(t *testing.T) {
	t.Run("bus - returns true if the departure has already occurred", func(t *testing.T) {
		expectedDepartureTime := "2019-05-20T11:22:33+01:00"
		dep := Departure{
			JourneyType:           Bus,
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		now, err := time.Parse(time.RFC3339, "2019-05-20T12:00:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != true {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, true)
		}
	})

	t.Run("bus - returns false if the departure is in the future", func(t *testing.T) {
		expectedDepartureTime := "2019-05-20T11:22:33+01:00"
		dep := Departure{
			JourneyType:           Bus,
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		now, err := time.Parse(time.RFC3339, "2019-05-20T11:00:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("bus - returns false if the departure is right now", func(t *testing.T) {
		expectedDepartureTime := "2019-05-20T11:22:33+01:00"
		dep := Departure{
			JourneyType:           Bus,
			AimedDepartureTime:    "2019-05-20T12:34:56+01:00",
			ExpectedDepartureTime: &expectedDepartureTime,
		}

		now, err := time.Parse(time.RFC3339, expectedDepartureTime)
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("bus - works when only aimed departure time is available", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Bus,
			AimedDepartureTime: "2019-05-20T12:34:56+01:00",
		}

		now, err := time.Parse(time.RFC3339, "2019-05-20T13:00:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != true {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, true)
		}
	})

	t.Run("train - returns true if the departure minute has passed and the departure status is `On time`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:25:00+01:00",
			DepartureStatus:    aws.String("On time"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:26:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != true {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, true)
		}
	})

	t.Run("train - returns false if the departure minute is now and the departure status is `On time`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("On time"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:26:59+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns false if the departure minute is in the future and the departure status is `On time`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("On time"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:25:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns true if the departure minute has passed and the departure status is `Cancelled`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:25:00+01:00",
			DepartureStatus:    aws.String("Cancelled"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:26:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != true {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, true)
		}
	})

	t.Run("train - returns false if the departure minute is now and the departure status is `Cancelled`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("Cancelled"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:26:59+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns false if the departure minute is in the future and the departure status is `Cancelled`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("Cancelled"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:25:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns false if the departure minute has passed and the departure status is `Delayed`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:25:00+01:00",
			DepartureStatus:    aws.String("Delayed"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:26:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns false if the departure minute is now and the departure status is `Delayed`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("Delayed"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:26:59+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns false if the departure minute is in the future and the departure status is `Delayed`", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("Delayed"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:25:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns true if the departure status is a time and that time has passed", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:25:00+01:00",
			DepartureStatus:    aws.String("14:26"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:27:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != true {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, true)
		}
	})

	t.Run("train - returns false if the departure status is a time and that time is now", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("14:27"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:27:59+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})

	t.Run("train - returns false if the departure status is a time and that time is in the future", func(t *testing.T) {
		dep := Departure{
			JourneyType:        Train,
			AimedDepartureTime: "2019-07-18T14:26:00+01:00",
			DepartureStatus:    aws.String("14:28"),
		}

		now, err := time.Parse(time.RFC3339, "2019-07-18T14:27:00+01:00")
		if err != nil {
			t.Error(err)
		}

		got := dep.IsExpired(now)

		if got != false {
			t.Errorf("got `%v`, want `%v` for departure is expired", got, false)
		}
	})
}

func Test_GetServiceNumberParts(t *testing.T) {
	t.Run("returns prefix, digits and suffix", func(t *testing.T) {
		d := Departure{
			ServiceNumber: "X42A",
		}

		prefix, digits, suffix := d.GetServiceNumberParts()

		if *prefix != "X" {
			t.Errorf("Expected prefix `%s`, got `%s`", "X", *prefix)
		}

		if *digits != 42 {
			t.Errorf("Expected suffix `%d`, got `%d`", 42, *digits)
		}

		if *suffix != "A" {
			t.Errorf("Expected suffix `%s`, got `%s`", "A", *suffix)
		}
	})

	t.Run("returns empty string for prefix when there isn't a prefix", func(t *testing.T) {
		d := Departure{
			ServiceNumber: "42A",
		}

		prefix, digits, suffix := d.GetServiceNumberParts()

		if prefix != nil {
			t.Errorf("Expected prefix `%v`, got `%v`", nil, &prefix)
		}

		if *digits != 42 {
			t.Errorf("Expected suffix `%d`, got `%d`", 42, &digits)
		}

		if *suffix != "A" {
			t.Errorf("Expected suffix `%s`, got `%s`", "A", *suffix)
		}
	})

	t.Run("returns empty string for suffix when there isn't a suffix", func(t *testing.T) {
		d := Departure{
			ServiceNumber: "X42",
		}

		prefix, digits, suffix := d.GetServiceNumberParts()

		if *prefix != "X" {
			t.Errorf("Expected prefix `%s`, got `%s`", "X", *prefix)
		}

		if *digits != 42 {
			t.Errorf("Expected suffix `%d`, got `%d`", 42, &digits)
		}

		if suffix != nil {
			t.Errorf("Expected suffix `%v`, got `%v`", nil, &suffix)
		}
	})

	t.Run("returns empty string for prefix and suffix when there isn't a prefix or suffix", func(t *testing.T) {
		d := Departure{
			ServiceNumber: "42",
		}

		prefix, digits, suffix := d.GetServiceNumberParts()

		if prefix != nil {
			t.Errorf("Expected prefix `%v`, got `%v`", nil, &prefix)
		}

		if *digits != 42 {
			t.Errorf("Expected suffix `%d`, got `%d`", 42, &digits)
		}

		if suffix != nil {
			t.Errorf("Expected suffix `%v`, got `%v`", nil, &suffix)
		}
	})

	t.Run("returns service numbers that are all letters as the prefix", func(t *testing.T) {
		d := Departure{
			ServiceNumber: "TP",
		}

		prefix, digits, suffix := d.GetServiceNumberParts()

		if *prefix != "TP" {
			t.Errorf("Expected prefix `%s`, got `%s`", "TP", *prefix)
		}

		if digits != nil {
			t.Errorf("Expected suffix `%v`, got `%v`", nil, &digits)
		}

		if suffix != nil {
			t.Errorf("Expected suffix `%v`, got `%v`", nil, &suffix)
		}
	})
}

func TestDeparture_GetStand(t *testing.T) {
	t.Run("returns nil if the departure is for a normal stop", func(t *testing.T) {
		dep := Departure{
			LocationAtcocode: "1800NE43431",
		}

		got := dep.GetStand()

		if got != nil {
			t.Errorf("got `%s`, want `%v` for normal bus stop", *got, nil)
		}
	})

	t.Run("returns nil if the departure is for a bus station", func(t *testing.T) {
		dep := Departure{
			LocationAtcocode: "1800BNIN",
		}

		got := dep.GetStand()

		if got != nil {
			t.Errorf("got `%s`, want `%v` for a bus station", *got, nil)
		}
	})

	t.Run("returns the stand letter if the departure is for a bus station stand at Bolton", func(t *testing.T) {
		dep := Departure{
			LocationAtcocode: "1800BNIN0A1",
		}

		got := dep.GetStand()

		if got == nil {
			t.Errorf("got `%v`, want `%s` for Bolton bus station stand", got, "A")
			return
		}

		if *got != "A" {
			t.Errorf("got `%s`, want `%s` for Bolton bus station stand", *got, "A")
		}
	})

	t.Run("returns the stand letter if the departure is for a bus station stand at Wigan", func(t *testing.T) {
		dep := Departure{
			LocationAtcocode: "180GWNBS0A1",
		}

		got := dep.GetStand()

		if got == nil {
			t.Errorf("got `%v`, want `%s` for Wigan bus station stand", got, "A")
			return
		}

		if *got != "A" {
			t.Errorf("got `%s`, want `%s` for Wigan bus station stand", *got, "A")
		}
	})

	t.Run("returns the stand letter if the departure is for a bus station stand at Altrincham", func(t *testing.T) {
		dep := Departure{
			LocationAtcocode: "1800AMIC0A1",
		}

		got := dep.GetStand()

		if got == nil {
			t.Errorf("got `%v`, want `%s` for Altrincham bus station stand", got, "A")
			return
		}

		if *got != "A" {
			t.Errorf("got `%s`, want `%s` for Altrincham bus station stand", *got, "A")
		}
	})

	t.Run("handles cases where the stand is a number", func(t *testing.T) {
		dep := Departure{
			LocationAtcocode: "1800TCBS011",
		}

		got := dep.GetStand()

		if got == nil {
			t.Errorf("got `%v`, want `%s` for Trafford Centre bus station stand", got, "1")
			return
		}

		if *got != "1" {
			t.Errorf("got `%s`, want `%s` for Trafford Centre bus station stand", *got, "1")
		}
	})

	t.Run("handles cases where the stand has two characters", func(t *testing.T) {
		dep := Departure{
			LocationAtcocode: "1800TCBS141",
		}

		got := dep.GetStand()

		if got == nil {
			t.Errorf("got `%v`, want `%s` for Trafford Centre bus station stand", got, "14")
			return
		}

		if *got != "14" {
			t.Errorf("got `%s`, want `%s` for Trafford Centre bus station stand", *got, "14")
		}
	})
}
