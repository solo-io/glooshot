package options

import (
	"os"
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
	DefaultPrometheusPollingInterval = time.Second * 5

	EnvPrometheusURL = "PROMETHEUS_URL"
)

var (
	DefaultPrometheusURL = func() string {
		if promUrl := os.Getenv(EnvPrometheusURL); promUrl != "" {
			return promUrl
		}
		return "http://glooshot-prometheus-server:9090"
	}()
)

func DefaultOpts() Opts {
	return Opts{
		SummaryBindAddr:           DefaultSummaryBindAddr,
		MeshResourceNamespace:     DefaultMeshResourceNamespace,
		PrometheusURL:             DefaultPrometheusURL,
		PrometheusPollingInterval: DefaultPrometheusPollingInterval,
	}
}
