package main

import (
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/naptan"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/alicebob/miniredis"
	"github.com/fortytw2/leaktest"
	"github.com/gomodule/redigo/redis"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createNaptanSourceStub(t *testing.T) *httptest.Server {
	t.Helper()

	naptanSourceHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		zip, err := ioutil.ReadFile("../test_resources/NaPTANcsv.zip")
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename='NaPTANcsv.zip'")
		if _, err := w.Write(zip); err != nil {
			t.Fatal(err)
		}
	}

	return httptest.NewServer(http.HandlerFunc(naptanSourceHandler))
}

func TestStopsInArea_Handler(t *testing.T) {
	defer leaktest.Check(t)()

	t.Run("happy path", func(t *testing.T) {
		naptanStub := createNaptanSourceStub(t)
		defer naptanStub.Close()

		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		sia := StopsInArea{
			Filename: "StopsInArea.csv",
			Logger:   logger,
			NaptanClient: &naptan.Naptan{
				Client: naptanStub.Client(),
				Logger: logger,
				URL:    naptanStub.URL,
			},
			RedisPipeline: &repository.RedisPipeline{
				FlushAfter: 3,
				Pool: repository.NewRedisPool([]repository.RedisPoolOption{
					repository.RedisPoolDial(func() (redis.Conn, error) {
						return redis.Dial("tcp", s.Addr())
					}),
					repository.RedisPoolMaxActive(10),
				}...),
			},
		}

		if err := sia.Handler(); err != nil {
			t.Error(err)
			return
		}

		s.CheckGet(t, "1800ESBS0A1", "180GESBS")
		s.CheckGet(t, "1800HEBS0D1", "180GHEBS")
		s.CheckGet(t, "1800LHBS0H1", "180GLHBS")
		s.CheckGet(t, "1800MABS001", "180GMABS")
		s.CheckGet(t, "1800MNBS0E1", "180GMNBS")
		s.CheckGet(t, "1800SHIC0B1", "180GSHIC")
		s.CheckGet(t, "1800TCBS031", "180GTCBS")
		s.CheckGet(t, "1800WEIC0B1", "180GWEIC")
	})

	t.Run("file not found in zip", func(t *testing.T) {
		naptanStub := createNaptanSourceStub(t)
		defer naptanStub.Close()

		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		sia := StopsInArea{
			Filename: "NonExistant.csv",
			Logger:   logger,
			NaptanClient: &naptan.Naptan{
				Client: naptanStub.Client(),
				Logger: logger,
				URL:    naptanStub.URL,
			},
			RedisPipeline: &repository.RedisPipeline{
				FlushAfter: 3,
				Pool: repository.NewRedisPool([]repository.RedisPoolOption{
					repository.RedisPoolDial(func() (redis.Conn, error) {
						return redis.Dial("tcp", s.Addr())
					}),
					repository.RedisPoolMaxActive(10),
				}...),
			},
		}

		if err := sia.Handler(); err == nil {
			t.Error("an error should have occurred")
			return
		}
	})

	t.Run("handles Redis connection failure", func(t *testing.T) {
		naptanStub := createNaptanSourceStub(t)
		defer naptanStub.Close()

		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		sia := StopsInArea{
			Filename: "StopsInArea.csv",
			Logger:   logger,
			NaptanClient: &naptan.Naptan{
				Client: naptanStub.Client(),
				Logger: logger,
				URL:    naptanStub.URL,
			},
			RedisPipeline: &repository.RedisPipeline{
				FlushAfter: 3,
				Pool: repository.NewRedisPool([]repository.RedisPoolOption{
					repository.RedisPoolDial(func() (redis.Conn, error) {
						return redis.Dial("tcp", "")
					}),
					repository.RedisPoolMaxActive(10),
				}...),
			},
		}

		if err := sia.Handler(); err == nil {
			t.Error("an error should have occurred")
			return
		}
	})
}
