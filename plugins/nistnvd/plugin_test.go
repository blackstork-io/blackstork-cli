// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package nistnvd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlugin_Schema(t *testing.T) {
	schema := Plugin("1.2.3", nil)
	assert.Equal(t, "blackstork/nist_nvd", schema.Name)
	assert.Equal(t, "1.2.3", schema.Version)
	assert.NotNil(t, schema.DataSources["nist_nvd_cves"])
}
