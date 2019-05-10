package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/solo-io/go-utils/stats"

	"github.com/solo-io/glooshot/pkg/cli/gsutil"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"go.uber.org/zap"
)

type Opts struct {
	StreetAddress ind
	Neighbors []string
}

const (
	EnvStreetAddress = "NEIGHBOR_INDEX"
	EnvNeighborServiceList = "NEIGHBOR_SERVICE_LIST"
)

var about = `BLOCK PARTY
This app exists to demonstrate the capabilities of Glooshot.
It expects to have a set of "neighbor" services. All services run the same code with different configs.
According to its configuration, the service can identify itself and its neighbor services.
Upon initialization, each service gets the following from its environment (through the pod spec):
- a list of all services in the neighborhood, self included
- a way to identify itself among the members of the neighborhood
The service provides various metrics that can be used to verify affect of a Glooshot experiment. These include:
- number of messages received from each neighbor
- number of seconds that the service has been running
During runtime, certain properties can be configured.
- to be determined/added as necessary
`

func getOptsFromEnv() (Opts, error) {
	neighborList := strings.Split(os.Getenv(EnvNeighborServiceList), ",")
	if len(neighborList) == 0 {
		return Opts{}, fmt.Errorf("no neighbors found, please pass a comma-separated list of neighbor services through %v env var", EnvNeighborServiceList)
	}
	streetAddress := os.Getenv(EnvStreetAddress)
	if streetAddress == "" {
		return Opts{}, fmt.Errorf("no street address found, please provide an integer between 0 and %v", )
	}
	return Opts{
		StreetAddress:
		Neighbors:neighborList,
	}, nil
}

func main() {
	ctx := context.Background()
	opts, err := getOptsFromEnv()
	if err != nil {
		contextutils.LoggerFrom(ctx).Fatalw("unable to get options from env", zap.Error(err))
	}

	stats.StartStatsServer()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", newSummaryHandler(ctx))
		contextutils.LoggerFrom(ctx).Fatal(http.ListenAndServe(opts.SummaryBindAddr, mux))
	}()

}

type StatsHandler struct {
	ctx context.Context
}

func (d StatsHandler) reportError(err error, w http.ResponseWriter) {
	contextutils.LoggerFrom(d.ctx).Errorw("error getting client", zap.Error(err))
	fmt.Fprintf(w, "error in stats handler %v", err)
}

func (d StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Glooshot stats\n")
	client, err := gsutil.GetExperimentClient(d.ctx, true)
	if err != nil {
		d.reportError(err, w)
		return
	}
	experimentNamespaces := getExperimentNamespaces(d.ctx)
	expCount := 0
	summary := "Experiment Summary\n"
	for _, ns := range experimentNamespaces {
		exps, err := client.List(ns, clients.ListOpts{})
		if err != nil {
			d.reportError(err, w)
			return
		}
		for _, exp := range exps {
			summary += fmt.Sprintf("%v, %v: %v\n",
				exp.Metadata.Namespace,
				exp.Metadata.Name,
				exp.Status.State.String())
			expCount++
		}
	}
	fmt.Fprintf(w, "Count: %v\n", expCount)
	fmt.Fprintf(w, "%v", summary)
}

func getExperimentNamespaces(ctx context.Context) []string {
	contextutils.LoggerFrom(ctx).Errorw("TODO: implement getExperimentNamespaces")
	return []string{"default"}
}

type summaryHandler struct {
	ctx context.Context
}

func newSummaryHandler(ctx context.Context) summaryHandler {
	loggingContext := []interface{}{"type", "stats"}
	return summaryHandler{
		ctx: contextutils.WithLoggerValues(ctx, loggingContext...),
	}
}

func (d summaryHandler) reportError(err error, status int, w http.ResponseWriter) {
	contextutils.LoggerFrom(d.ctx).Errorw("error getting client", zap.Error(err))
	w.WriteHeader(status)
	fmt.Fprint(w, err)
}

type glooshotSummary struct {
	ExperimentCount int    `json:"experiment_count"`
	Summary         string `json:"summary"`
}

func (d summaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client, err := gsutil.GetExperimentClient(d.ctx, true)
	if err != nil {
		d.reportError(err, http.StatusInternalServerError, w)
		return
	}
	experimentNamespaces := getExperimentNamespaces(d.ctx)
	expCount := 0
	summary := "Experiment Summary\n"
	for _, ns := range experimentNamespaces {
		exps, err := client.List(ns, clients.ListOpts{Ctx: r.Context()})
		if err != nil {
			d.reportError(err, http.StatusInternalServerError, w)
			return
		}
		for _, exp := range exps {
			summary += fmt.Sprintf("%v, %v: %v\n",
				exp.Metadata.Namespace,
				exp.Metadata.Name,
				exp.Status.State.String())
			expCount++
		}
	}
	err = json.NewEncoder(w).Encode(glooshotSummary{
		ExperimentCount: expCount,
		Summary:         summary,
	})
	if err != nil {
		d.reportError(err, http.StatusInternalServerError, w)
		return
	}
}
