package main

import (
	"github.com/pkg/errors"
	"github.com/solo-io/go-utils/versionutils"

	"github.com/solo-io/solo-kit/pkg/code-generator/cmd"
	"github.com/solo-io/solo-kit/pkg/code-generator/docgen/options"
	"github.com/solo-io/solo-kit/pkg/utils/log"
)

//go:generate go run generate.go

func main() {
	err := checkVersions()
	if err != nil {
		log.Fatalf("generate failed!: %v", err)
	}
	log.Printf("starting generate")
	docsOpts := cmd.DocsOptions{
		Output: options.Hugo,
	}
	if err := cmd.Run(".", true, &docsOpts, nil, nil); err != nil {
		log.Fatalf("generate failed!: %v", err)
	}
}

func checkVersions() error {
	log.Printf("Checking expected solo kit version...")
	tomlTree, err := versionutils.ParseToml()
	if err != nil {
		return err
	}

	expectedSoloKitVersion, err := versionutils.GetVersion(versionutils.SoloKitPkg, tomlTree)
	if err != nil {
		return err
	}

	log.Printf("Checking repo versions...")
	actualSoloKitVersion, err := versionutils.GetGitVersion("../solo-kit")
	if err != nil {
		return err
	}
	expectedTaggedSoloKitVersion := versionutils.GetTag(expectedSoloKitVersion)
	if expectedTaggedSoloKitVersion != actualSoloKitVersion {
		return errors.Errorf("Expected solo kit version %s, found solo kit version %s in repo. Run 'make pin-repos' or fix manually.", expectedTaggedSoloKitVersion, actualSoloKitVersion)
	}
	log.Printf("Versions are pinned correctly.")
	return nil
}
