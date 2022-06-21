package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"

	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/go-enry/go-license-detector/v4/licensedb/filer"
	"github.com/package-url/packageurl-go"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/vacation"
)

func main() {
	version := os.Args[1]
	id := os.Args[2]
	name := os.Args[3]
	output := os.Args[4]

	fmt.Printf("version=%s\n", version)
	fmt.Printf("id=%s\n", id)
	fmt.Printf("name=%s\n", name)
	fmt.Printf("output=%s\n", output)

	source := fmt.Sprintf("https://github.com/yarnpkg/yarn/releases/download/v%[1]s/yarn-v%[1]s.tar.gz", version)
	sha := getSHA256Sum(source)

	dependencyVersion := cargo.ConfigMetadataDependency{
		Version:         version,
		Source:          source,
		SourceSHA256:    sha,
		DeprecationDate: nil,
		CPE:             fmt.Sprintf("cpe:2.3:a:%[1]spkg:%[1]s:%[2]s:*:*:*:*:*:*:*", id, version),
		PURL:            generatePURL(id, version, sha, source),
		Licenses:        lookupLicenses(source),
	}
	dependencyVersion.ID = id
	dependencyVersion.Name = name
	bytes, err := json.Marshal(dependencyVersion)
	if err != nil {
		panic(fmt.Errorf("cannot marshal: %w", err))
	}

	err = os.WriteFile(output, bytes, os.ModePerm)
	if err != nil {
		panic(fmt.Errorf("cannot write to %s: %w", output, err))
	}

	fmt.Println(string(bytes))
}

func getSHA256Sum(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	file, err := os.CreateTemp("", "yarn-download")
	if err != nil {
		panic(err)
	}

	defer file.Close()
	defer os.RemoveAll(file.Name())

	// Write the body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		panic(err)
	}

	calculatedSHA256, err := fs.NewChecksumCalculator().Sum(file.Name())
	if err != nil {
		panic(err)
	}

	return calculatedSHA256
}

func lookupLicenses(sourceURL string) []interface{} {
	// getting the dependency artifact from sourceURL
	resp, err := http.Get(sourceURL)
	if err != nil {
		panic(fmt.Errorf("failed to query url: %w", err))
	}
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("failed to query url %s with: status code %d", sourceURL, resp.StatusCode))
	}

	// decompressing the dependency artifact
	tempDir, err := os.MkdirTemp("", "destination")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	err = defaultDecompress(resp.Body, tempDir, 1)
	if err != nil {
		panic(err)
	}

	// scanning artifact for license file
	filer, err := filer.FromDirectory(tempDir)
	if err != nil {
		panic(fmt.Errorf("failed to setup a licensedb filer: %w", err))
	}

	licenses, err := licensedb.Detect(filer)
	// if no licenses are found, just return an empty slice.
	if err != nil {
		if err.Error() != "no license file was found" {
			panic(fmt.Errorf("failed to detect licenses: %w", err))
		}
		return []interface{}{}
	}

	// Only return the license IDs, in alphabetical order
	var licenseIDs []string
	for key := range licenses {
		licenseIDs = append(licenseIDs, key)
	}
	sort.Strings(licenseIDs)

	var licenseIDsAsInterface []interface{}
	for _, licenseID := range licenseIDs {
		licenseIDsAsInterface = append(licenseIDsAsInterface, licenseID)
	}

	return licenseIDsAsInterface
}

func defaultDecompress(artifact io.Reader, destination string, stripComponents int) error {
	archive := vacation.NewArchive(artifact)

	err := archive.StripComponents(stripComponents).Decompress(destination)
	if err != nil {
		return fmt.Errorf("failed to decompress source file: %w", err)
	}

	return nil
}

func generatePURL(name, version, sourceSHA, source string) string {
	purl := packageurl.NewPackageURL(
		packageurl.TypeGeneric,
		"",
		name,
		version,
		packageurl.QualifiersFromMap(map[string]string{
			"checksum":     sourceSHA,
			"download_url": source,
		}),
		"",
	)

	// Unescape the path to remove the added `%2F` and other encodings added to
	// the URL by packageurl-go
	// If the unescaping fails, we should still return the path URL with the
	// encodings, packageurl-go has examples with both the encodings and without,
	// we prefer to avoid the encodings when we can for convenience.
	purlString, err := url.PathUnescape(purl.ToString())
	if err != nil {
		return purl.ToString()
	}

	return purlString
}
