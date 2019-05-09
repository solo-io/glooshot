package gsutil

import (
	"context"

	sgv1 "github.com/solo-io/supergloo/pkg/api/v1"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/kubeutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"
)

func GetExperimentClient(ctx context.Context, skipCrdCreation bool) (v1.ExperimentClient, error) {
	cfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return nil, err
	}
	cache := kube.NewKubeCache(ctx)
	rcFactory := &factory.KubeResourceClientFactory{
		Crd:             v1.ExperimentCrd,
		Cfg:             cfg,
		SharedCache:     cache,
		SkipCrdCreation: skipCrdCreation,
	}
	client, err := v1.NewExperimentClient(rcFactory)
	client.Register()
	return client, nil
}
func GetRoutingRuleClient(ctx context.Context, skipCrdCreation bool) (sgv1.RoutingRuleClient, error) {
	cfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return nil, err
	}
	cache := kube.NewKubeCache(ctx)
	rcFactory := &factory.KubeResourceClientFactory{
		Crd:             v1.ExperimentCrd,
		Cfg:             cfg,
		SharedCache:     cache,
		SkipCrdCreation: skipCrdCreation,
	}
	client, err := sgv1.NewRoutingRuleClient(rcFactory)
	client.Register()
	return client, nil
}

func GetKubeClient() (*kubernetes.Clientset, error) {
	restCfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return &kubernetes.Clientset{}, errors.Wrapf(err, "no Kubernetes context config found; please double check your Kubernetes environment")
	}
	kubeClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return &kubernetes.Clientset{}, errors.Wrapf(err, "error connecting to current Kubernetes Context Host %s; please double check your Kubernetes environment", restCfg.Host)
	}
	return kubeClient, nil
}

// ClientCache provides a lazy-loaded, cached set of clients
// Benefits over one-off clients:
// - CLI's using this will function without needing a valid kubeconfig
// - Clients will not be recreated during the life of the process
type ClientCache struct {
	// user-provided
	ctx          context.Context
	check        func(error)
	registerCrds bool

	// internal cache
	kubeClient *kubernetes.Clientset
	expClient  *v1.ExperimentClient
}

func NewClientCache(ctx context.Context, registerCrds bool, handleError func(error)) ClientCache {
	return ClientCache{
		ctx:          ctx,
		registerCrds: registerCrds,
		check:        handleError,
	}
}

func (cc *ClientCache) KubeClient() *kubernetes.Clientset {
	if cc.kubeClient == nil {
		var err error
		cc.kubeClient, err = GetKubeClient()
		cc.check(err)
	}
	return cc.kubeClient
}

func (cc *ClientCache) ExpClient() v1.ExperimentClient {
	if cc.expClient == nil {
		expClient, err := GetExperimentClient(cc.ctx, !cc.registerCrds)
		cc.check(err)
		cc.expClient = &expClient
	}
	return *cc.expClient
}

func (cc *ClientCache) Ctx() context.Context {
	return cc.ctx
}
