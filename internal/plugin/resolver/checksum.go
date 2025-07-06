package resolver

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Checksum for plugin binaries and archives.
// It contains the object name, os, arch, and the hash sum value.
//
// Format in string: '<object>:<os>:<arch>:<base64-sha256-sum>'.
//
// Example: 'archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8='.
type Checksum struct {
	Object string
	OS     string
	Arch   string
	Sum    []byte
}

func encodeChecksums(w io.Writer, checksums []Checksum) error {
	writer := bufio.NewWriter(w)
	for _, c := range checksums {
		if _, err := writer.WriteString(c.String() + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func decodeChecksums(r io.Reader) ([]Checksum, error) {
	var checksums []Checksum
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var c Checksum
		if err := c.UnmarshalText(scanner.Bytes()); err != nil {
			return nil, err
		}
		checksums = append(checksums, c)
	}
	return checksums, nil
}

func (c Checksum) Compare(other Checksum) int {
	cmp := strings.Compare(c.Object, other.Object)
	if cmp != 0 {
		return cmp
	}
	cmp = strings.Compare(c.OS, other.OS)
	if cmp != 0 {
		return cmp
	}
	cmp = strings.Compare(c.Arch, other.Arch)
	if cmp != 0 {
		return cmp
	}
	return bytes.Compare(c.Sum, other.Sum)
}

func (c Checksum) Match(list []Checksum) bool {
	for _, other := range list {
		if c.Compare(other) == 0 {
			return true
		}
	}
	return false
}

func (c Checksum) String() string {
	return strings.Join([]string{
		c.Object,
		c.OS,
		c.Arch,
		base64.StdEncoding.EncodeToString(c.Sum),
	}, ":")
}

func (c Checksum) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *Checksum) UnmarshalText(data []byte) error {
	raw := string(data)
	parts := strings.Split(raw, ":")
	if len(parts) != 4 {
		return fmt.Errorf("invalid checksum format: %s", raw)
	}
	sum, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return fmt.Errorf("failed to decode checksum: %w", err)
	}
	*c = Checksum{
		Object: parts[0],
		OS:     parts[1],
		Arch:   parts[2],
		Sum:    sum,
	}
	return nil
}

func (c Checksum) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(c.String())), nil
}

func (c *Checksum) UnmarshalJSON(data []byte) error {
	raw, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("failed to unquote checksum: %w", err)
	}
	return c.UnmarshalText([]byte(raw))
}
