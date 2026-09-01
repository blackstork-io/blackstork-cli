// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package client implements the VirusTotal API client.
package client

import (
	"fmt"
	"net/url"
	"time"
)

type GetUserAPIUsageReq struct {
	User      string `url:"-"`
	StartDate *Date  `url:"start_date,omitempty"`
	EndDate   *Date  `url:"end_date,omitempty"`
}

type GetGroupAPIUsageReq struct {
	Group     string `url:"-"`
	StartDate *Date  `url:"start_date,omitempty"`
	EndDate   *Date  `url:"end_date,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type ErrorRes struct {
	Error Error `json:"error"`
}

type GetUserAPIUsageRes struct {
	Data map[string]any `json:"data"`
}

type GetGroupAPIUsageRes struct {
	Data map[string]any `json:"data"`
}

type Date struct {
	time.Time
}

func (d Date) EncodeValues(key string, v *url.Values) error {
	v.Add(key, d.Format("20060102"))
	return nil
}
