package controller

import (
	"context"

	"github.com/solo-io/glooshot/pkg/cli/gsutil"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Controller struct {
	clients gsutil.ClientCache
}

func NewController(ctx context.Context, registerCrds bool, initError func(error)) Controller {
	return Controller{
		clients: gsutil.NewClientCache(ctx, registerCrds, initError),
	}
}
func From(clientCache gsutil.ClientCache) Controller {
	return Controller{
		clients: clientCache,
	}
}

func (op Controller) DeleteExperiment(namespace, name string) error {
	return op.clients.ExpClient().Delete(namespace, name, clients.DeleteOpts{Ctx: op.clients.Ctx()})
}

func (op Controller) DeleteExperiments(namespace string) error {

	exps, err := op.clients.ExpClient().List(namespace, clients.ListOpts{})
	if err != nil {
		return err
	}
	for _, exp := range exps {
		if err := op.DeleteExperiment(namespace, exp.Metadata.Name); err != nil {
			return err
		}
	}
	return nil
}

func (op Controller) DeleteAllExperiments() error {
	namespaces, err := op.clients.KubeClient().CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ns := range namespaces.Items {
		if err := op.DeleteExperiments(ns.Name); err != nil {
			return err
		}
	}
	return nil
}
