package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/joshuatcasey/bundler/libdependency/common"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"golang.org/x/exp/slices"
)

type Matrix struct {
	Image   string   `json:"image"`
	Version string   `json:"version"`
	Target  string   `json:"target"`
	Stacks  []string `json:"stacks"`
}

func main() {
	id := os.Args[1]
	artifactPath := os.Args[2]
	buildpackTomlPath := os.Args[3]

	fmt.Printf("id=%s\n", id)
	fmt.Printf("artifactPath=%s\n", artifactPath)
	fmt.Printf("buildpackTomlPath=%s\n", buildpackTomlPath)

	if exists, err := fs.Exists(artifactPath); err != nil {
		panic(err)
	} else if !exists {
		panic(fmt.Errorf("directory %s does not exist", artifactPath))
	} else if fs.IsEmptyDir(artifactPath) {
		panic(fmt.Errorf("directory %s is empty", artifactPath))
	}

	versionToMetadata := getMetadata(artifactPath)
	fmt.Println("Found metadata:")
	printAsJson(versionToMetadata)

	artifacts := findArtifacts(artifactPath, id, versionToMetadata)

	fmt.Println("Found artifacts:")
	printAsJson(artifacts)

	prepareCommit(artifacts, buildpackTomlPath)

	bytes, err := os.ReadFile(buildpackTomlPath)
	if err != nil {
		panic(err)
	}

	fmt.Printf(">>> %s\n", buildpackTomlPath)
	fmt.Print(string(bytes))
	fmt.Printf("\n<<<\n")

	prune(buildpackTomlPath)
}

func prepareCommit(artifacts []cargo.ConfigMetadataDependency, buildpackTomlPath string) {
	config := common.ParseBuildpackToml(buildpackTomlPath)

	config.Metadata.Dependencies = append(config.Metadata.Dependencies, artifacts...)

	file, err := os.OpenFile(buildpackTomlPath, os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		panic(fmt.Errorf("failed to open buildpack config file: %w", err))
	}
	defer file.Close()

	err = cargo.EncodeConfig(file, config)
	if err != nil {
		panic(fmt.Errorf("failed to write buildpack config: %w", err))
	}
}

func getMetadata(artifactPath string) map[string]cargo.ConfigMetadataDependency {
	versionToMetadata := make(map[string]cargo.ConfigMetadataDependency)
	metadataGlob := filepath.Join(artifactPath, "metadata-*.json")
	if metadataFiles, err := filepath.Glob(metadataGlob); err != nil {
		panic(err)
	} else if len(metadataFiles) < 1 {
		panic(fmt.Errorf("no metadata files found: %s", metadataGlob))
	} else {
		fmt.Printf("Found metadata files:\n")
		for _, metadata := range metadataFiles {
			fmt.Printf("- %s\n", filepath.Base(metadata))

			version := strings.TrimPrefix(filepath.Base(metadata), "metadata-")
			version = strings.TrimSuffix(version, ".json")

			var depVersion cargo.ConfigMetadataDependency

			metadataContents, err := os.ReadFile(filepath.Join(metadata, filepath.Base(metadata)))
			if err != nil {
				panic(err)
			}

			err = json.Unmarshal(metadataContents, &depVersion)
			if err != nil {
				panic(fmt.Errorf("failed to parse metadata file: %w", err))
			}

			versionToMetadata[version] = depVersion
		}
	}
	return versionToMetadata
}

func printAsJson(item interface{}) {
	bytes, err := json.Marshal(item)
	if err != nil {
		panic("cannot marshal")
	}
	fmt.Println(string(bytes))
}

func findArtifacts(artifactDir string, id string, versionsToMetadata map[string]cargo.ConfigMetadataDependency) []cargo.ConfigMetadataDependency {
	var artifacts []cargo.ConfigMetadataDependency

	tarballGlob := filepath.Join(artifactDir, fmt.Sprintf("%s-*", id))
	if allDirsForArtifacts, err := filepath.Glob(tarballGlob); err != nil {
		panic(err)
	} else if len(allDirsForArtifacts) < 1 {
		panic(fmt.Errorf("no compiled artifact folders found: %s", tarballGlob))
	} else {
		fmt.Printf("Found compiled artifact folders:\n")
		for _, singleDirForArtifact := range allDirsForArtifacts {
			fmt.Printf("- %s\n", filepath.Base(singleDirForArtifact))

			dir, err := os.Open(singleDirForArtifact)
			if err != nil {
				panic(err)
			}

			files, err := dir.Readdir(0)
			if err != nil {
				panic(err)
			}

			var artifact cargo.ConfigMetadataDependency

			tarballSHA256 := ""
			tarballPath := ""

			for _, file := range files {
				fullpath := filepath.Join(singleDirForArtifact, file.Name())
				fmt.Printf("  - %s\n", file.Name())

				if isTarball(file) {
					tarballPath = fullpath
					continue
				}

				bytes, err := os.ReadFile(fullpath)
				if err != nil {
					panic(err)
				}

				if file.Name() == "matrix.json" {
					var matrix Matrix
					err = json.Unmarshal(bytes, &matrix)
					if err != nil {
						panic(err)
					}

					artifact = versionsToMetadata[matrix.Version]
					// TODO: fix unknown to real URI
					artifact.URI = "<UNKNOWN>"
					artifact.Stacks = matrix.Stacks
				}

				if isSHA256(file) {
					tarballSHA256 = strings.TrimSpace(string(bytes))
				}
			}

			calculatedSHA256, err := fs.NewChecksumCalculator().Sum(tarballPath)
			if err != nil {
				panic(err)
			}
			if !strings.HasPrefix(tarballSHA256, calculatedSHA256) {
				fmt.Printf("SHA256 does not match! Expected=%s, Calculated=%s\n", tarballSHA256, calculatedSHA256)
				panic("SHA256 does not match!")
			}

			artifact.SHA256 = calculatedSHA256
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts
}

func isTarball(file os.FileInfo) bool {
	return strings.HasSuffix(file.Name(), ".tgz")
}

func isSHA256(file os.FileInfo) bool {
	return strings.HasSuffix(file.Name(), ".sha256")
}

func prune(buildpackTomlPath string) {
	config := common.ParseBuildpackToml(buildpackTomlPath)

	// Get a map from constraints to dependencies
	constraintToDependencies := make(map[cargo.ConfigMetadataDependencyConstraint][]cargo.ConfigMetadataDependency)

	for _, dependency := range config.Metadata.Dependencies {
		dependencyVersionAsSemver := semver.MustParse(dependency.Version)
		for _, constraint := range config.Metadata.DependencyConstraints {
			constraintAsSemver, err := semver.NewConstraint(constraint.Constraint)
			if err != nil {
				panic(err)
			}

			if dependency.ID == constraint.ID && constraintAsSemver.Check(dependencyVersionAsSemver) {
				constraintToDependencies[constraint] = append(constraintToDependencies[constraint], dependency)
			}
		}
	}

	constraintToPatches := make(map[cargo.ConfigMetadataDependencyConstraint][]string)

	// We can have more than one dependency with the same version
	// so we have to figure out which versions are captured in the patches
	for constraint, dependencies := range constraintToDependencies {
		for _, dependency := range dependencies {
			if !slices.Contains(constraintToPatches[constraint], dependency.Version) {
				constraintToPatches[constraint] = append(constraintToPatches[constraint], dependency.Version)
			}
		}

		sort.Slice(constraintToPatches[constraint], func(i, j int) bool {
			iVersion := semver.MustParse(constraintToPatches[constraint][i])
			jVersion := semver.MustParse(constraintToPatches[constraint][j])
			return iVersion.LessThan(jVersion)
		})

		if constraint.Patches < len(constraintToPatches[constraint]) {
			constraintToPatches[constraint] = constraintToPatches[constraint][len(constraintToPatches[constraint])-constraint.Patches:]
		}
	}

	var patchesToKeep []string
	for _, versions := range constraintToPatches {
		patchesToKeep = append(patchesToKeep, versions...)
	}

	fmt.Println("patchesToKeep")
	fmt.Println(patchesToKeep)

	var dependenciesToKeep []cargo.ConfigMetadataDependency

	for _, dependency := range config.Metadata.Dependencies {
		if slices.Contains(patchesToKeep, dependency.Version) {
			dependenciesToKeep = append(dependenciesToKeep, dependency)
		}
	}

	// Sort the stacks within the dependency
	for _, dependency := range dependenciesToKeep {
		sort.Slice(dependency.Stacks, func(i, j int) bool {
			return dependency.Stacks[i] > dependency.Stacks[j]
		})
	}

	// Sort the dependencies by:
	// 1. ID
	// 2. Version
	// 3. len(Stacks)
	sort.Slice(dependenciesToKeep, func(i, j int) bool {
		dep1 := dependenciesToKeep[i]
		dep2 := dependenciesToKeep[j]
		if dep1.ID == dep2.ID {
			dep1Version := semver.MustParse(dep1.Version)
			dep2Version := semver.MustParse(dep2.Version)

			if dep1Version.Equal(dep2Version) {
				return len(dep1.Stacks) < len(dep2.Stacks)
			}

			return dep1Version.LessThan(dep2Version)
		}
		return dep1.ID < dep2.ID
	})

	fmt.Println("dependenciesToKeep")
	fmt.Println(dependenciesToKeep)

	config.Metadata.Dependencies = dependenciesToKeep

	file, err := os.OpenFile(buildpackTomlPath, os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		panic(fmt.Errorf("failed to open buildpack config file: %w", err))
	}
	defer file.Close()

	err = cargo.EncodeConfig(file, config)
	if err != nil {
		panic(fmt.Errorf("failed to write buildpack config: %w", err))
	}
}
