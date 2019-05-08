package setup

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"


	"github.com/solo-io/glooshot/pkg/gsutil"

	"go.uber.org/zap"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-checkpoint"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
)

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

func getExperimentNamespaces(ctx context.Context) []string {
	contextutils.LoggerFrom(ctx).Errorw("TODO: implement getExperimentNamespaces")
	return []string{"default"}
}

const (
	START_STATS_SERVER = "START_STATS_SERVER"
)

type Opts struct {
	SummaryBindAddr string
}

func Run(ctx context.Context) error {
	start := time.Now()
	checkpoint.CallCheck(version.AppName, version.Version, start)

	var opts Opts
	flag.StringVar(&opts.SummaryBindAddr, "summary-bind-addr", ":8085", "bind address for serving "+
		"experiment summaries (debug info)")
	flag.Parse()

	if os.Getenv(START_STATS_SERVER) != "" {
		stats.StartStatsServer()
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", newSummaryHandler(ctx))
		contextutils.LoggerFrom(ctx).Fatal(http.ListenAndServe(opts.SummaryBindAddr, mux))
	}()

	client, err := gsutil.GetExperimentClient(ctx, true)
	if err != nil {
		return err
	}
	syncer := NewSyncer(client)
	el := v1.NewApiEventLoop(v1.NewApiEmitter(client), syncer)
	errs, err := el.Run([]string{}, clients.WatchOpts{
		Ctx:         ctx,
		RefreshRate: time.Second,
	})

	for err := range errs {
		contextutils.LoggerFrom(ctx).Fatalw("error in setup", zap.Error(err))
	}
	return nil
}
