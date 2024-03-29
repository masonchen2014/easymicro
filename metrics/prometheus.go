package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type LabelValues []string

// With validates the input, and returns a new aggregate labelValues.
func (lvs LabelValues) With(labelValues ...string) LabelValues {
	if len(labelValues)%2 != 0 {
		labelValues = append(labelValues, "unknown")
	}
	return append(lvs, labelValues...)
}

// Counter implements Counter, via a Prometheus CounterVec.
type PrometheusCounter struct {
	cv  *prometheus.CounterVec
	lvs LabelValues
}

// NewCounterFrom constructs and registers a Prometheus CounterVec,
// and returns a usable Counter object.
func NewCounterFrom(opts prometheus.CounterOpts, labelNames []string) *PrometheusCounter {
	cv := prometheus.NewCounterVec(opts, labelNames)
	prometheus.MustRegister(cv)
	return NewCounter(cv)
}

// NewCounter wraps the CounterVec and returns a usable Counter object.
func NewCounter(cv *prometheus.CounterVec) *PrometheusCounter {
	return &PrometheusCounter{
		cv: cv,
	}
}

// With implements Counter.
func (c *PrometheusCounter) With(labelValues ...string) *PrometheusCounter {
	return &PrometheusCounter{
		cv:  c.cv,
		lvs: c.lvs.With(labelValues...),
	}
}

// Add implements Counter.
func (c *PrometheusCounter) Add(delta float64) {
	c.cv.With(makeLabels(c.lvs...)).Add(delta)
}

// Gauge implements Gauge, via a Prometheus GaugeVec.
type PrometheusGauge struct {
	gv  *prometheus.GaugeVec
	lvs LabelValues
}

// NewGaugeFrom construts and registers a Prometheus GaugeVec,
// and returns a usable Gauge object.
func NewGaugeFrom(opts prometheus.GaugeOpts, labelNames []string) *PrometheusGauge {
	gv := prometheus.NewGaugeVec(opts, labelNames)
	prometheus.MustRegister(gv)
	return NewGauge(gv)
}

// NewGauge wraps the GaugeVec and returns a usable Gauge object.
func NewGauge(gv *prometheus.GaugeVec) *PrometheusGauge {
	return &PrometheusGauge{
		gv: gv,
	}
}

// With implements Gauge.
func (g *PrometheusGauge) With(labelValues ...string) *PrometheusGauge {
	return &PrometheusGauge{
		gv:  g.gv,
		lvs: g.lvs.With(labelValues...),
	}
}

// Set implements Gauge.
func (g *PrometheusGauge) Set(value float64) {
	g.gv.With(makeLabels(g.lvs...)).Set(value)
}

// Add is supported by Prometheus GaugeVecs.
func (g *PrometheusGauge) Add(delta float64) {
	g.gv.With(makeLabels(g.lvs...)).Add(delta)
}

// Summary implements Histogram, via a Prometheus SummaryVec. The difference
// between a Summary and a Histogram is that Summaries don't require predefined
// quantile buckets, but cannot be statistically aggregated.
type PrometheusSummary struct {
	sv  *prometheus.SummaryVec
	lvs LabelValues
}

// NewSummaryFrom constructs and registers a Prometheus SummaryVec,
// and returns a usable Summary object.
func NewSummaryFrom(opts prometheus.SummaryOpts, labelNames []string) *PrometheusSummary {
	sv := prometheus.NewSummaryVec(opts, labelNames)
	prometheus.MustRegister(sv)
	return NewSummary(sv)
}

// NewSummary wraps the SummaryVec and returns a usable Summary object.
func NewSummary(sv *prometheus.SummaryVec) *PrometheusSummary {
	return &PrometheusSummary{
		sv: sv,
	}
}

// With implements Histogram.
func (s *PrometheusSummary) With(labelValues ...string) *PrometheusSummary {
	return &PrometheusSummary{
		sv:  s.sv,
		lvs: s.lvs.With(labelValues...),
	}
}

// Observe implements Histogram.
func (s *PrometheusSummary) Observe(value float64) {
	s.sv.With(makeLabels(s.lvs...)).Observe(value)
}

// Histogram implements Histogram via a Prometheus HistogramVec. The difference
// between a Histogram and a Summary is that Histograms require predefined
// quantile buckets, and can be statistically aggregated.
type PrometheusHistogram struct {
	hv  *prometheus.HistogramVec
	lvs LabelValues
}

// NewHistogramFrom constructs and registers a Prometheus HistogramVec,
// and returns a usable Histogram object.
func NewHistogramFrom(opts prometheus.HistogramOpts, labelNames []string) *PrometheusHistogram {
	hv := prometheus.NewHistogramVec(opts, labelNames)
	prometheus.MustRegister(hv)
	return NewHistogram(hv)
}

// NewHistogram wraps the HistogramVec and returns a usable Histogram object.
func NewHistogram(hv *prometheus.HistogramVec) *PrometheusHistogram {
	return &PrometheusHistogram{
		hv: hv,
	}
}

// With implements Histogram.
func (h *PrometheusHistogram) With(labelValues ...string) *PrometheusHistogram {
	return &PrometheusHistogram{
		hv:  h.hv,
		lvs: h.lvs.With(labelValues...),
	}
}

// Observe implements Histogram.
func (h *PrometheusHistogram) Observe(value float64) {
	h.hv.With(makeLabels(h.lvs...)).Observe(value)
}

func makeLabels(labelValues ...string) prometheus.Labels {
	labels := prometheus.Labels{}
	for i := 0; i < len(labelValues); i += 2 {
		labels[labelValues[i]] = labelValues[i+1]
	}
	return labels
}
