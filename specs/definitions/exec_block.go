// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

// ExecBlock is implemented by blocks powered by plugin components.
type ExecBlock interface {
	isExecBlock()

	GetSource() *ExecBlockDef
	GetSourceKind() string

	GetMeta() *MetaBlock
	GetRunner() string
	GetName() string

	// all exec blocks are detachable
	isDetachableBlock()
}
