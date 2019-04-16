package printer

import (
	"fmt"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/cliutils"
)

func Experiment(exp v1.Experiment) {
	fmt.Printf("Experiment: %s in namespace: %s\n", exp.Metadata.Name, exp.Metadata.Namespace)
	fmt.Printf("Status: %s\n", exp.Status.State.String())
	PrintExperiments([]*v1.Experiment{&exp}, "")
}

func PrintExperiments(exps []*v1.Experiment, outputType string) {
	err := cliutils.PrintList(outputType, "", exps,
		func(data interface{}, w io.Writer) error {
			experimentTable(exps, w)
			return nil
		}, os.Stdout)
	if err != nil {
		fmt.Printf("error during print: %v\n", err)
	}
}

func experimentTable(list []*v1.Experiment, w io.Writer) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Experiment", "Namespace", "Status"})

	for _, v := range list {
		name := v.GetMetadata().Name
		namespace := v.GetMetadata().Namespace
		status := v.Status.String()

		table.Append([]string{name, namespace, status})
	}

	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.Render()
}
