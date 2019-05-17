// Code generated by solo-kit. DO NOT EDIT.

package v1

import (
	"sort"

	"github.com/solo-io/go-utils/hashutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/kube/crd"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"github.com/solo-io/solo-kit/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewReport(namespace, name string) *Report {
	report := &Report{}
	report.SetMetadata(core.Metadata{
		Name:      name,
		Namespace: namespace,
	})
	return report
}

func (r *Report) SetMetadata(meta core.Metadata) {
	r.Metadata = meta
}

func (r *Report) Hash() uint64 {
	metaCopy := r.GetMetadata()
	metaCopy.ResourceVersion = ""
	return hashutils.HashAll(
		metaCopy,
		r.Experiment,
		r.FailureConditionHistory,
	)
}

type ReportList []*Report

// namespace is optional, if left empty, names can collide if the list contains more than one with the same name
func (list ReportList) Find(namespace, name string) (*Report, error) {
	for _, report := range list {
		if report.GetMetadata().Name == name {
			if namespace == "" || report.GetMetadata().Namespace == namespace {
				return report, nil
			}
		}
	}
	return nil, errors.Errorf("list did not find report %v.%v", namespace, name)
}

func (list ReportList) AsResources() resources.ResourceList {
	var ress resources.ResourceList
	for _, report := range list {
		ress = append(ress, report)
	}
	return ress
}

func (list ReportList) Names() []string {
	var names []string
	for _, report := range list {
		names = append(names, report.GetMetadata().Name)
	}
	return names
}

func (list ReportList) NamespacesDotNames() []string {
	var names []string
	for _, report := range list {
		names = append(names, report.GetMetadata().Namespace+"."+report.GetMetadata().Name)
	}
	return names
}

func (list ReportList) Sort() ReportList {
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].GetMetadata().Less(list[j].GetMetadata())
	})
	return list
}

func (list ReportList) Clone() ReportList {
	var reportList ReportList
	for _, report := range list {
		reportList = append(reportList, resources.Clone(report).(*Report))
	}
	return reportList
}

func (list ReportList) Each(f func(element *Report)) {
	for _, report := range list {
		f(report)
	}
}

func (list ReportList) EachResource(f func(element resources.Resource)) {
	for _, report := range list {
		f(report)
	}
}

func (list ReportList) AsInterfaces() []interface{} {
	var asInterfaces []interface{}
	list.Each(func(element *Report) {
		asInterfaces = append(asInterfaces, element)
	})
	return asInterfaces
}

var _ resources.Resource = &Report{}

// Kubernetes Adapter for Report

func (o *Report) GetObjectKind() schema.ObjectKind {
	t := ReportCrd.TypeMeta()
	return &t
}

func (o *Report) DeepCopyObject() runtime.Object {
	return resources.Clone(o).(*Report)
}

var ReportCrd = crd.NewCrd("glooshot.solo.io",
	"reports",
	"glooshot.solo.io",
	"v1",
	"Report",
	"report",
	false,
	&Report{})
