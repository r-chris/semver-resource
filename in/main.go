package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"

	"github.com/concourse/semver-resource/models"
	"github.com/concourse/semver-resource/version"
)

func main() {
	if len(os.Args) < 2 {
		println("usage: " + os.Args[0] + " <destination>")
		os.Exit(1)
	}

	destination := os.Args[1]

	err := os.MkdirAll(destination, 0755)
	if err != nil {
		fatal("creating destination", err)
	}

	var request models.InRequest
	err = json.NewDecoder(os.Stdin).Decode(&request)
	if err != nil {
		fatal("reading request", err)
	}

	auth := aws.Auth{
		AccessKey: request.Source.AccessKeyID,
		SecretKey: request.Source.SecretAccessKey,
	}

	regionName := request.Source.RegionName
	if len(regionName) == 0 {
		regionName = aws.USEast.Name
	}

	region, ok := aws.Regions[regionName]
	if !ok {
		fatal("resolving region name", errors.New(fmt.Sprintf("No such region '%s'", regionName)))
	}

	if len(request.Source.Endpoint) != 0 {
		region = aws.Region{S3Endpoint: fmt.Sprintf("https://%s", request.Source.Endpoint)}
	}

	client := s3.New(auth, region)
	bucket := client.Bucket(request.Source.Bucket)

	inputVersion, err := semver.Parse(request.Version.Number)
	if err != nil {
		fatal("parsing semantic version", err)
	}

	bumped := version.BumpFromParams(request.Params).Apply(inputVersion)

	if !bumped.Equals(inputVersion) {
		fmt.Printf("bumped locally from %s to %s\n", inputVersion, bumped)
	}

	numberFile, err := os.Create(filepath.Join(destination, "number"))
	if err != nil {
		fatal("opening number file", err)
	}

	defer numberFile.Close()

	_, err = fmt.Fprintf(numberFile, "%s", bumped.String())
	if err != nil {
		fatal("writing to number file", err)
	}

	json.NewEncoder(os.Stdout).Encode(models.InResponse{
		Version: request.Version,
		Metadata: models.Metadata{
			{"number", request.Version.Number},
		},
	})
}

func fatal(doing string, err error) {
	println("error " + doing + ": " + err.Error())
	os.Exit(1)
}
