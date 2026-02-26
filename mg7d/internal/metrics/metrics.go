package metrics

import (
	"net/http"
	"sync"

	"github.com/mg7d/mg7d/internal/state"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry holds Prometheus metrics for one instance.
type Registry struct {
	instance string
	gauges   struct {
		fps      prometheus.Gauge
		players  prometheus.Gauge
		chunks   prometheus.Gauge
		entities prometheus.Gauge
		zombies  prometheus.Gauge
		heapMB   prometheus.Gauge
		rssMB    prometheus.Gauge
	}
	mu sync.Mutex
}

// NewRegistry creates a registry with instance label for multi-instance support.
func NewRegistry(instance string) *Registry {
	if instance == "" {
		instance = "default"
	}
	r := &Registry{instance: instance}
	return r
}

// RegisterCollectors registers all gauges with the default Prometheus registry.
func (r *Registry) RegisterCollectors() {
	labels := prometheus.Labels{"instance": r.instance}
	r.gauges.fps = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "mg7d_fps",
		Help:        "Current FPS from game log.",
		ConstLabels: labels,
	})
	r.gauges.players = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "mg7d_players",
		Help:        "Current player count.",
		ConstLabels: labels,
	})
	r.gauges.chunks = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "mg7d_chunks",
		Help:        "Current chunk count.",
		ConstLabels: labels,
	})
	r.gauges.entities = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "mg7d_entities",
		Help:        "Total entities.",
		ConstLabels: labels,
	})
	r.gauges.zombies = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "mg7d_zombies",
		Help:        "Zombie count.",
		ConstLabels: labels,
	})
	r.gauges.heapMB = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "mg7d_heap_mb",
		Help:        "Heap size in MB.",
		ConstLabels: labels,
	})
	r.gauges.rssMB = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "mg7d_rss_mb",
		Help:        "RSS in MB.",
		ConstLabels: labels,
	})

	prometheus.MustRegister(
		r.gauges.fps,
		r.gauges.players,
		r.gauges.chunks,
		r.gauges.entities,
		r.gauges.zombies,
		r.gauges.heapMB,
		r.gauges.rssMB,
	)
}

// UpdateFromSnapshot updates all gauges from a snapshot.
func (r *Registry) UpdateFromSnapshot(s state.Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges.fps.Set(s.FPS)
	r.gauges.players.Set(float64(s.Players))
	r.gauges.chunks.Set(float64(s.Chunks))
	r.gauges.entities.Set(float64(s.EntitiesTotal))
	r.gauges.zombies.Set(float64(s.Zombies))
	r.gauges.heapMB.Set(s.HeapMB)
	r.gauges.rssMB.Set(s.RSSMB)
}

// Handler returns the HTTP handler for GET /metrics (Prometheus text format).
func (r *Registry) Handler() http.Handler {
	return promhttp.Handler()
}
