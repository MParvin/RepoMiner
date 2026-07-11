package httpclient

// Accessor methods for provider reconfiguration.

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// Token returns the configured auth token.
func (c *Client) Token() string { return c.token }
