package filebasics

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

type OutputFormat string

const (
	defaultJSONIndent              = "  "
	OutputFormatYaml  OutputFormat = "yaml"
	OutputFormatJSON  OutputFormat = "json"
)

// ReadFile reads file contents.
// Reads from stdin if filename == "-"
func ReadFile(filename string) ([]byte, error) {
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
		return nil, err
	}
	return body, nil
}

// MustReadFile reads file contents. Will panic if reading fails.
// Reads from stdin if filename == "-"
func MustReadFile(filename string) []byte {
	body, err := ReadFile(filename)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}

	return body
}

// WriteFile writes the output to a file.
// Writes to stdout if filename == "-"
func WriteFile(filename string, content []byte) error {
	var f *os.File
	var err error

	if filename != "-" {
		// write to file
		f, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create output file '%s'; %w", filename, err)
		}
		defer f.Close()
	} else {
		// writing to stdout
		f = os.Stdout
	}
	_, err = f.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write to output file '%s'; %w", filename, err)
	}
	return nil
}

// MustWriteFile writes the output to a file. Will panic if writing fails.
// Writes to stdout if filename == "-"
func MustWriteFile(filename string, content []byte) {
	err := WriteFile(filename, content)
	if err != nil {
		panic(err)
	}
}

// Serialize will serialize the result as a JSON/YAML. Format parameter is case-insensitive.
func Serialize(content map[string]interface{}, format OutputFormat) ([]byte, error) {
	var (
		str []byte
		err error
	)

	format = OutputFormat(strings.ToLower(string(format)))

	switch format {
	case OutputFormatYaml:
		str, err = yaml.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("failed to yaml-serialize the resulting file; %w", err)
		}
	case OutputFormatJSON:
		str, err = json.MarshalIndent(content, "", defaultJSONIndent)
		if err != nil {
			return nil, fmt.Errorf("failed to json-serialize the resulting file; %w", err)
		}
	default:
		return nil, fmt.Errorf("expected 'format' to be either '%s' or '%s', got: '%s'",
			OutputFormatYaml, OutputFormatJSON, format)
	}

	return str, nil
}

// MustSerialize will serialize the result as a JSON/YAML. Will panic
// if serializing fails.
func MustSerialize(content map[string]interface{}, format OutputFormat) []byte {
	result, err := Serialize(content, format)
	if err != nil {
		panic(err)
	}
	return result
}

// Deserialize will deserialize data as a JSON or YAML object. Will return an error
// if deserializing fails or if it isn't an object.
func Deserialize(data []byte) (map[string]interface{}, error) {
	var output interface{}

	err1 := json.Unmarshal(data, &output)
	if err1 != nil {
		err2 := yaml.Unmarshal(data, &output)
		if err2 != nil {
			return nil, errors.New("failed deserializing data as JSON and as YAML")
		}
	}

	switch output := output.(type) {
	case map[string]interface{}:
		return output, nil
	}

	return nil, errors.New("expected the data to be an Object")
}

// MustDeserialize will deserialize data as a JSON or YAML object. Will panic
// if deserializing fails or if it isn't an object. Will never return nil.
func MustDeserialize(data []byte) map[string]interface{} {
	jsondata, err := Deserialize(data)
	if err != nil {
		log.Fatal("%w", err)
	}
	return jsondata
}

// WriteSerializedFile will serialize the data and write it to a file.
// Writes to stdout if filename == "-"
func WriteSerializedFile(filename string, content map[string]interface{}, format OutputFormat) error {
	serializedContent, err := Serialize(content, format)
	if err != nil {
		return err
	}
	err = WriteFile(filename, serializedContent)
	if err != nil {
		return err
	}
	return nil
}

// MustWriteSerializedFile will serialize the data and write it to a file. Will
// panic if it fails. Writes to stdout if filename == "-"
func MustWriteSerializedFile(filename string, content map[string]interface{}, format OutputFormat) {
	MustWriteFile(filename, MustSerialize(content, format))
}

// DeserializeFile will read a JSON or YAML file and return the top-level object. Will return an
// error if it fails reading or the content isn't an object. Reads from stdin if filename == "-".
func DeserializeFile(filename string) (map[string]interface{}, error) {
	bytedata, err := ReadFile(filename)
	if err != nil {
		return nil, err
	}
	data, err := Deserialize(bytedata)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MustDeserializeFile will read a JSON or YAML file and return the top-level object. Will
// panic if it fails reading or the content isn't an object. Reads from stdin if filename == "-".
// This will never return nil.
func MustDeserializeFile(filename string) map[string]interface{} {
	return MustDeserialize(MustReadFile(filename))
}
