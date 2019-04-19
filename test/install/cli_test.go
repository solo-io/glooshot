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
		FIt("should return human-friendly errors on bad input", func() {
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
			// match the tags that are part of the rich log output
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("level"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("ts"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("warn"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("error"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("dev"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("msg"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("logger"))
			// match (or not) the fragments that we get in the console. Using regex since timestamp is random
			// see sampleLogFileContent for an example of the full output
			Expect(cliOut.LoggerFileContent).NotTo(MatchRegexp("CliLog* utils"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("ok because this is a passed error"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("info log"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("warn log"))
			Expect(cliOut.LoggerFileContent).To(MatchRegexp("error log"))

		})

	})
})

const sampleLogFileContent = `{"level":"info","ts":"2019-04-19T16:43:36.214-0400","logger":"dev","msg":"this info log should go to file and console","version":"dev","cli":"this info log should go to file and console"}
{"level":"warn","ts":"2019-04-19T16:43:36.214-0400","logger":"dev","msg":"this warn log should go to file and console","version":"dev","cli":"this warn log should go to file and console"}
{"level":"error","ts":"2019-04-19T16:43:36.214-0400","logger":"dev","msg":"this error log should go to file and console","version":"dev","cli":"this error log should go to file and console"}
{"level":"error","ts":"2019-04-19T16:43:36.215-0400","logger":"dev","msg":"error during glooshot cli execution","version":"dev","error":"cobra says 'hisssss' again - it's ok because this is a passed error"}
`

func glooshot(args string) (string, string, error) {
	co := glooshotWithLoggerOutput(args)
	return co.CobraStdout, co.CobraStderr, nil
}

func glooshotWithLoggerOutput(args string) clilog.CliOutput {
	cliOutput, err := cli.GlooshotConfig.RunForTest(args)
	Expect(err).NotTo(HaveOccurred())
	return cliOutput
}
