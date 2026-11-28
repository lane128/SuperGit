package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const pendingPackageDir = ".release-pack-pending"

type releaseArtifact struct {
	name   string
	digest [sha256.Size]byte
}

func existingPackage(archive, manifest string) (bool, error) {
	complete := true
	for _, path := range []string{archive, manifest} {
		if err := requireRegularArtifact(path); errors.Is(err, os.ErrNotExist) {
			complete = false
		} else if err != nil {
			return false, err
		}
	}
	return complete, nil
}

func validatePendingPackage(path, archiveName string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 2 {
		return errors.New("pending release package does not contain exactly two expected files")
	}
	for _, name := range []string{archiveName, "update.json"} {
		if err := requireRegularArtifact(filepath.Join(path, name)); err != nil {
			return fmt.Errorf("invalid pending release package: %w", err)
		}
	}
	return nil
}

// installPackage runs with the output directory locked. A prepared directory
// survives interruption between the two links. The manifest is linked last.
func installPackage(output *os.File, prepared, pending string, recovering bool, artifacts []releaseArtifact) error {
	for _, artifact := range artifacts {
		final := filepath.Join(output.Name(), artifact.name)
		if err := compareArtifact(final, artifact.digest); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if !recovering {
		for _, artifact := range artifacts {
			if err := syncPath(filepath.Join(prepared, artifact.name)); err != nil {
				return err
			}
		}
		if err := syncPath(prepared); err != nil {
			return err
		}
		if err := os.Rename(prepared, pending); err != nil {
			return err
		}
		if err := output.Sync(); err != nil {
			return err
		}
	}
	for _, artifact := range artifacts {
		final := filepath.Join(output.Name(), artifact.name)
		if err := os.Link(filepath.Join(pending, artifact.name), final); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
			}
			if err := compareArtifact(final, artifact.digest); err != nil {
				return err
			}
		}
	}
	if err := output.Sync(); err != nil {
		return err
	}
	// Retire the journal atomically before the caller removes its scratch tree.
	return os.Rename(pending, prepared)
}

func syncPath(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

// compareBundleArchives compares archive entries while ignoring ZIP container
// timestamps. This allows a prior package to be resumed without signing or
// replacing different bundle contents (including AppleDouble resource data).
func compareBundleArchives(staged, pending string) error {
	fresh, err := zip.OpenReader(staged)
	if err != nil {
		return err
	}
	defer fresh.Close()
	previous, err := zip.OpenReader(pending)
	if err != nil {
		return err
	}
	defer previous.Close()
	conflict := errors.New("pending archive differs from the current app bundle")
	if len(fresh.File) != len(previous.File) || fresh.Comment != previous.Comment {
		return conflict
	}
	byName := func(a, b *zip.File) int { return strings.Compare(a.Name, b.Name) }
	slices.SortFunc(fresh.File, byName)
	slices.SortFunc(previous.File, byName)
	for i, entry := range fresh.File {
		prior := previous.File[i]
		if i > 0 && entry.Name == fresh.File[i-1].Name {
			return fmt.Errorf("duplicate archive entry %q", entry.Name)
		}
		if entry.Name != prior.Name || entry.Mode() != prior.Mode() || entry.NonUTF8 != prior.NonUTF8 ||
			entry.UncompressedSize64 != prior.UncompressedSize64 || entry.Comment != prior.Comment ||
			!equalZIPExtras(entry.Extra, prior.Extra) {
			return fmt.Errorf("%w: metadata for %q", conflict, entry.Name)
		}
		currentDigest, err := zipEntryDigest(entry)
		if err != nil {
			return err
		}
		priorDigest, err := zipEntryDigest(prior)
		if err != nil {
			return err
		}
		if currentDigest != priorDigest {
			return fmt.Errorf("%w: content of %q", conflict, entry.Name)
		}
	}
	return nil
}

func zipEntryDigest(entry *zip.File) ([sha256.Size]byte, error) {
	file, err := entry.Open()
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return [sha256.Size]byte{}, err
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

func equalZIPExtras(a, b []byte) bool {
	withoutTimestamps := func(extra []byte) ([]byte, bool) {
		var fields []byte
		for len(extra) >= 4 {
			id := binary.LittleEndian.Uint16(extra)
			size := 4 + int(binary.LittleEndian.Uint16(extra[2:]))
			if size > len(extra) {
				return nil, false
			}
			switch {
			case id == 0x5455: // extended timestamp
			case id == 0x5855 && size >= 12: // old Info-ZIP Unix field used by ditto
				// Ignore the eight bytes of timestamps and preserve any UID/GID suffix.
				fields = append(fields, extra[:4]...)
				fields = append(fields, extra[12:size]...)
			default:
				fields = append(fields, extra[:size]...)
			}
			extra = extra[size:]
		}
		return fields, len(extra) == 0
	}
	a, aOK := withoutTimestamps(a)
	b, bOK := withoutTimestamps(b)
	return aOK && bOK && bytes.Equal(a, b)
}
