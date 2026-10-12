package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/KitsuneForgering/FileOrganizer/internal/organizer"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in io.Reader, out, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "preview" && args[0] != "apply") {
		fmt.Fprintln(stderr, "usage: fileorganizer-core preview|apply < request.json")
		return 2
	}
	var request organizer.Request
	decoder := json.NewDecoder(in)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		fmt.Fprintln(stderr, "expected exactly one JSON request")
		return 2
	}
	var result organizer.Result
	var err error
	if args[0] == "preview" {
		result, err = organizer.Preview(request)
	} else {
		result, err = organizer.Apply(request)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := json.NewEncoder(out).Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
