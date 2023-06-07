package patch

import (
	"fmt"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"gopkg.in/yaml.v3"
)

// DeckPatchFile represents a list of patches.
type DeckPatchFile struct {
	VersionMajor int // 0 if not present
	VersionMinor int // 0 if not present
	Patches      []DeckPatch
}

// ParseFile parses a patchfile. Any non-object in the 'patches' array will be
// ignored. If the array doesn't exist, it returns an empty array.
func (patchFile *DeckPatchFile) ParseFile(filename string) error {
	data, err := filebasics.DeserializeFile(filename)
	if err != nil {
		return err
	}

	if data[deckformat.VersionKey] != nil {
		logbasics.Debug("parsed patch file", "file", filename, "version", data[deckformat.VersionKey])
		patchFile.VersionMajor, patchFile.VersionMinor, err = deckformat.ParseFormatVersion(data)
		if err != nil {
			return fmt.Errorf("%s: has an invalid "+deckformat.VersionKey+" specified; %w", filename, err)
		}
	} else {
		logbasics.Debug("parsed unversioned patch-file", "file", filename)
	}

	patchesRead, err := jsonbasics.GetObjectArrayField(data, "patches")
	if err != nil {
		return fmt.Errorf("%s: field 'patches' is not an array; %w", filename, err)
	}

	patchFile.Patches = make([]DeckPatch, 0)
	for i, patch := range patchesRead {
		if patch["values"] != nil || patch["remove"] != nil {
			// deck patch
			var patchParsed DeckPatch
			err := patchParsed.Parse(patch, fmt.Sprintf("%s: patches[%d]", filename, i))
			if err != nil {
				return err
			}
			patchFile.Patches = append(patchFile.Patches, patchParsed)
		}
	}

	return nil
}

// Apply applies the set of patches on the yaml.Node given.
func (patchFile *DeckPatchFile) Apply(yamlData *yaml.Node) error {
	for i, patch := range patchFile.Patches {
		err := patch.ApplyToNodes(yamlData)
		if err != nil {
			return fmt.Errorf("failed to apply patch %d; %w", i, err)
		}
	}
	return nil
}

// MustApply applies the set of patches on the yaml.Node given. Same as Apply, but
// in case of an error it will panic.
// 'source' will be used to format the error in case of a panic.
func (patchFile *DeckPatchFile) MustApply(yamlData *yaml.Node, source string) {
	err := patchFile.Apply(yamlData)
	if err != nil {
		panic(fmt.Errorf("failed to apply patchfile '%s'; %w", source, err))
	}
}
