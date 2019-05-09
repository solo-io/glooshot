package setup

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/solo-io/glooshot/pkg/translator"

	"github.com/solo-io/go-utils/stats"

	"github.com/solo-io/glooshot/pkg/gsutil"

	"go.uber.org/zap"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-checkpoint"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
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

	expClient, err := gsutil.GetExperimentClient(ctx, true)
	if err != nil {
		return err
	}
	rrClient, err := gsutil.GetRoutingRuleClient(ctx, true)
	if err != nil {
		return err
	}
	syncer := translator.NewSyncer(expClient, rrClient)
	el := v1.NewApiEventLoop(v1.NewApiEmitter(expClient), syncer)
	errs, err := el.Run([]string{}, clients.WatchOpts{
		Ctx:         ctx,
		RefreshRate: time.Second,
	})

	for err := range errs {
		contextutils.LoggerFrom(ctx).Fatalw("error in setup", zap.Error(err))
	}
	return nil
}
