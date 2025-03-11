package pihole

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ryanwholey/go-pihole"
)

type DNSRecordsListResponse struct {
	Data [][]string
}

// ToDNSRecordList converts a DNSRecordsListResponse into a DNSRecordList object.
func (rr DNSRecordsListResponse) ToDNSRecordList() DNSRecordList {
	list := DNSRecordList{}

	for _, record := range rr.Data {
		list = append(list, DNSRecord{
			Domain: record[0],
			IP:     record[1],
		})
	}

	return list
}

type DNSRecordList = pihole.DNSRecordList
type DNSRecord = pihole.DNSRecord

// ListDNSRecords Returns the list of custom DNS records configured in pihole
func (c Client) ListDNSRecords(ctx context.Context) (DNSRecordList, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: list dns records", ErrNotImplementedTokenClient)
	}

	req, err := c.RequestWithSession2(ctx, "GET", "/api/config/dns/hosts", nil)
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
				Hosts []string `json:"hosts"`
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

	var list DNSRecordList
	for _, v := range response.Config.DNS.Hosts {
		splitted := strings.Split(v, " ")
		if len(splitted) != 2 {
			return nil, fmt.Errorf("failed to parse dns records")
		}
		list = append(list, pihole.DNSRecord{
			IP:     splitted[0],
			Domain: splitted[1],
		})
	}

	return list, nil
}

type CreateDNSRecordResponse struct {
	Success bool
	Message string
}

// CreateDNSRecord creates a pihole DNS record entry
func (c Client) CreateDNSRecord(ctx context.Context, record *DNSRecord) (*DNSRecord, error) {
	if c.tokenClient != nil {
		return c.tokenClient.LocalDNS.Create(ctx, record.Domain, record.IP)
	}

	cfg := strings.Join([]string{record.IP, record.Domain}, "%20")
	req, err := c.RequestWithSession2(ctx, "PUT", fmt.Sprintf("/api/config/dns/hosts/%s", cfg), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 201 {
		return nil, fmt.Errorf("failed to create dns records, got status code %d", res.StatusCode)
	}

	return record, nil
}

// GetDNSRecord searches the pihole local DNS records for the passed domain and returns a result if found
func (c Client) GetDNSRecord(ctx context.Context, domain string) (*DNSRecord, error) {
	if c.tokenClient != nil {
		record, err := c.tokenClient.LocalDNS.Get(ctx, domain)
		if err != nil {
			if errors.Is(err, pihole.ErrorLocalDNSNotFound) {
				return nil, NewNotFoundError(fmt.Sprintf("dns record with domain %q not found", domain))
			}

			return nil, err
		}

		return record, nil
	}

	list, err := c.ListDNSRecords(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range list {
		if r.Domain == domain {
			return &r, nil
		}
	}

	return nil, NewNotFoundError(fmt.Sprintf("record %q not found", domain))
}

// DeleteDNSRecord deletes a pihole local DNS record by domain name
func (c Client) DeleteDNSRecord(ctx context.Context, domain string) error {
	if c.tokenClient != nil {
		return c.tokenClient.LocalDNS.Delete(ctx, domain)
	}

	record, err := c.GetDNSRecord(ctx, domain)
	if err != nil {
		return err
	}

	cfg := strings.Join([]string{record.IP, record.Domain}, "%20")
	req, err := c.RequestWithSession2(ctx, "DELETE", fmt.Sprintf("/api/config/dns/hosts/%s", cfg), nil)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 204 {
		return fmt.Errorf("failed to delete dns records, got status code %d", res.StatusCode)
	}

	return nil
}
