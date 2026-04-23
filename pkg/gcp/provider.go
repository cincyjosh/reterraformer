package gcp

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2/google"
	computeapi "google.golang.org/api/compute/v1"
	containerapi "google.golang.org/api/container/v1"
	iamapi "google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	sqladminapi "google.golang.org/api/sqladmin/v1beta4"
	storageapi "google.golang.org/api/storage/v1"
)

// NewHTTPClient returns an OAuth2 HTTP client using Application Default Credentials.
func NewHTTPClient(ctx context.Context) (*http.Client, error) {
	return google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
}

// NewRegistry creates API service clients and returns all supported GCP enumerators.
func NewRegistry(client *http.Client) (map[string]Enumerator, error) {
	ctx := context.Background()
	opt := option.WithHTTPClient(client)

	computeSvc, err := computeapi.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("compute service: %w", err)
	}
	storageSvc, err := storageapi.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("storage service: %w", err)
	}
	containerSvc, err := containerapi.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("container service: %w", err)
	}
	iamSvc, err := iamapi.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("iam service: %w", err)
	}
	sqladminSvc, err := sqladminapi.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("sqladmin service: %w", err)
	}

	return map[string]Enumerator{
		"google_compute_network":       newNetworkEnumerator(computeSvc),
		"google_compute_subnetwork":    newSubnetworkEnumerator(computeSvc),
		"google_compute_instance":      newInstanceEnumerator(computeSvc),
		"google_compute_firewall":      newFirewallEnumerator(computeSvc),
		"google_storage_bucket":        newBucketEnumerator(storageSvc),
		"google_container_cluster":     newClusterEnumerator(containerSvc),
		"google_service_account":       newServiceAccountEnumerator(iamSvc),
		"google_sql_database_instance": newSQLInstanceEnumerator(sqladminSvc),
	}, nil
}
