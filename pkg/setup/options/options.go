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
	DefaultPrometheusURL             = "http://prometheus:9090"
	DefaultPrometheusPollingInterval = time.Second * 5
)

func DefaultOpts() Opts {
	return Opts{
		SummaryBindAddr:           DefaultSummaryBindAddr,
		MeshResourceNamespace:     DefaultMeshResourceNamespace,
		PrometheusURL:             DefaultPrometheusURL,
		PrometheusPollingInterval: DefaultPrometheusPollingInterval,
	}
}
