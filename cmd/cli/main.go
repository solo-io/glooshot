package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/solo-io/glooshot/pkg/gsutil"

	"github.com/solo-io/glooshot/pkg/version"
	"go.uber.org/zap"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"

	gloov1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1"
	"github.com/solo-io/gloo/projects/gloo/pkg/api/v1/plugins/faultinjection"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/contextutils"
)

func getInitialContext() context.Context {
	loggingContext := []interface{}{"version", version.Version}
	ctx := contextutils.WithLogger(context.Background(), version.CliAppName)
	ctx = contextutils.WithLoggerValues(ctx, loggingContext...)
	return ctx
}

func main() {
	ctx := getInitialContext()
	if err := Run(ctx); err != nil {
		contextutils.LoggerFrom(ctx).Fatalw("msg", zap.Error(err))
	}
}

func Run(ctx context.Context) error {
	var mode string
	var namespace string
	var name string
	flag.StringVar(&mode, "mode", "", "specify a mode (temporary helper cmd, will be removed)")
	flag.StringVar(&namespace, "namespace", "default", "namespace of experiment")
	flag.StringVar(&name, "name", "", "name of experiment")
	flag.Parse()

	if name == "" {
		return fmt.Errorf("must provide an experiment namme")
	}

	client, err := gsutil.GetExperimentClient(ctx, false)
	if err != nil {
		return err
	}

	minute := time.Minute
	delayTime := time.Second
	exp := &v1.Experiment{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
	}
	faults1 := []*v1.ExperimentSpec_InjectedFault{{
		Service: &gloov1.Destination{
			Upstream: core.ResourceRef{
				Name:      "todo",
				Namespace: "default",
			},
		},
		Fault: &faultinjection.RouteFaults{
			Delay: &faultinjection.RouteDelay{
				Percentage: 100,
				FixedDelay: &delayTime,
			},
		},
	}}
	faults2 := []*v1.ExperimentSpec_InjectedFault{{
		Service: &gloov1.Destination{
			Upstream: core.ResourceRef{
				Name:      "todo",
				Namespace: "default",
			},
		},
		Fault: &faultinjection.RouteFaults{
			Abort: &faultinjection.RouteAbort{
				Percentage: 100,
				HttpStatus: 404,
			},
		},
	}}
	stop1 := &v1.StopCondition{
		Duration: &minute,
		Metric: []*v1.MetricThreshold{
			{
				MetricName: "dinner",
				Value:      1800,
			}},
	}
	switch mode {
	case "a":
		exp.Spec = &v1.ExperimentSpec{
			Faults:        faults1,
			StopCondition: stop1,
		}
	case "b":
		exp.Spec = &v1.ExperimentSpec{
			Faults:        faults2,
			StopCondition: stop1,
		}
	default:
	}
	fmt.Println("attempting to write")
	_, err = client.Write(exp, clients.WriteOpts{})
	return err
}
