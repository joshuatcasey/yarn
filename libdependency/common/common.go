package common

import (
	"fmt"

	"github.com/paketo-buildpacks/packit/v2/cargo"
)

func ParseBuildpackToml(buildpackTomlPath string) cargo.Config {
	configParser := cargo.NewBuildpackParser()
	config, err := configParser.Parse(buildpackTomlPath)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %s: %s", buildpackTomlPath, err))
	}
	return config
}

type RetrievalOutput struct {
	Versions []string
	ID       string
	Name     string
}
