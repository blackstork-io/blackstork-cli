package client

import "slices"

type CreateSearchJobReq struct {
	ID            string   `url:"id"`
	ExecMode      string   `url:"exec_mode"`
	Search        string   `url:"search"`
	StatusBuckets *int     `url:"status_buckets,omitempty"`
	MaxCount      *int     `url:"max_count,omitempty"`
	RF            []string `url:"rf,omitempty"`
	EarliestTime  *string  `url:"earliest_time,omitempty"`
	LatestTime    *string  `url:"latest_time,omitempty"`
}

func String(s string) *string {
	return &s
}

func Int(i int) *int {
	return &i
}

type CreateSearchJobRes struct {
	Sid string `json:"sid"`
}

type GetSearchJobByIDReq struct {
	ID string
}

type DispatchState string

const (
	DispatchStateQueued         DispatchState = "QUEUED"
	DispatchStateParsing        DispatchState = "PARSING"
	DispatchStateRunning        DispatchState = "RUNNING"
	DispatchStateFinalizing     DispatchState = "FINALIZING"
	DispatchStateDone           DispatchState = "DONE"
	DispatchStatePause          DispatchState = "PAUSE"
	DispatchStateInternalCancel DispatchState = "INTERNAL_CANCEL"
	DispatchStateUserCancel     DispatchState = "USER_CANCEL"
	DispatchStateBadInputCancel DispatchState = "BAD_INPUT_CANCEL"
	DispatchStateQuit           DispatchState = "QUIT"
	DispatchStateFailed         DispatchState = "FAILED"
)

func (d DispatchState) Wait() bool {
	return slices.Contains([]DispatchState{
		DispatchStateQueued,
		DispatchStateParsing,
		DispatchStateRunning,
		DispatchStateFinalizing,
	}, d)
}

func (d DispatchState) Done() bool {
	return DispatchStateDone == d
}

func (d DispatchState) Failed() bool {
	return !d.Wait() && !d.Done()
}

type GetSearchJobByIDRes struct {
	DispatchState DispatchState `json:"dispatchState"`
}

type GetSearchJobResultsReq struct {
	ID         string `url:"-"`
	OutputMode string `url:"output_mode"`
}

type GetSearchJobResultsRes struct {
	Results []any `json:"results"`
}
