package filebasics

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"sigs.k8s.io/yaml"
)

const (
	defaultJSONIndent = "  "
)

// MustReadFile reads file contents. Will panic if reading fails.
// Reads from stdin if filename == "-"
func MustReadFile(filename string) *[]byte {
	var (
		body []byte
		err  error
	)

	if filename == "-" {
		body, err = io.ReadAll(os.Stdin)
	} else {
		body, err = os.ReadFile(filename)
	}

	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}
	return &body
}

// mustWriteFile writes the output to a file. Will panic if writing fails.
// Writes to stdout if filename == "-"
func MustWriteFile(filename string, content *[]byte) {
	var f *os.File
	var err error

	if filename != "-" {
		// write to file
		f, err = os.Create(filename)
		if err != nil {
			log.Fatalf("failed to create output file '%s'", filename)
		}
		defer f.Close()
	} else {
		// writing to stdout
		f = os.Stdout
	}
	_, err = f.Write(*content)
	if err != nil {
		log.Fatalf(fmt.Sprintf("failed to write to output file '%s'; %%w", filename), err)
	}
}

// mustSerialize will serialize the result as a JSON/YAML. Will panic
// if serializing fails.
func MustSerialize(content map[string]interface{}, asYaml bool) *[]byte {
	var (
		str []byte
		err error
	)

	if asYaml {
		str, err = yaml.Marshal(content)
		if err != nil {
			log.Fatal("failed to yaml-serialize the resulting file; %w", err)
		}
	} else {
		str, err = json.MarshalIndent(content, "", defaultJSONIndent)
		if err != nil {
			log.Fatal("failed to json-serialize the resulting file; %w", err)
		}
	}

	return &str
}

// MustWriteSerializedFile will serialize the data and write it to a file. Will
// panic if it fails. Writes to stdout if filename == "-"
func MustWriteSerializedFile(filename string, content map[string]interface{}, asYaml bool) {
	MustWriteFile(filename, MustSerialize(content, asYaml))
}
