package client

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func String(s string) *string {
	return &s
}

func Int(i int) *int {
	return &i
}

type IntList []int

func (list IntList) EncodeValues(key string, v *url.Values) error {
	if len(list) == 0 {
		return nil
	}
	dst := make([]string, len(list))
	for i, id := range list {
		dst[i] = strconv.Itoa(id)
	}
	v.Add(key, strings.Join(dst, ","))
	return nil
}

type StringList []string

func (list StringList) EncodeValues(key string, v *url.Values) error {
	if len(list) == 0 {
		return nil
	}
	v.Add(key, strings.Join(list, ","))
	return nil
}

type Error struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (err *Error) Error() string {
	return fmt.Sprintf("status: %s message: %s", err.Status, err.Message)
}

type ListCasesReq struct {
	Page           int     `url:"page"`
	PerPage        *int    `url:"per_page,omitempty"`
	CaseIDs        IntList `url:"case_ids,omitempty"`
	CaseCustomerID *int    `url:"case_customer_id,omitempty"`
	CaseOwnerID    *int    `url:"case_owner_id,omitempty"`
	CaseSeverityID *int    `url:"case_severity_id,omitempty"`
	CaseStateID    *int    `url:"case_state_id,omitempty"`
	CaseSocID      *string `url:"case_soc_id,omitempty"`
	Sort           *string `url:"sort,omitempty"`
	StartOpenDate  *string `url:"start_open_date,omitempty"`
	EndOpenDate    *string `url:"end_open_date,omitempty"`
}

type ListCasesRes struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	Data    *CasesData `json:"data"`
}

type CasesData struct {
	CurrentPage int   `json:"current_page"`
	LastPage    int   `json:"last_page"`
	NextPage    *int  `json:"next_page"`
	Total       int   `json:"total"`
	Cases       []any `json:"cases"`
}

type ListAlertsReq struct {
	Page                  int        `url:"page"`
	PerPage               *int       `url:"per_page,omitempty"`
	Sort                  *string    `url:"sort,omitempty"`
	AlertIDs              IntList    `url:"alert_ids,omitempty"`
	AlertTags             StringList `url:"alert_tags,omitempty"`
	AlertSource           *string    `url:"alert_source,omitempty"`
	CaseID                *int       `url:"case_id,omitempty"`
	AlertOwnerID          *int       `url:"alert_owner_id,omitempty"`
	AlertStatusID         *int       `url:"alert_status_id,omitempty"`
	AlertSeverityID       *int       `url:"alert_severity_id,omitempty"`
	AlertClassificationID *int       `url:"alert_classification_id,omitempty"`
	AlertCustomerID       *int       `url:"alert_customer_id,omitempty"`
	AlertStartDate        *string    `url:"alert_start_date,omitempty"`
	AlertEndDate          *string    `url:"alert_end_date,omitempty"`
}

type ListAlertsRes struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    *AlertsData `json:"data"`
}

type AlertsData struct {
	CurrentPage int   `json:"current_page"`
	LastPage    int   `json:"last_page"`
	NextPage    *int  `json:"next_page"`
	Total       int   `json:"total"`
	Alerts      []any `json:"alerts"`
}
