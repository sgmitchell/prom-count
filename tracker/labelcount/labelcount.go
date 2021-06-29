//
package labelcount

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sgmitchell/prom-count/tracker"
	"sync"
	"time"
)

const LabelPrefix = "label"

type data struct {
	reducedFp model.Fingerprint
	last      time.Time
}

type Tracker struct {
	ttl           time.Duration                      // how long to keep metrics around for
	labelMappings map[model.LabelName]string         // a map of input label names to output names
	fullToSig     map[model.Fingerprint]data         // map the full input fingerprint to a reduced fingerprint and last seen timestamp
	reduced       map[model.Fingerprint]model.Metric // map of reduced fingerprints to their reduced metric values
	mu            sync.Mutex

	metric *prometheus.GaugeVec // the metric we're exposing
}

var _ tracker.Tracker = (*Tracker)(nil)

func NewTracker(ttl time.Duration, labels ...string) *Tracker {
	// determine the labels for the metric and
	labelNames := make([]string, len(labels))
	labelMappings := map[model.LabelName]string{}
	for i, v := range labels {
		outLabel := fmt.Sprintf("%s_%s", LabelPrefix, v)
		// TODO "__name__" => "label___name__" is kinda gross. Is there a better mapping
		labelNames[i] = outLabel
		labelMappings[model.LabelName(v)] = outLabel
	}

	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "metric_count",
		Help: fmt.Sprintf("the number of metrics observed in the last %s for the given %s_ labels", ttl, LabelPrefix),
	}, labelNames)

	return &Tracker{
		ttl:           ttl,
		metric:        metric,
		labelMappings: labelMappings,
		fullToSig:     map[model.Fingerprint]data{},
		reduced:       map[model.Fingerprint]model.Metric{},
	}
}

func (t *Tracker) Observe(metrics []model.Metric) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for _, m := range metrics {
		fp := m.FastFingerprint()
		// if we've seen this fingerprint before, just update the timestamp to keep it from being GCed
		if val, seen := t.fullToSig[fp]; seen {
			val.last = now
			t.fullToSig[fp] = val
			continue
		}

		// the fingerprint is new so create a reduced metric containing only the labels we care about to save space
		reduced := model.Metric{}
		for k := range t.labelMappings {
			reduced[k] = m[k]
		}
		reducedFp := reduced.FastFingerprint()
		t.fullToSig[fp] = data{reducedFp: reducedFp, last: now}
		t.reduced[reducedFp] = reduced
	}
	return nil
}

func (t *Tracker) CalculateMetrics() {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-1 * t.ttl)
	counts := map[model.Fingerprint]float64{}

	for fullSig, val := range t.fullToSig {
		if cutoff.After(val.last) {
			delete(t.fullToSig, fullSig)
			continue
		}
		counts[val.reducedFp]++
	}

	for shortSig, metric := range t.reduced {
		count, found := counts[shortSig]
		// always set labels for everything in reduced, even if it is not seen, so the values go to 0
		lvs := prometheus.Labels{}
		for inLabel, outLabel := range t.labelMappings {
			val := metric[inLabel]
			lvs[outLabel] = string(val)
		}
		t.metric.With(lvs).Set(count)

		// remove unseen metrics
		if !found {
			delete(t.reduced, shortSig)
		}
	}
}
