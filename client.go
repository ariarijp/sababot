package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	mkr "github.com/mackerelio/mackerel-client-go"
)

type myClient struct {
	*mkr.Client
}

func (c *myClient) urlFor(path string) *url.URL {
	newURL, err := url.Parse(c.BaseURL.String())
	if err != nil {
		panic("invalid url passed")
	}

	newURL.Path = path

	return newURL
}

func (c *myClient) fetchMetricNames(hostID string) ([]string, error) {
	req, err := http.NewRequest(
		"GET",
		c.urlFor(fmt.Sprintf("/api/v0/hosts/%s/metric-names", hostID)).String(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.Request(req)
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	var data struct {
		Names []string `json:"names"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data.Names, nil
}

func (c *myClient) fetchLatestMetricValues(host mkr.Host) (map[string]*mkr.MetricValue, error) {
	metricNames, err := c.fetchMetricNames(host.ID)
	if err != nil {
		return nil, err
	}

	metrics, err := c.FetchLatestMetricValues([]string{host.ID}, metricNames)
	if err != nil {
		return nil, err
	}

	return metrics[host.ID], nil
}
