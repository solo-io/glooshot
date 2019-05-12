package setup

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/checker"
	"github.com/solo-io/glooshot/pkg/cli/gsutil"
	"github.com/solo-io/glooshot/pkg/promquery"
	"github.com/solo-io/glooshot/pkg/setup/options"
	"github.com/solo-io/glooshot/pkg/starter"
	"github.com/solo-io/glooshot/pkg/translator"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-checkpoint"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/stats"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/wrapper"
)

func GetOptions() options.Opts {
	var opts options.Opts
	flag.StringVar(&opts.SummaryBindAddr, "summary-bind-addr", ":8085", "bind address for serving "+
		"experiment summaries (debug info)")
	flag.StringVar(&opts.MeshResourceNamespace, "mesh-namespace", "", "optional, namespace "+
		"where Glooshot should look for mesh.supergloo.solo.io CRDs, unless otherwise specified, defaults to all namespaces")
	flag.StringVar(&opts.PrometheusURL, "prometheus-url", "http://prometheus:9090", "required, url on which to reach the prometheus server")
	flag.DurationVar(&opts.PrometheusPollingInterval, "polling-interval", time.Second*5, "optional, "+
		"interval between polls on running prometheus queries for experiments")
	flag.Parse()
	return opts
}

func Run(ctx context.Context, opts options.Opts) error {
	start := time.Now()
	checkpoint.CallCheck(version.AppName, version.Version, start)

	if os.Getenv(START_STATS_SERVER) != "" {
		stats.StartStatsServer()
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", newSummaryHandler(ctx))
		contextutils.LoggerFrom(ctx).Warn(http.ListenAndServe(opts.SummaryBindAddr, mux))
	}()

	expClient, err := gsutil.GetExperimentClient(ctx, true)
	if err != nil {
		return err
	}
	rrClient, err := gsutil.GetRoutingRuleClient(ctx, true)
	if err != nil {
		return err
	}
	meshClient, err := gsutil.GetMeshClient(ctx, true)
	if err != nil {
		return err
	}

	promClient, err := api.NewClient(api.Config{Address: opts.PrometheusURL})
	if err != nil {
		return errors.Wrapf(err, "connecting to prometheus")
	}
	promApi := promv1.NewAPI(promClient)

	if err := checkPrometheusConnection(ctx, promApi); err != nil {
		return err
	}

	promCache := promquery.NewQueryPubSub(ctx, promApi, opts.PrometheusPollingInterval)
	failureChecker := checker.NewChecker(promCache, expClient)

	syncers := []v1.ApiSyncer{
		starter.NewExperimentStarter(expClient),
		translator.NewSyncer(expClient, rrClient, meshClient, opts),
		checker.NewFailureChecker(failureChecker),
	}

	emitter := v1.NewApiSimpleEmitter(wrapper.AggregatedWatchFromClients(wrapper.ClientWatchOpts{
		BaseClient: expClient.BaseClient(),
	}))
	el := v1.NewApiSimpleEventLoop(emitter, syncers...)
	errs, err := el.Run(ctx)

	for err := range errs {
		return errors.Wrapf(err, "error in setup")
	}
	return nil
}

func checkPrometheusConnection(ctx context.Context, promApi promv1.API) error {
	cfg, err := promApi.Config(ctx)
	if err != nil {
		return err
	}
	contextutils.LoggerFrom(ctx).Debugf("prom cfg: \n%v", cfg.YAML)
	return nil
}
