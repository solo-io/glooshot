package metrics

import (
	"fmt"
	"time"
)

// query templates for well known metrics

func IstioSuccessRateQuery(namespace, name string, interval time.Duration) string {
	return fmt.Sprintf(`
sum(
	rate (
		istio_requests_total{
			response_code!~"5.*",
			destination_service_namespace="%v"
			destination_service_name="%v",
		}[%v]
	)
)
	/
sum(
	rate(
		istio_requests_total{
			destination_service_namespace="%v"
			destination_service_name="%v",
		}[%v]
	)
)
`, namespace, name, interval, namespace, name, interval)
}
