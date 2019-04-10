package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/solo-io/go-utils/errors"
	version "github.com/solo-io/go-utils/versionutils"
	"github.com/solo-io/solo-kit/pkg/utils/log"
)

func main() {
	tomlTree, err := version.ParseToml()
	fatalCheck(err, "parsing error")

	soloKitVersion, err := version.GetVersion(version.SoloKitPkg, tomlTree)
	fatalCheck(err, "getting solo-kit version")

	glooVersion, err := version.GetVersion(version.GlooPkg, tomlTree)
	fatalCheck(err, "getting gloo version")

	fatalCheck(version.PinGitVersion("../solo-kit", soloKitVersion), "consider git fetching in solo-kit repo")

	if err := FetchGitRepo("../gloo"); err != nil {
		fmt.Printf("Error while fetching gloo: %v\n", err)
	}
	fatalCheck(version.PinGitVersion("../gloo", glooVersion), "consider git fetching in gloo repo")
}

func fatalCheck(err error, msg string) {
	if err != nil {
		log.Fatalf("Error (%v) unable to pin repos!: %v", msg, err)
	}
}

func FetchGitRepo(relativeRepoDir string) error {
	tag := GetTag(version)
	cmd := exec.Command("git", "fetch", origin)
	cmd.Dir = relativeRepoDir
	buf := &bytes.Buffer{}
	out := io.MultiWriter(buf, os.Stdout)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "%v failed: %s", cmd.Args, buf.String())
	}
	return nil
}
