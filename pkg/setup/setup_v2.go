package setup

import (
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/wrapper"
)

func setup() error {
	experimentClient, err := v1.NewExperimentClient()
	if err != nil {
		return err
	}
	watch := wrapper.AggregatedWatchFromClients(wrapper.ClientWatchOpts{
		BaseClient: experimentClient.BaseClient(),
	})

	emitter := v1.NewApiSimpleEmitter(watch)
	eventLoop := v1.NewApiSimpleEventLoop(emitter, syncer())
	return eventLoop.Run(ctx)
}

func syncers() {

}
