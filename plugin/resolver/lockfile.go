package resolver

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
)

// LockFile is a plugin lock configuration.
type LockFile struct {
	Plugins []PluginLock `json:"plugins"`
}

// PluginLock is a lock for a one plugin.
type PluginLock struct {
	Name      Name       `json:"name"`
	Version   Version    `json:"version"`
	Checksums []Checksum `json:"checksums"`
}

// LockCheckResult is the result of a lock check.
type LockCheckResult struct {
	Missing  ConstraintMap
	Mismatch ConstraintMap
	Removed  map[Name]Version
}

// IsInstallRequired returns true if the lock check result requires an install.
func (result LockCheckResult) IsInstallRequired() bool {
	return len(result.Missing) > 0 || len(result.Mismatch) > 0
}

// Check the lock configuration against the given constraints.
func (file *LockFile) Check(constraints ConstraintMap) LockCheckResult {
	missing := ConstraintMap{}
	mismatch := ConstraintMap{}
	removed := map[Name]Version{}
	for name, constraint := range constraints {
		idx := slices.IndexFunc(file.Plugins, func(lock PluginLock) bool {
			return lock.Name == name
		})
		if idx == -1 {
			missing[name] = constraint
			continue
		}
		lock := file.Plugins[idx]
		if !constraint.Check(lock.Version.Version) {
			mismatch[name] = constraint
		}
	}
	for _, lock := range file.Plugins {
		if _, ok := constraints[lock.Name]; !ok {
			removed[lock.Name] = lock.Version
		}
	}
	return LockCheckResult{
		Missing:  missing,
		Mismatch: mismatch,
		Removed:  removed,
	}
}

// ReadLockFile parses a lock configuration from a reader.
func ReadLockFile(r io.Reader) (*LockFile, error) {
	var lockFile LockFile
	err := json.NewDecoder(r).Decode(&lockFile)
	if err != nil {
		return nil, err
	}
	return &lockFile, nil
}

// ReadLockFileFrom parses a lock configuration from a local file.
func ReadLockFileFrom(path string) (*LockFile, error) {
	file, err := os.Open(path) //nolint:gosec // Path is controlled by admin configuration
	if os.IsNotExist(err) {
		return &LockFile{}, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()
	return ReadLockFile(file)
}

// SaveLockFile saves a lock configuration to a writer.
func SaveLockFile(w io.Writer, lockFile *LockFile) error {
	if lockFile == nil {
		return fmt.Errorf("plugin lock file is nil")
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(lockFile)
}

// SaveLockFileTo saves a lock configuration to a local file.
func SaveLockFileTo(path string, lockFile *LockFile) error {
	if lockFile == nil {
		return fmt.Errorf("plugin lock file is nil")
	}
	file, err := os.Create(path) //nolint:gosec // Path is controlled by admin configuration
	if err != nil {
		return err
	}
	defer file.Close()
	return SaveLockFile(file, lockFile)
}
