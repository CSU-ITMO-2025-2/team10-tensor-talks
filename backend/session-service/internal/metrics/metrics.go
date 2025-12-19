package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// BusinessSessionsCreatedTotal счетчик созданных сессий
	BusinessSessionsCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tensortalks_business_sessions_created_total",
			Help: "Total number of sessions created",
		},
		[]string{"service", "status"},
	)
)

func init() {
	prometheus.MustRegister(BusinessSessionsCreatedTotal)
}
