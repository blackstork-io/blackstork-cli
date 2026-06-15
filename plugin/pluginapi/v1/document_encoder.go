// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package pluginapiv1

import (
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

func EncodeDocument(
	templateName string,
	content plugin.Content,
	dataCtx plugindata.Data,
) *Document {
	doc := &Document{
		TemplateName: templateName,
		Content:      EncodeContent(content),
		Data:         EncodeData(dataCtx),
	}
	return doc
}
