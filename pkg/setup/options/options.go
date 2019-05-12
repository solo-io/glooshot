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
