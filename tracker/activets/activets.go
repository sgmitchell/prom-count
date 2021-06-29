//
package activets

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sgmitchell/prom-count/tracker"
	"sync"
	"time"
)

type Tracker struct {
	windows []time.Duration
	metric  *prometheus.GaugeVec

	seen map[model.Fingerprint]time.Time
	mu   sync.Mutex
}

var _ tracker.Tracker = (*Tracker)(nil)

func NewTracker(windows ...time.Duration) *Tracker {
	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: tracker.MetricNs,
		Name:      "active_metrics",
		Help:      "the number of active metrics over the given window",
	}, []string{"window"})

	prometheus.MustRegister(metric)

	return &Tracker{
		windows: windows,
		seen:    map[model.Fingerprint]time.Time{},
		metric:  metric,
	}
}

func (t *Tracker) Observe(metrics []model.Metric) error {
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, m := range metrics {
		t.seen[m.FastFingerprint()] = now
	}
	return nil
}

func (t *Tracker) CalculateMetrics() {
	if t.metric == nil || len(t.windows) == 0 {
		return
	}
	now := time.Now()
	seen := map[time.Duration]float64{}

	t.mu.Lock()
	for k, last := range t.seen {
		for i, dur := range t.windows {
			if now.Sub(last) <= dur {
				seen[dur] += 1
			} else if i == len(t.windows)-1 {
				delete(t.seen, k)
			}
		}
	}
	t.mu.Unlock()

	for _, k := range t.windows {
		v := seen[k]
		t.metric.WithLabelValues(k.String()).Set(v)
	}
}
