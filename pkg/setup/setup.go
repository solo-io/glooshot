package setup

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/solo-io/glooshot/pkg/setup/options"

	"github.com/solo-io/glooshot/pkg/translator"

	"github.com/solo-io/go-utils/stats"

	"github.com/solo-io/glooshot/pkg/cli/gsutil"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-checkpoint"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
)

func Run(ctx context.Context) error {
	start := time.Now()
	checkpoint.CallCheck(version.AppName, version.Version, start)

	var opts options.Opts
	flag.StringVar(&opts.SummaryBindAddr, "summary-bind-addr", ":8085", "bind address for serving "+
		"experiment summaries (debug info)")
	flag.StringVar(&opts.MeshResourceNamespace, "mesh-namespace", "", "optional, namespace "+
		"where Glooshot should look for mesh.supergloo.solo.io CRDs, unless otherwise specified, defaults to all namespaces")
	flag.Parse()

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
	syncer := translator.NewSyncer(expClient, rrClient, meshClient, opts)
	el := v1.NewApiEventLoop(v1.NewApiEmitter(expClient), syncer)
	errs, err := el.Run([]string{}, clients.WatchOpts{
		Ctx:         ctx,
		RefreshRate: time.Second,
	})

	for err := range errs {
		return errors.Wrapf(err, "error in setup")
	}
	return nil
}
