package gateway

import "net/http"

type restGateway struct {
	c *http.Client
}

