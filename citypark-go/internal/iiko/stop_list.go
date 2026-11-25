package iiko

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type terminalStopList struct {
	TerminalGroupID string `json:"terminalGroupId"`
	Items           []struct {
		ProductID string `json:"productId"`
	} `json:"items"`
}

// FetchStopList returns the product IDs listed for the delivery terminal group.
func (c *Client) FetchStopList(ctx context.Context, terminalGroupID string) (map[string]bool, error) {
	body := struct {
		OrganizationIDs  []string `json:"organizationIds"`
		TerminalGroupIDs []string `json:"terminalGroupsIds,omitempty"`
		ReturnSize       bool     `json:"returnSize"`
	}{OrganizationIDs: []string{c.organizationID}, ReturnSize: true}
	if terminalGroupID != "" {
		body.TerminalGroupIDs = []string{terminalGroupID}
	}

	var out struct {
		Organizations []struct {
			OrganizationID string             `json:"organizationId"`
			Groups         []terminalStopList `json:"items"`
		} `json:"terminalGroupStopLists"`
	}
	if err := c.withToken(ctx, func(token string) error {
		return c.post(ctx, "1/stop_lists", token, body, &out)
	}); err != nil {
		return nil, err
	}

	var matches []terminalStopList
	for _, org := range out.Organizations {
		if !strings.EqualFold(org.OrganizationID, c.organizationID) {
			continue
		}
		for _, group := range org.Groups {
			if terminalGroupID == "" || strings.EqualFold(group.TerminalGroupID, terminalGroupID) {
				matches = append(matches, group)
			}
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("iiko: expected one stop list for the delivery terminal group, got %d", len(matches))
	}
	group := matches[0]
	if group.TerminalGroupID == "" || group.Items == nil {
		return nil, errors.New("iiko: incomplete terminal group stop list")
	}
	stopped := make(map[string]bool, len(group.Items))
	for _, item := range group.Items {
		if item.ProductID == "" {
			return nil, errors.New("iiko: stop-list entry has no product ID")
		}
		stopped[strings.ToLower(item.ProductID)] = true
	}
	return stopped, nil
}
