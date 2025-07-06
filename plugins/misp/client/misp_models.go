package client

type RestSearchEventsRequest struct {
	Page              *int64   `json:"page,omitempty"`
	Limit             *int64   `json:"limit,omitempty"`
	Value             string   `json:"value,omitempty"`
	Type              string   `json:"type,omitempty"`
	Category          string   `json:"category,omitempty"`
	Org               string   `json:"org,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	EventTags         []string `json:"event_tags,omitempty"`
	SearchAll         string   `json:"searchall,omitempty"`
	From              *string  `json:"from,omitempty"`
	To                *string  `json:"to,omitempty"`
	Last              *string  `json:"last,omitempty"`
	EventID           string   `json:"eventid,omitempty"`
	WithAttachments   *bool    `json:"withAttachments,omitempty"`
	SharingGroups     []string `json:"sharinggroup,omitempty"`
	Metadata          *bool    `json:"metadata,omitempty"`
	UUID              string   `json:"uuid,omitempty"`
	IncludeSightingdb *bool    `json:"includeSightingdb,omitempty"`
	ThreatLevelID     string   `json:"threat_level_id,omitempty"`
}

type RestSearchEventsResponse struct {
	Response []EventResponse `json:"response"`
}

type EventResponse struct {
	Event Event `json:"Event"`
}

type Event struct {
	ID                 string `json:"id"`
	OrgId              string `json:"org_id"`
	Distribution       string `json:"distribution"`
	Info               string `json:"info"`
	OrgcId             string `json:"orgc_id"`
	Uuid               string `json:"uuid"`
	Date               string `json:"date"`
	Published          bool   `json:"published"`
	Analysis           string `json:"analysis"`
	AttributeCount     string `json:"attribute_count"`
	Timestamp          string `json:"timestamp"`
	SharingGroupId     string `json:"sharing_group_id"`
	ProposalEmailLock  bool   `json:"proposal_email_lock"`
	Locked             bool   `json:"locked"`
	ThreatLevelId      string `json:"threat_level_id"`
	PublishTimestamp   string `json:"publish_timestamp"`
	SightingTimestamp  string `json:"sighting_timestamp"`
	DisableCorrelation bool   `json:"disable_correlation"`
}

type AddEventReportRequest struct {
	Uuid           string  `json:"uuid"`
	EventId        string  `json:"event_id"`
	Name           string  `json:"name"`
	Content        string  `json:"content"`
	Distribution   *string `json:"distribution"`
	SharingGroupId *string `json:"sharing_group_id"`
	Timestamp      *string `json:"timestamp"`
	Deleted        bool    `json:"deleted"`
}

type EventReport struct {
	Id             string  `json:"id"`
	Uuid           string  `json:"uuid"`
	EventId        string  `json:"event_id"`
	Name           string  `json:"name"`
	Content        string  `json:"content"`
	Distribution   string  `json:"distribution"`
	SharingGroupId *string `json:"sharing_group_id"`
	Timestamp      *string `json:"timestamp"`
	Deleted        bool    `json:"deleted"`
}

type AddEventReportResponse struct {
	EventReport EventReport `json:"EventReport"`
}
