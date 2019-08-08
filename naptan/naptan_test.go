package naptan

import (
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/fortytw2/leaktest"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNaptan_Download(t *testing.T) {
	t.Run("downloads the NaPTANcsv file from the source", func(t *testing.T) {
		defer leaktest.Check(t)()

		stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Helper()
			zipFile, err := ioutil.ReadFile("../test_resources/NaPTANcsv.zip")
			if err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", "attachment; filename='NaPTANcsv.zip'")
			if _, err := w.Write(zipFile); err != nil {
				t.Fatal(err)
			}

		}))
		defer stub.Close()

		n := Naptan{
			Client: stub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			URL: stub.URL,
		}

		zipReader, err := n.Download()
		if err != nil {
			t.Error(err)
			return
		}

		var filesInZip []string

		for _, zf := range zipReader.File {
			filesInZip = append(filesInZip, zf.Name)
		}

		got := len(filesInZip)
		want := 6

		if got != want {
			t.Errorf("got %d files in zip; want %d", got, want)
		}
	})

	t.Run("handles an error from the remote server", func(t *testing.T) {
		defer leaktest.Check(t)()

		stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Helper()
			w.WriteHeader(404)
		}))

		defer stub.Close()

		n := Naptan{
			Client: stub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			URL: stub.URL,
		}

		_, err := n.Download()
		if err == nil {
			t.Error("expected an error")
			return
		}
	})

	t.Run("handles duff data from the remote server", func(t *testing.T) {
		defer leaktest.Check(t)()

		stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Helper()
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "text/html")
			if _, err := w.Write([]byte(string(`<html><head><title>Something went wrong</title></head><body></body></html>`))); err != nil {
				t.Fatal(err)
			}
		}))

		defer stub.Close()

		n := Naptan{
			Client: stub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			URL: stub.URL,
		}

		_, err := n.Download()
		if err == nil {
			t.Error("expected an error")
			return
		}
	})
}
