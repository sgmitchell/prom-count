package main

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sgmitchell/prom-count/tracker"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

var (
	totalRcvMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "timeseries_rcv_total",
		Help: "the total number of timeseries received",
	})
	latestSampleTimestampMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "latest_sample_ts",
		Help: "the furthest in the future timestamp for all seen samples",
	})
)

type Receiver struct {
	trackers []tracker.Tracker
	maxTs    int64
}

func NewReceiver(trackers ...tracker.Tracker) *Receiver {
	for _, tmp := range trackers {
		go func(t tracker.Tracker) {
			t.CalculateMetrics()
			ticker := time.NewTicker(15 * time.Second)
			for {
				select {
				case <-ticker.C:
					t.CalculateMetrics()
				}
			}
		}(tmp)
	}
	return &Receiver{
		trackers: trackers,
	}
}

func (rcv *Receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts := getTimeseries(w, r)
	if len(ts) == 0 {
		return
	}
	rcv.updateMaxTimestamp(ts)
	totalRcvMetric.Add(float64(len(ts)))
	latestSampleTimestampMetric.Set(float64(rcv.maxTs))

	if len(rcv.trackers) == 0 {
		return
	}
	batch := toMetrics(ts)
	for _, t := range rcv.trackers {
		t.Observe(batch)
	}
}

func getTimeseries(w http.ResponseWriter, r *http.Request) []prompb.TimeSeries {
	// https://github.com/prometheus/prometheus/blob/release-2.24/documentation/examples/remote_storage/example_write_adapter/server.go
	compressed, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	var req prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}
	return req.GetTimeseries()
}

func toMetrics(timeseries []prompb.TimeSeries) []model.Metric {
	if len(timeseries) == 0 {
		return nil
	}
	batch := make([]model.Metric, len(timeseries))
	for idx, ts := range timeseries {
		ts.GetLabels()
		labels := ts.GetLabels()
		m := make(model.Metric, len(labels))
		for _, l := range labels {
			m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}
		batch[idx] = m
	}
	return batch
}

func (rcv *Receiver) updateMaxTimestamp(timeseries []prompb.TimeSeries) {
	if rcv == nil {
		return
	}
	for _, series := range timeseries {
		for _, sample := range series.GetSamples() {
			if timestamp := sample.GetTimestamp(); timestamp > rcv.maxTs {
				rcv.maxTs = timestamp
			}
		}
	}
}
