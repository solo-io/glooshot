package install

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"

	"github.com/solo-io/glooshot/pkg/cli"
	"github.com/spf13/cobra"

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
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).To(Equal(noResourcesTable))

			out, stdErr, err = glooshot("create experiment -f ../../examples/gs_delay.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).To(Equal(""))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("delete experiments -n default --all")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).To(Equal(noResourcesTable))
		})
	})

	Context("expect human-friendly errors", func() {

		FIt("should return human-friendly errors on bad input", func() {
			stdOut, stdErr, err := glooshot("--h")
			fmt.Println("start summary")
			fmt.Println(stdOut)
			fmt.Println(stdErr)
			fmt.Println(err)
			fmt.Println("end summary")
			time.Sleep(16 * time.Second)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown flag: --h"))
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(stdOut).To(standardCobraHelpBlockMatcher)
		})

		It("should return human-friendly errors on bad input", func() {
			stdOut, stdErr, err := glooshot("")
			Expect(err).To(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown flag: --h"))
			Expect(stdOut).To(standardCobraHelpBlockMatcher)
		})

	})
})

func glooshot(args string) (string, string, error) {
	mockTargets := cli.NewMockTargets()
	c, e := mockTargets.Stdout.Write([]byte("hello"))
	_, _ = mockTargets.Stdout.Write([]byte("hello2"))
	fmt.Println(c)
	if e != nil {
		fmt.Println("error")
		fmt.Println(e)
	}
	testCliLogger := cli.BuildMockedCliLogger([]string{".glooshot", "log"}, cli.OutputModeEnvVar, &mockTargets)
	ctx := cli.GetInitialContextAndSetLogger(testCliLogger)
	app := cli.App(ctx, "testglooshotcli")
	cStdout, cStderr, err := ExecuteCliOutErr(ctx, app, args)
	fmt.Println("summary stdout:")
	mts, mtw, mtsc := mockTargets.Stdout.Summarize()
	fmt.Printf("stdout %v\n%v\n%v", mts, mtw, mtsc)
	fmt.Println("summary stderr:")
	fmt.Println(mockTargets.Stderr.Summarize())
	return cStdout, cStderr, err
}

////////////////////////////////////////////////////////////////////////////////
// TODO(mitchdraft) replace with https://github.com/solo-io/go-utils/pull/125 on merge
////////////////////////////////////////////////////////////////////////////////
func ExecuteCli(command *cobra.Command, args string) error {
	command.SetArgs(strings.Split(args, " "))
	return command.Execute()
}

func ExecuteCliOutErr(ctx context.Context, command *cobra.Command, args string) (string, string, error) {
	stdOut := os.Stdout
	stdErr := os.Stderr
	r1, w1, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	os.Stdout = w1
	os.Stderr = w2

	command.SetArgs(strings.Split(args, " "))
	err = command.Execute()
	if err != nil {
		contextutils.LoggerFrom(ctx).Errorw("error during glooshot cli execution", zap.Error(err))
	}

	chan1 := make(chan string)
	chan2 := make(chan string)

	chan1err := make(chan error)
	chan2err := make(chan error)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r1)
		chan1err <- err
		chan1 <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r2)
		chan2err <- err
		chan2 <- buf.String()
	}()

	// back to normal state
	os.Stdout = stdOut // restoring the real stdout
	os.Stderr = stdErr
	if err := w1.Close(); err != nil {
		return "", "", err
	}
	if err := w2.Close(); err != nil {
		return "", "", err
	}
	if err := <-chan1err; err != nil {
		return "", "", err
	}
	if err := <-chan2err; err != nil {
		return "", "", err
	}
	capturedStdout := <-chan1
	capturedStderr := <-chan2
	//capturedStdout := ""
	//for len(chan1) > 0 {
	//	capturedStdout += <-chan1
	//}
	//capturedStderr := ""
	//for len(chan2) > 0 {
	//	capturedStderr += <-chan2
	//}

	return strings.TrimSuffix(capturedStdout, "\n"),
		strings.TrimSuffix(capturedStderr, "\n"),
		err
}
