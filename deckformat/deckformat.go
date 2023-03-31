package deckformat

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kong/go-apiops/jsonbasics"
)

const (
	VersionKey   = "_format_version"
	TransformKey = "_transform"
)

// CompatibleTransform checks if 2 files are compatible, by '_transform' keys.
// Returns nil if compatible, and error otherwise.
func CompatibleTransform(data1 map[string]interface{}, data2 map[string]interface{}) error {
	if data1 == nil {
		panic("expected 'data1' to be non-nil")
	}
	if data2 == nil {
		panic("expected 'data2' to be non-nil")
	}

	transform1 := true // this is the default value
	if data1[TransformKey] != nil {
		var err error
		if transform1, err = jsonbasics.GetBoolField(data1, TransformKey); err != nil {
			return err
		}
	}
	transform2 := true // this is the default value
	if data2[TransformKey] != nil {
		var err error
		if transform2, err = jsonbasics.GetBoolField(data2, TransformKey); err != nil {
			return err
		}
	}

	if transform1 != transform2 {
		return errors.New("files with '_transform: true' (default) and '_transform: false' are not compatible")
	}

	return nil
}

// CompatibleVersion checks if 2 files are compatible, by '_format_version'. Version is compatible
// if they are the same major. Missing versions are assumed to be compatible.
// Returns nil if compatible, and error otherwise.
func CompatibleVersion(data1 map[string]interface{}, data2 map[string]interface{}) error {
	if data1 == nil {
		panic("expected 'data1' to be non-nil")
	}
	if data2 == nil {
		panic("expected 'data2' to be non-nil")
	}

	if data1[VersionKey] == nil {
		if data2[VersionKey] == nil {
			return nil // neither given , so assume compatible
		}
		// data1 omitted, just validate data2 has a proper version, any version will do
		_, _, err := ParseFormatVersion(data2)
		return err
	}

	// data1 has a version
	if data2[VersionKey] == nil {
		// data2 omitted, just validate data1 has a proper version, any version will do
		_, _, err := ParseFormatVersion(data1)
		return err
	}

	// both versions given, go parse them
	major1, minor1, err1 := ParseFormatVersion(data1)
	if err1 != nil {
		return err1
	}
	major2, minor2, err2 := ParseFormatVersion(data2)
	if err2 != nil {
		return err2
	}

	if major1 != major2 {
		return fmt.Errorf("major versions are incompatible; %d.%d and %d.%d", major1, minor1, major2, minor2)
	}

	return nil
}

// CompatibleFile returns nil if the files are compatible. An error otherwise.
// see CompatibleVersion and CompatibleTransform for what compatibility means.
func CompatibleFile(data1 map[string]interface{}, data2 map[string]interface{}) error {
	err := CompatibleTransform(data1, data2)
	if err != nil {
		return fmt.Errorf("files are incompatible; %w", err)
	}
	err = CompatibleVersion(data1, data2)
	if err != nil {
		return fmt.Errorf("files are incompatible; %w", err)
	}
	return nil
}

// parseFormatVersion parses field `_format_version` and returns major+minor.
// Field must be present, a string, and have an 'x.y' format. Returns an error otherwise.
func ParseFormatVersion(data map[string]interface{}) (int, int, error) {
	// get the file version and check it
	v, err := jsonbasics.GetStringField(data, VersionKey)
	if err != nil {
		return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
	}
	elem := strings.Split(v, ".")
	if len(elem) > 2 {
		return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
	}

	majorVersion, err := strconv.Atoi(elem[0])
	if err != nil {
		return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
	}

	minorVersion := 0
	if len(elem) > 1 {
		minorVersion, err = strconv.Atoi(elem[1])
		if err != nil {
			return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
		}
	}

	return majorVersion, minorVersion, nil
}
