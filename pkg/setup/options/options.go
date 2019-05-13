package options

import (
	"time"
)

type Opts struct {
	SummaryBindAddr           string
	MeshResourceNamespace     string
	PrometheusURL             string
	PrometheusPollingInterval time.Duration
}

const (
	DefaultSummaryBindAddr           = ":8085"
	DefaultMeshResourceNamespace     = ""
	DefaultPrometheusURL             = "http://glooshot-prometheus-server:9090"
	DefaultPrometheusPollingInterval = time.Second * 5

	EnvPrometheusURL = "PROMETHEUS_URL"
)

func DefaultOpts() Opts {
	return Opts{
		SummaryBindAddr:           DefaultSummaryBindAddr,
		MeshResourceNamespace:     DefaultMeshResourceNamespace,
		PrometheusURL:             DefaultPrometheusURL,
		PrometheusPollingInterval: DefaultPrometheusPollingInterval,
	}
}
