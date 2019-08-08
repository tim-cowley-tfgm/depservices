package nationalrail

import (
	"log"
	"testing"
)

func TestGetAtcoCode(t *testing.T) {
	t.Run("should return the ATCO code for a valid CRS code", func(t *testing.T) {
		cases := make(map[string]string)

		cases["ALT"] = "9100ALTRNHM"
		cases["HAZ"] = "9100HAZL"
		cases["MAN"] = "9100MNCRPIC"
		cases["RCD"] = "9100RCHDALE"
		cases["WSR"] = "9100WMOR"

		for crs, want := range cases {
			got, err := GetAtcoCode(crs)

			if err != nil {
				t.Error(err)
				return
			}

			if got != want {
				t.Errorf("got %s, want %s", got, want)
				return
			}
		}
	})

	t.Run("should return an error for an invalid CRS code", func(t *testing.T) {
		got, err := GetAtcoCode("XXX")

		log.Print(got)

		if err == nil {
			t.Error("Expected an error for an invalid CRS code")
		}
	})
}
