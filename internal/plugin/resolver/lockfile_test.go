package resolver

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockCheckResult_IsInstallRequired(t *testing.T) {
	locks := LockFile{
		Plugins: []PluginLock{
			{
				Name:    mustName(t, "ns/name"),
				Version: mustVersion(t, "1.0.0"),
			},
			{
				Name:    mustName(t, "ns/name2"),
				Version: mustVersion(t, "2.0.0"),
			},
			{
				Name:    mustName(t, "ns/name3"),
				Version: mustVersion(t, "3.0.0"),
			},
		},
	}
	constraints := ConstraintMap{
		mustName(t, "ns/name"):  mustConstraint(t, ">1.0.0"),
		mustName(t, "ns/name2"): mustConstraint(t, "<=2.0.0"),
		mustName(t, "ns/name4"): mustConstraint(t, "3.0.0"),
	}
	result := locks.Check(constraints)
	assert.True(t, result.IsInstallRequired(), "expected install required")
	assert.Len(t, result.Missing, 1, "expected 1 missing")
	assert.Len(t, result.Mismatch, 1, "expected 1 mismatch")
	assert.Len(t, result.Removed, 1, "expected 1 removed")
}

func TestReadLockFile(t *testing.T) {
	buf := bytes.NewBufferString(`{
		"plugins": [
			{
				"name": "ns/name",
				"version": "1.0.0",
				"checksums": ["binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="]
			}
		]
	}`)
	locks, err := ReadLockFile(buf)
	assert.NoError(t, err)
	assert.Len(t, locks.Plugins, 1)
	assert.Equal(t, "ns/name", locks.Plugins[0].Name.String())
	assert.Equal(t, "1.0.0", locks.Plugins[0].Version.String())
	assert.Len(t, locks.Plugins[0].Checksums, 1)
	assert.Equal(t, "binary", locks.Plugins[0].Checksums[0].Object)
	assert.Equal(t, "darwin", locks.Plugins[0].Checksums[0].OS)
	assert.Equal(t, "arm64", locks.Plugins[0].Checksums[0].Arch)
	assert.Equal(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=", base64.StdEncoding.EncodeToString(locks.Plugins[0].Checksums[0].Sum))
}

func TestSaveLockFile(t *testing.T) {
	locks := &LockFile{
		Plugins: []PluginLock{
			{
				Name:    mustName(t, "ns/name"),
				Version: mustVersion(t, "1.0.0"),
				Checksums: []Checksum{
					{
						Object: "binary",
						OS:     "darwin",
						Arch:   "arm64",
						Sum:    mustBase64(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					},
				},
			},
		},
	}
	buf := bytes.NewBuffer(nil)
	err := SaveLockFile(buf, locks)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"plugins": [
			{
				"name": "ns/name",
				"version": "1.0.0",
				"checksums": ["binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="]
			}
		]
	}`, buf.String())
}
