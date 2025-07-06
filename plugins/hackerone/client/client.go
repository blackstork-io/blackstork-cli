package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
)

func String(s string) *string {
	return &s
}

func Bool(b bool) *bool {
	return &b
}

func Int(i int) *int {
	return &i
}

type GetAllReportsReq struct {
	PageSize                          *int              `url:"page[size],omitempty"`
	PageNumber                        *int              `url:"page[number],omitempty"`
	Sort                              *string           `url:"sort,omitempty"`
	FilterProgram                     []string          `url:"filter[program][],omitempty"`
	FilterInboxIDs                    []int             `url:"filter[inbox_ids][],omitempty"`
	FilterReporter                    []string          `url:"filter[reporter][],omitempty"`
	FilterAssignee                    []string          `url:"filter[assignee][],omitempty"`
	FilterState                       []string          `url:"filter[state][],omitempty"`
	FilterID                          []int             `url:"filter[id][],omitempty"`
	FilterWeaknessID                  []int             `url:"filter[weakness_id][],omitempty"`
	FilterSeverity                    []string          `url:"filter[severity][],omitempty"`
	FilterHackerPublished             *bool             `url:"filter[hacker_published],omitempty"`
	FilterCreatedAtGT                 *time.Time        `url:"filter[created_at__gt],omitempty"`
	FilterCreatedAtLT                 *time.Time        `url:"filter[created_at__lt],omitempty"`
	FilterSubmittedAtGT               *time.Time        `url:"filter[submitted_at__gt],omitempty"`
	FilterSubmittedAtLT               *time.Time        `url:"filter[submitted_at__lt],omitempty"`
	FilterTriagedAtGT                 *time.Time        `url:"filter[triaged_at__gt],omitempty"`
	FilterTriagedAtLT                 *time.Time        `url:"filter[triaged_at__lt],omitempty"`
	FilterTriagedAtNull               *bool             `url:"filter[triaged_at__null],omitempty"`
	FilterClosedAtGT                  *time.Time        `url:"filter[closed_at__gt],omitempty"`
	FilterClosedAtLT                  *time.Time        `url:"filter[closed_at__lt],omitempty"`
	FilterClosedAtNull                *bool             `url:"filter[closed_at__null],omitempty"`
	FilterDisclosedAtGT               *time.Time        `url:"filter[disclosed_at__gt],omitempty"`
	FilterDisclosedAtLT               *time.Time        `url:"filter[disclosed_at__lt],omitempty"`
	FilterDisclosedAtNull             *bool             `url:"filter[disclosed_at__null],omitempty"`
	FilterReporterAgreedOnGoingPublic *bool             `url:"filter[reporter_agreed_on_going_public],omitempty"`
	FilterBountyAwardedAtGT           *time.Time        `url:"filter[bounty_awarded_at__gt],omitempty"`
	FilterBountyAwardedAtLT           *time.Time        `url:"filter[bounty_awarded_at__lt],omitempty"`
	FilterBountyAwardedAtNull         *bool             `url:"filter[bounty_awarded_at__null],omitempty"`
	FilterSwagAwardedAtGT             *time.Time        `url:"filter[swag_awarded_at__gt],omitempty"`
	FilterSwagAwardedAtLT             *time.Time        `url:"filter[swag_awarded_at__lt],omitempty"`
	FilterSwagAwardedAtNull           *bool             `url:"filter[swag_awarded_at__null],omitempty"`
	FilterLastReportActivityAtGT      *time.Time        `url:"filter[last_report_activity_at__gt],omitempty"`
	FilterLastReportActivityAtLT      *time.Time        `url:"filter[last_report_activity_at__lt],omitempty"`
	FilterFirstProgramActivityAtGT    *time.Time        `url:"filter[first_program_activity_at__gt],omitempty"`
	FilterFirstProgramActivityAtLT    *time.Time        `url:"filter[first_program_activity_at__lt],omitempty"`
	FilterFirstProgramActivityAtNull  *bool             `url:"filter[first_program_activity_at__null],omitempty"`
	FilterLastProgramActivityAtGT     *time.Time        `url:"filter[last_program_activity_at__gt],omitempty"`
	FilterLastProgramActivityAtLT     *time.Time        `url:"filter[last_program_activity_at__lt],omitempty"`
	FilterLastProgramActivityAtNull   *bool             `url:"filter[last_program_activity_at__null],omitempty"`
	FilterLastActivityAtGT            *time.Time        `url:"filter[last_activity_at__gt],omitempty"`
	FilterLastActivityAtLT            *time.Time        `url:"filter[last_activity_at__lt],omitempty"`
	FilterLastPublicActivityAtGT      *time.Time        `url:"filter[last_public_activity_at__gt],omitempty"`
	FilterLastPublicActivityAtLT      *time.Time        `url:"filter[last_public_activity_at__lt],omitempty"`
	FilterKeyword                     *string           `url:"filter[keyword],omitempty"`
	FilterCustomFields                map[string]string `url:"filter[custom_fields][],omitempty"`
}

type GetAllReportsRes struct {
	Data []any `json:"data"`
}

type client struct {
	url string
	usr string
	tkn string
}

type Client interface {
	GetAllReports(ctx context.Context, req *GetAllReportsReq) (*GetAllReportsRes, error)
}

func New(user, token string) Client {
	return &client{
		url: "https://api.hackerone.com",
		usr: user,
		tkn: token,
	}
}

func (c *client) auth(r *http.Request) {
	r.SetBasicAuth(c.usr, c.tkn)
}

func (c *client) GetAllReports(ctx context.Context, req *GetAllReportsReq) (*GetAllReportsRes, error) {
	u, err := url.Parse(c.url + "/v1/reports")
	if err != nil {
		return nil, err
	}
	q, err := query.Values(req)
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(r)
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hackerone client returned status code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	var data GetAllReportsRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
