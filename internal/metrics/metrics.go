package metrics

import (
	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func InstallRoute(r *gin.Engine) {
	p := ginprom.New(
		ginprom.Engine(r),
		ginprom.Subsystem("gin"),
		ginprom.Path("/metrics"),
	)
	r.Use(p.Instrument())
}

const (
	ns = "emage"
)

var (
	UploadsRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: ns,
		Subsystem: "http",
		Name:      "uploads_running",
		Help:      "Number of images being currently uploaded",
	})
	RequestCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of requested images",
	})
)
