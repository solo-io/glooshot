package options

import (
	"time"
)

type Opts struct {
	SummaryBindAddr           string
	MeshResourceNamespace     string
	PrometheusAddr            string
	PrometheusPollingInterval time.Duration
}
