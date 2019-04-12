package setup

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/solo-io/solo-kit/pkg/utils/stats"

	"github.com/solo-io/glooshot/pkg/gsutil"

	"go.uber.org/zap"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-checkpoint"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
)

type StatsHandler struct {
	ctx context.Context
}

func NewStatsHandler(ctx context.Context) StatsHandler {
	loggingContext := []interface{}{"type", "stats"}
	return StatsHandler{
		ctx: contextutils.WithLoggerValues(ctx, loggingContext...),
	}
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

const (
	START_STATS_SERVER = "START_STATS_SERVER"
)

func Run(ctx context.Context) error {
	start := time.Now()
	checkpoint.CallCheck(version.AppName, version.Version, start)
	flag.Parse()

	if os.Getenv(START_STATS_SERVER) != "" {
		stats.StartStatsServer()
	}

	sh := NewStatsHandler(ctx)
	http.Handle("/", sh)
	go http.ListenAndServe("localhost:8085", nil)

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
