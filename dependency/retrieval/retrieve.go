package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joshuatcasey/bundler/libdependency/common"
)

var id = "yarn"

func main() {
	buildpackTomlPath := os.Args[1]
	output := os.Args[2]

	fmt.Printf("buildpackTomlPath=%s\n", buildpackTomlPath)
	fmt.Printf("output=%s\n", output)

	retrievalOutput := common.RetrievalOutput{
		Versions: []string{"1.22.18", "1.22.19"},
		ID:       id,
		Name:     "Yarn",
	}

	bytes, err := json.Marshal(retrievalOutput)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(output, bytes, os.ModePerm)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(bytes))
}
