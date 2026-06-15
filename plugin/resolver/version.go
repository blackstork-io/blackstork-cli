// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package resolver

import (
	"fmt"
	"strconv"

	"github.com/Masterminds/semver/v3"
)

// Version is a version of a plugin. It is a wrapper around semver.Version with strict parsing.
type Version struct {
	*semver.Version
}

// UnmarshalJSON parses a JSON string into a PluginVersion using strict semver parsing.
func (v *Version) UnmarshalJSON(data []byte) error {
	raw, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("failed to unquote version: %w", err)
	}
	ver, err := semver.StrictNewVersion(raw)
	if err != nil {
		return err
	}
	*v = Version{ver}
	return nil
}

// Compare compares the version with another version.
func (v Version) Compare(other Version) int {
	return v.Version.Compare(other.Version)
}

// ConstraintMap is a map of plugin names to version constraints.
type ConstraintMap map[Name]*semver.Constraints

// ParseConstraintMap parses string map into a PluginConstraintMap.
func ParseConstraintMap(src map[string]string) (ConstraintMap, error) {
	if src == nil {
		return nil, nil
	}
	parsed := make(ConstraintMap)
	for name, version := range src {
		if version == "" {
			return nil, fmt.Errorf("missing plugin version constraint for '%s'", name)
		}
		parsedName, err := ParseName(name)
		if err != nil {
			return nil, err
		}
		constraints, err := semver.NewConstraint(version)
		if err != nil {
			return nil, err
		}
		parsed[parsedName] = constraints
	}
	return parsed, nil
}
