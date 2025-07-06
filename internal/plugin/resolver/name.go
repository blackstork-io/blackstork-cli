package resolver

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	namespaceIdx = 0
	shortNameIdx = 1
)

// Name of a plugin structured as '<namespace>/<name>'.
type Name [2]string

// Namespace returns the namespace part of the plugin name.
func (n Name) Namespace() string {
	return n[namespaceIdx]
}

// Short returns the short name part of the plugin name.
func (n Name) Short() string {
	return n[shortNameIdx]
}

// String returns the full plugin name in the form '<namespace>/<name>'.
func (n Name) String() string {
	return fmt.Sprintf("%s/%s", n[namespaceIdx], n[shortNameIdx])
}

// Compare compares the plugin name with another plugin name.
func (n Name) Compare(other Name) int {
	cmp := strings.Compare(n.Namespace(), other.Namespace())
	if cmp != 0 {
		return cmp
	}
	return strings.Compare(n.Short(), other.Short())
}

// MarshalJSON returns the JSON representation of the plugin name in '<namespace>/<name>' format.
func (n Name) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(n.String())), nil
}

// UnmarshalJSON parses a JSON string into a PluginName from '<namespace>/<name>' format.
func (n *Name) UnmarshalJSON(data []byte) error {
	raw, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("failed to unquote plugin name: %w", err)
	}
	name, err := ParseName(raw)
	if err != nil {
		return err
	}
	*n = Name(name)
	return nil
}

// ParseName parses a plugin name from '<namespace>/<name>' format.
func ParseName(name string) (Name, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || len(parts[namespaceIdx]) == 0 || len(parts[shortNameIdx]) == 0 {
		return [2]string{}, fmt.Errorf("plugin name '%s' is not in the form '<namespace>/<name>'", name)
	}
	return Name(parts), nil
}
