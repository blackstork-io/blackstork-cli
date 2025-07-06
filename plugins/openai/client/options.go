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
