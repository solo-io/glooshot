package main

import "github.com/solo-io/go-utils/githubutils"

func main() {
	assets := make([]githubutils.ReleaseAssetSpec, 4)
	assets[0] = githubutils.ReleaseAssetSpec{
		Name:       "glooshot.yaml",
		ParentPath: "install",
	}
	assets[1] = githubutils.ReleaseAssetSpec{
		Name:       "glooshot-darwin-amd64",
		ParentPath: "_output",
		UploadSHA:  true,
	}
	assets[2] = githubutils.ReleaseAssetSpec{
		Name:       "glooshot-linux-amd64",
		ParentPath: "_output",
		UploadSHA:  true,
	}
	assets[3] = githubutils.ReleaseAssetSpec{
		Name:       "glooshot-windows-amd64.exe",
		ParentPath: "_output",
		UploadSHA:  true,
	}
	spec := githubutils.UploadReleaseAssetSpec{
		Owner:             "solo-io",
		Repo:              "glooshot",
		Assets:            assets,
		SkipAlreadyExists: true,
	}
	githubutils.UploadReleaseAssetCli(&spec)
}
