package pihole

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

type GroupResponseList struct {
	Data []GroupResponse
}

type GroupResponse struct {
	ID           int64  `json:"id"`
	Enabled      int    `json:"enabled"`
	Name         string `json:"name"`
	DateAdded    int64  `json:"date_added"`
	DateModified int64  `json:"date_modified"`
	Description  string `json:"description"`
}

type GroupList []*Group

type Group struct {
	ID           int64
	Enabled      bool
	Name         string
	DateAdded    time.Time
	DateModified time.Time
	Description  string
}

type GroupUpdateRequest struct {
	Name        string
	Enabled     *bool
	Description string
}

type GroupCreateRequest struct {
	Name        string
	Description string
}

// ToGroup converts a GroupResponseList to a GroupList
func (grl GroupResponseList) ToGroupList() GroupList {
	list := make(GroupList, len(grl.Data))

	for i, g := range grl.Data {
		list[i] = g.ToGroup()
	}

	return list
}

// ToGroup converts a GroupResponse to a Group
func (gr GroupResponse) ToGroup() *Group {
	return &Group{
		ID:           gr.ID,
		Enabled:      gr.Enabled == 1,
		Name:         gr.Name,
		DateAdded:    time.Unix(gr.DateAdded, 0),
		DateModified: time.Unix(gr.DateModified, 0),
		Description:  gr.Description,
	}
}

// ListGroups returns the list of gravity DB groups
func (c Client) ListGroups(ctx context.Context) (GroupList, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: list groups", ErrNotImplementedTokenClient)
	}

	req, err := c.RequestWithSession2(ctx, "GET", "/api/groups", nil)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to retrieve groups, got status code %d", res.StatusCode)
	}

	defer res.Body.Close()
	type Response struct {
		Groups []struct {
			Name      string  `json:"name"`
			Comment   *string `json:"comment"`
			Enabled   bool    `json:"enabled"`
			ID        int64   `json:"id"`
			CreatedAt int64   `json:"date_added"`
			UpdatedAt int64   `json:"date_modified"`
		} `json:"groups"`
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response Response
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, err
	}

	var list GroupList
	for _, v := range response.Groups {
		comment := ""
		if v.Comment != nil {
			comment = *v.Comment
		}

		list = append(list, &Group{
			ID:           v.ID,
			Enabled:      v.Enabled,
			Name:         v.Name,
			DateAdded:    time.Unix(v.UpdatedAt, 0),
			DateModified: time.Unix(v.UpdatedAt, 0),
			Description:  comment,
		})
	}

	return list, nil
}

// GetGroup returns a Pi-hole group by name
func (c Client) GetGroup(ctx context.Context, name string) (*Group, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: get groups", ErrNotImplementedTokenClient)
	}

	groups, err := c.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.Name == name {
			return g, nil
		}
	}
	return nil, NewNotFoundError(fmt.Sprintf("Group with name %q not found", name))
}

// GetGroupByID returns a Pi-hole group by ID
func (c Client) GetGroupByID(ctx context.Context, id int64) (*Group, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: get group", ErrNotImplementedTokenClient)
	}

	groups, err := c.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.ID == id {
			return g, nil
		}
	}

	return nil, NewNotFoundError(fmt.Sprintf("Group with ID %q not found", id))
}

// validName indicates whether the name given to the group is valid
func validGroupName(name string) bool {
	validName := regexp.MustCompile(`^\S*$`)

	return validName.MatchString(name)
}

type GroupBasicResponse struct {
	Success bool
	Message string
}

// CreateGroup creates a group with the passed attributes
func (c Client) CreateGroup(ctx context.Context, gr *GroupCreateRequest) (*Group, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: create group", ErrNotImplementedTokenClient)
	}

	name := strings.TrimSpace(gr.Name)

	if !validGroupName(name) {
		return nil, fmt.Errorf("group names must not contain spaces")
	}

	req, err := c.RequestWithSession2(ctx, "POST", "/api/groups", map[string]any{
		"name":    gr.Name,
		"comment": gr.Description,
	})
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 201 {
		return nil, fmt.Errorf("failed to create group, got status code %d", res.StatusCode)
	}

	return c.GetGroup(ctx, name)
}

// UpdateGroup updates a group resource with the passed attribute
func (c Client) UpdateGroup(ctx context.Context, gr *GroupUpdateRequest) (*Group, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: update group", ErrNotImplementedTokenClient)
	}

	path := fmt.Sprintf("/api/groups/%s", gr.Name)
	req, err := c.RequestWithSession2(ctx, "PUT", path, map[string]any{
		"name":    gr.Name,
		"comment": gr.Description,
		"enabled": gr.Enabled,
	})
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to update group, got status code %d", res.StatusCode)
	}

	return c.GetGroup(ctx, gr.Name)
}

// DeleteGroup deletes a group
func (c Client) DeleteGroup(ctx context.Context, name string) error {
	if c.tokenClient != nil {
		return fmt.Errorf("%w: delete group", ErrNotImplementedTokenClient)
	}

	path := fmt.Sprintf("/api/groups/%s", name)
	req, err := c.RequestWithSession2(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 204 {
		return fmt.Errorf("failed to delete group, got status code %d", res.StatusCode)
	}

	return nil
}
