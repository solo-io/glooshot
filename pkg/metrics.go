package pkg

import (
	"strings"
	"time"
)

func analyseCanary(r *flaggerv1.Canary) bool {
	// run external checks
	for _, webhook := range r.Spec.CanaryAnalysis.Webhooks {
		if webhook.Type == "" || webhook.Type == flaggerv1.RolloutHook {
			err := CallWebhook(r.Name, r.Namespace, flaggerv1.CanaryProgressing, webhook)
			if err != nil {
				c.recordEventWarningf(r, "Halt %s.%s advancement external check %s failed %v",
					r.Name, r.Namespace, webhook.Name, err)
				return false
			}
		}
	}

	// run metrics checks
	for _, metric := range r.Spec.CanaryAnalysis.Metrics {
		if metric.Interval == "" {
			metric.Interval = r.GetMetricInterval()
		}

		// App Mesh checks
		if c.meshProvider == "appmesh" {
			if metric.Name == "request-success-rate" || metric.Name == "envoy_cluster_upstream_rq" {
				val, err := c.observer.GetEnvoySuccessRate(r.Spec.TargetRef.Name, r.Namespace, metric.Name, metric.Interval)
				if err != nil {
					if strings.Contains(err.Error(), "no values found") {
						c.recordEventWarningf(r, "Halt advancement no values found for metric %s probably %s.%s is not receiving traffic",
							metric.Name, r.Spec.TargetRef.Name, r.Namespace)
					} else {
						c.recordEventErrorf(r, "Metrics server %s query failed: %v", c.observer.GetMetricsServer(), err)
					}
					return false
				}
				if float64(metric.Threshold) > val {
					c.recordEventWarningf(r, "Halt %s.%s advancement success rate %.2f%% < %v%%",
						r.Name, r.Namespace, val, metric.Threshold)
					return false
				}
			}

			if metric.Name == "request-duration" || metric.Name == "envoy_cluster_upstream_rq_time_bucket" {
				val, err := c.observer.GetEnvoyRequestDuration(r.Spec.TargetRef.Name, r.Namespace, metric.Name, metric.Interval)
				if err != nil {
					c.recordEventErrorf(r, "Metrics server %s query failed: %v", c.observer.GetMetricsServer(), err)
					return false
				}
				t := time.Duration(metric.Threshold) * time.Millisecond
				if val > t {
					c.recordEventWarningf(r, "Halt %s.%s advancement request duration %v > %v",
						r.Name, r.Namespace, val, t)
					return false
				}
			}
		}

		// Istio checks
		if c.meshProvider == "istio" {
			if metric.Name == "request-success-rate" || metric.Name == "istio_requests_total" {
				val, err := c.observer.GetIstioSuccessRate(r.Spec.TargetRef.Name, r.Namespace, metric.Name, metric.Interval)
				if err != nil {
					if strings.Contains(err.Error(), "no values found") {
						c.recordEventWarningf(r, "Halt advancement no values found for metric %s probably %s.%s is not receiving traffic",
							metric.Name, r.Spec.TargetRef.Name, r.Namespace)
					} else {
						c.recordEventErrorf(r, "Metrics server %s query failed: %v", c.observer.GetMetricsServer(), err)
					}
					return false
				}
				if float64(metric.Threshold) > val {
					c.recordEventWarningf(r, "Halt %s.%s advancement success rate %.2f%% < %v%%",
						r.Name, r.Namespace, val, metric.Threshold)
					return false
				}
			}

			if metric.Name == "request-duration" || metric.Name == "istio_request_duration_seconds_bucket" {
				val, err := c.observer.GetIstioRequestDuration(r.Spec.TargetRef.Name, r.Namespace, metric.Name, metric.Interval)
				if err != nil {
					c.recordEventErrorf(r, "Metrics server %s query failed: %v", c.observer.GetMetricsServer(), err)
					return false
				}
				t := time.Duration(metric.Threshold) * time.Millisecond
				if val > t {
					c.recordEventWarningf(r, "Halt %s.%s advancement request duration %v > %v",
						r.Name, r.Namespace, val, t)
					return false
				}
			}
		}

		// custom checks
		if metric.Query != "" {
			val, err := c.observer.GetScalar(metric.Query)
			if err != nil {
				if strings.Contains(err.Error(), "no values found") {
					c.recordEventWarningf(r, "Halt advancement no values found for metric %s probably %s.%s is not receiving traffic",
						metric.Name, r.Spec.TargetRef.Name, r.Namespace)
				} else {
					c.recordEventErrorf(r, "Metrics server %s query failed: %v", c.observer.GetMetricsServer(), err)
				}
				return false
			}
			if val > float64(metric.Threshold) {
				c.recordEventWarningf(r, "Halt %s.%s advancement %s %.2f > %v",
					r.Name, r.Namespace, metric.Name, val, metric.Threshold)
				return false
			}
		}
	}

	return true
}
