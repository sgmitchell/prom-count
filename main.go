package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sgmitchell/prom-count/tracker/activets"
	"github.com/sgmitchell/prom-count/tracker/labelcount"
	flag "github.com/spf13/pflag"
	"log"
	"net/http"
	"time"
)

var (
	defaultWindows = []time.Duration{5 * time.Minute, 15 * time.Minute, 20 * time.Minute, 30 * time.Minute}
	defaultLabels  = []string{"__name__", "job"}

	port    = flag.Int("port", 8080, "the http port to listen on")
	windows = flag.DurationSlice("windows", defaultWindows, "windows to use for active ts tracker")
	labels  = flag.StringSlice("labels", defaultLabels, "the labels to track for the ")
)

func main() {
	flag.Parse()

	activeTracker := activets.NewTracker(*windows...)
	labelTracker := labelcount.NewTracker(time.Minute, *labels...)

	r := NewReceiver(activeTracker, labelTracker)

	http.Handle("/receive", r)
	http.Handle("/metrics", promhttp.Handler())

	listenAddr := fmt.Sprintf(":%d", *port)
	log.Printf("listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
