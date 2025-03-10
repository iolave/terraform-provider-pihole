package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/iolave/go-proxmox/pkg/cloudflare"
	"github.com/ryanwholey/terraform-provider-pihole/internal/pihole"
)

// Config defines the configuration options for the Pi-hole client
type Config struct {
	// The Pi-hole URL
	URL string

	// The Pi-hole admin password
	Password string

	// UserAgent for requests
	UserAgent string

	// Pi-hole API token
	APIToken string

	// Custom CA file
	CAFile         string
	CFServiceToken *cloudflare.ServiceToken
}

// Client initializes a new pihole client from the passed configuration
func (c Config) Client(ctx context.Context) (*pihole.Client, error) {
	HttpClient := &http.Client{}
	if c.CAFile != "" {
		certs, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file %q: %v", c.CAFile, err)
		}

		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(certs)
		tlsConfig := &tls.Config{
			RootCAs: rootCAs,
		}

		HttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
	}

	config := pihole.Config{
		URL:            c.URL,
		Password:       c.Password,
		UserAgent:      c.UserAgent,
		APIToken:       c.APIToken,
		Client:         HttpClient,
		CFServiceToken: c.CFServiceToken,
	}

	client := pihole.New(config)

	if err := client.Init(ctx); err != nil {
		return nil, err
	}

	return client, nil
}
