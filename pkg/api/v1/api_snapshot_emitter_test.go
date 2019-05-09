// Code generated by solo-kit. DO NOT EDIT.

// +build solokit

package v1

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/kubeutils"
	"github.com/solo-io/go-utils/log"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	kuberc "github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"
	"github.com/solo-io/solo-kit/test/helpers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// Needed to run tests in GKE
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	// From https://github.com/kubernetes/client-go/blob/53c7adfd0294caa142d961e1f780f74081d5b15f/examples/out-of-cluster-client-configuration/main.go#L31
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var _ = Describe("V1Emitter", func() {
	if os.Getenv("RUN_KUBE_TESTS") != "1" {
		log.Printf("This test creates kubernetes resources and is disabled by default. To enable, set RUN_KUBE_TESTS=1 in your env.")
		return
	}
	var (
		namespace1       string
		namespace2       string
		name1, name2     = "angela" + helpers.RandString(3), "bob" + helpers.RandString(3)
		cfg              *rest.Config
		kube             kubernetes.Interface
		emitter          ApiEmitter
		experimentClient ExperimentClient
	)

	BeforeEach(func() {
		namespace1 = helpers.RandString(8)
		namespace2 = helpers.RandString(8)
		kube = helpers.MustKubeClient()
		err := kubeutils.CreateNamespacesInParallel(kube, namespace1, namespace2)
		Expect(err).NotTo(HaveOccurred())
		cfg, err = kubeutils.GetConfig("", "")
		Expect(err).NotTo(HaveOccurred())
		// Experiment Constructor
		experimentClientFactory := &factory.KubeResourceClientFactory{
			Crd:         ExperimentCrd,
			Cfg:         cfg,
			SharedCache: kuberc.NewKubeCache(context.TODO()),
		}

		experimentClient, err = NewExperimentClient(experimentClientFactory)
		Expect(err).NotTo(HaveOccurred())
		emitter = NewApiEmitter(experimentClient)
	})
	AfterEach(func() {
		err := kubeutils.DeleteNamespacesInParallelBlocking(kube, namespace1, namespace2)
		Expect(err).NotTo(HaveOccurred())
	})
	It("tracks snapshots on changes to any resource", func() {
		ctx := context.Background()
		err := emitter.Register()
		Expect(err).NotTo(HaveOccurred())

		snapshots, errs, err := emitter.Snapshots([]string{namespace1, namespace2}, clients.WatchOpts{
			Ctx:         ctx,
			RefreshRate: time.Second,
		})
		Expect(err).NotTo(HaveOccurred())

		var snap *ApiSnapshot

		/*
			Experiment
		*/

		assertSnapshotExperiments := func(expectExperiments ExperimentList, unexpectExperiments ExperimentList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectExperiments {
						if _, err := snap.Experiments.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectExperiments {
						if _, err := snap.Experiments.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := experimentClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := experimentClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		experiment1a, err := experimentClient.Write(NewExperiment(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		experiment1b, err := experimentClient.Write(NewExperiment(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(ExperimentList{experiment1a, experiment1b}, nil)
		experiment2a, err := experimentClient.Write(NewExperiment(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		experiment2b, err := experimentClient.Write(NewExperiment(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(ExperimentList{experiment1a, experiment1b, experiment2a, experiment2b}, nil)

		err = experimentClient.Delete(experiment2a.GetMetadata().Namespace, experiment2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = experimentClient.Delete(experiment2b.GetMetadata().Namespace, experiment2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(ExperimentList{experiment1a, experiment1b}, ExperimentList{experiment2a, experiment2b})

		err = experimentClient.Delete(experiment1a.GetMetadata().Namespace, experiment1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = experimentClient.Delete(experiment1b.GetMetadata().Namespace, experiment1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(nil, ExperimentList{experiment1a, experiment1b, experiment2a, experiment2b})
	})
	It("tracks snapshots on changes to any resource using AllNamespace", func() {
		ctx := context.Background()
		err := emitter.Register()
		Expect(err).NotTo(HaveOccurred())

		snapshots, errs, err := emitter.Snapshots([]string{""}, clients.WatchOpts{
			Ctx:         ctx,
			RefreshRate: time.Second,
		})
		Expect(err).NotTo(HaveOccurred())

		var snap *ApiSnapshot

		/*
			Experiment
		*/

		assertSnapshotExperiments := func(expectExperiments ExperimentList, unexpectExperiments ExperimentList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectExperiments {
						if _, err := snap.Experiments.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectExperiments {
						if _, err := snap.Experiments.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := experimentClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := experimentClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		experiment1a, err := experimentClient.Write(NewExperiment(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		experiment1b, err := experimentClient.Write(NewExperiment(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(ExperimentList{experiment1a, experiment1b}, nil)
		experiment2a, err := experimentClient.Write(NewExperiment(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		experiment2b, err := experimentClient.Write(NewExperiment(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(ExperimentList{experiment1a, experiment1b, experiment2a, experiment2b}, nil)

		err = experimentClient.Delete(experiment2a.GetMetadata().Namespace, experiment2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = experimentClient.Delete(experiment2b.GetMetadata().Namespace, experiment2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(ExperimentList{experiment1a, experiment1b}, ExperimentList{experiment2a, experiment2b})

		err = experimentClient.Delete(experiment1a.GetMetadata().Namespace, experiment1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = experimentClient.Delete(experiment1b.GetMetadata().Namespace, experiment1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotExperiments(nil, ExperimentList{experiment1a, experiment1b, experiment2a, experiment2b})
	})
})
