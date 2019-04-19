package install

import (
	"github.com/solo-io/glooshot/pkg/cli"
	clilog "github.com/solo-io/glooshot/pkg/pregoutils-clilog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/gloo/projects/gloo/cli/pkg/helpers"
)

var _ = Describe("Glooshot CLI", func() {

	var noResourcesTable = `+------------+-----------+--------+
| EXPERIMENT | NAMESPACE | STATUS |
+------------+-----------+--------+
+------------+-----------+--------+`
	var standardCobraHelpBlockMatcher = MatchRegexp("Available Commands:")

	BeforeEach(func() {
		helpers.UseMemoryClients()
		_, _, _ = glooshot("delete experiment --every-resource")
	})

	Context("basic args and flags", func() {
		It("should return help messages without error", func() {
			_, _, err := glooshot("-h")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = glooshot("help")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = glooshot("--help")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform basic create, get, and delete commands", func() {
			out, stdErr, err := glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).To(Equal(noResourcesTable))

			out, stdErr, err = glooshot("create experiment -f ../../examples/gs_delay.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).To(Equal(""))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("delete experiments -n default --all")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).To(Equal(noResourcesTable))
		})
	})

	Context("expect human-friendly errors", func() {
		It("should return human-friendly errors on bad input", func() {
			cliOut := glooshotWithLoggerOutput("--h")
			Expect(cliOut.CobraStdout).To(Equal(""))
			Expect(cliOut.CobraStderr).To(standardCobraHelpBlockMatcher)
			// logs are not used in this code path so they should be empty
			Expect(cliOut.LoggerConsoleStout).To(Equal(""))
			Expect(cliOut.LoggerConsoleStderr).To(Equal(""))
		})
	})

	Context("expect human-friendly logs", func() {
		It("should return human-friendly errors on bad input", func() {
			cliOut := glooshotWithLoggerOutput("--temp")
			Expect(cliOut.CobraStdout).
				To(Equal("cobra says 'hisssss' - but he should leave the console logs to the CliLog* utils."))
			Expect(cliOut.CobraStderr).
				To(MatchRegexp("Error: cobra says 'hisssss' again - it's ok because this is a passed error"))
			Expect(cliOut.CobraStderr).
				To(standardCobraHelpBlockMatcher)
			Expect(cliOut.LoggerConsoleStout).
				To(Equal(`this info log should go to file and console
this warn log should go to file and console`))
			Expect(cliOut.LoggerConsoleStderr).To(Equal(`this error log should go to file and console
`))
		})

	})
})

func glooshot(args string) (string, string, error) {
	co := glooshotWithLoggerOutput(args)
	return co.CobraStdout, co.CobraStderr, nil
}

func glooshotWithLoggerOutput(args string) clilog.CliOutput {
	cliOutput, err := cli.GlooshotConfig.RunForTest(args)
	Expect(err).NotTo(HaveOccurred())
	return cliOutput
}
