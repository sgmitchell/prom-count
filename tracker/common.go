package tracker

import (
	"github.com/prometheus/common/model"
)

const MetricNs = "prom_count"

type Tracker interface {
	Observe(metrics []model.Metric) error
	CalculateMetrics()
}
