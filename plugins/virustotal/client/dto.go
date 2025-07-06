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
	v.Add(key, d.Time.Format("20060102"))
	return nil
}
