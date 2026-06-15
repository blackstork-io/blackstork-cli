// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package client

type Option func(*client)

func WithBaseURL(baseURL string) Option {
	return func(c *client) {
		c.baseURL = baseURL
	}
}

func WithOrgID(orgID string) Option {
	return func(c *client) {
		c.orgID = orgID
	}
}

func WithAPIKey(apiKey string) Option {
	return func(c *client) {
		c.apiKey = apiKey
	}
}
