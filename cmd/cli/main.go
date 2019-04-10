package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/solo-io/glooshot/pkg/setup"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-utils/contextutils"
)

func main() {

	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func Run() error {
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

	ctx := contextutils.WithLogger(context.Background(), version.AppName)
	client, err := setup.GetExperimentClient(ctx, true)
	if err != nil {
		return err
	}

	minute := time.Minute
	exp := &v1.Experiment{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
	}
	switch mode {
	case "a":
		exp.Spec = &v1.ExperimentSpec{
			StopCondition: &v1.StopCondition{
				Duration: &minute,
				Metric: []*v1.MetricThreshold{
					{
						MetricName: "frank",
						Value:      9000,
					}},
			},
		}
	case "b":
		exp.Spec = &v1.ExperimentSpec{
			StopCondition: &v1.StopCondition{
				Duration: &minute,
				Metric: []*v1.MetricThreshold{
					{
						MetricName: "cores",
						Value:      10,
					}},
			},
		}
	default:
	}
	fmt.Println("attempting to write")
	_, err = client.Write(exp, clients.WriteOpts{})
	return err
}
