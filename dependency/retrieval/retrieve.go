package main

import (
	"fmt"
	"os"

	"github.com/joshuatcasey/bundler/libdependency/common"
)

func main() {
	buildpackTomlPath := os.Args[1]
	output := os.Args[2]

	fmt.Printf("buildpackTomlPath=%s\n", buildpackTomlPath)
	fmt.Printf("output=%s\n", output)

	common.GetNewVersions("yarn", "Yarn", buildpackTomlPath, common.GetReleasesFromGithub("", "yarnpkg", "yarn"), output)
}
