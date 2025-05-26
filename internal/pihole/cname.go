package pihole

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	pihole "github.com/ryanwholey/go-pihole"
)

type CNAMERecordsListResponse struct {
	Data [][]string
}

// ToCNAMERecordList converts a CNAMERecordsListResponse into a CNAMERecordsList object.
func (rr CNAMERecordsListResponse) ToCNAMERecordList() CNAMERecordList {
	list := CNAMERecordList{}

	for _, record := range rr.Data {
		list = append(list, CNAMERecord{
			Domain: record[0],
			Target: record[1],
		})
	}

	return list
}

type CNAMERecord = pihole.CNAMERecord
type CNAMERecordList = pihole.CNAMERecordList

// ListCNAMERecords returns a list of the configured CNAME Pi-hole records
func (c Client) ListCNAMERecords(ctx context.Context) (CNAMERecordList, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: list dns records", ErrNotImplementedTokenClient)
	}

	req, err := c.RequestWithSession2(ctx, "GET", "/api/config/dns/cnameRecords", nil)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to retrieve dns records, got status code %d", res.StatusCode)
	}

	defer res.Body.Close()
	type Response struct {
		Config struct {
			DNS struct {
				Hosts []string `json:"cnameRecords"`
			} `json:"dns"`
		} `json:"config"`
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response Response
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, err
	}

	var list pihole.CNAMERecordList
	for _, v := range response.Config.DNS.Hosts {
		splitted := strings.Split(v, ",")
		if len(splitted) != 2 {
			return nil, fmt.Errorf("failed to parse dns records")
		}
		list = append(list, pihole.CNAMERecord{
			Domain: splitted[0],
			Target: splitted[1],
		})
	}

	return list, nil
}

// GetCNAMERecord returns a CNAMERecord for the passed domain if found
func (c Client) GetCNAMERecord(ctx context.Context, domain string) (*CNAMERecord, error) {
	if c.tokenClient != nil {
		record, err := c.tokenClient.LocalCNAME.Get(ctx, domain)
		if err != nil {
			if errors.Is(err, pihole.ErrorLocalCNAMENotFound) {
				return nil, NewNotFoundError(fmt.Sprintf("cname with domain %q not found", domain))
			}

			return nil, err
		}

		return record, nil
	}

	list, err := c.ListCNAMERecords(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range list {
		if r.Domain == domain {
			return &r, nil
		}
	}

	return nil, NewNotFoundError(fmt.Sprintf("cname with domain %q not found", domain))
}

type CreateCNAMERecordResponse struct {
	Success bool
	Message string
}

// CreateCNAMERecord handles CNAME record creation
func (c Client) CreateCNAMERecord(ctx context.Context, record *CNAMERecord) (*CNAMERecord, error) {
	if c.tokenClient != nil {
		return c.tokenClient.LocalCNAME.Create(ctx, record.Domain, record.Target)
	}

	cfg := strings.Join([]string{record.Domain, record.Target}, "%2C")
	req, err := c.RequestWithSession2(ctx, "PUT", fmt.Sprintf("/api/config/dns/cnameRecords/%s", cfg), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 201 {
		return nil, fmt.Errorf("failed to create CNAME records, got status code %d", res.StatusCode)
	}

	return record, nil
}

// DeleteCNAMERecord handles CNAME record deletion for the passed domain
func (c Client) DeleteCNAMERecord(ctx context.Context, domain string) error {
	if c.tokenClient != nil {
		return c.tokenClient.LocalCNAME.Delete(ctx, domain)
	}

	record, err := c.GetCNAMERecord(ctx, domain)
	if err != nil {
		return err
	}

	cfg := strings.Join([]string{record.Domain, record.Target}, "%2C")
	req, err := c.RequestWithSession2(ctx, "DELETE", fmt.Sprintf("/api/config/dns/cnameRecords/%s", cfg), nil)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 204 {
		return fmt.Errorf("failed to delete CNAME records, got status code %d", res.StatusCode)
	}

	return nil
}
