package main

import (
	"context"
	"log"
	"net/http"

	"github.com/solo-io/glooshot/demos/services/stats_app/pkg/chitchat"
	"github.com/solo-io/glooshot/demos/services/stats_app/pkg/setup"
	"github.com/solo-io/go-utils/stats"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
)

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func Run() error {
	ctx := context.Background()
	opts, err := setup.GetOptsFromEnv()
	if err != nil {
		contextutils.LoggerFrom(ctx).Fatalw("unable to get options from env", zap.Error(err))
	}

	stats.StartStatsServer()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", chitchat.NewChatterHandler(ctx, opts))
		contextutils.LoggerFrom(ctx).Fatal(http.ListenAndServe(opts.BindAddress, mux))
	}()

	chitchat.MakeSmallTalk(opts)
	return nil
}
