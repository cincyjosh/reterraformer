package gcp

import (
	"context"

	storageapi "google.golang.org/api/storage/v1"
)

type bucketEnumerator struct{ svc *storageapi.Service }

func newBucketEnumerator(svc *storageapi.Service) *bucketEnumerator {
	return &bucketEnumerator{svc: svc}
}

func (e *bucketEnumerator) TerraformType() string { return "google_storage_bucket" }

func (e *bucketEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	var out []ResourceImport
	req := e.svc.Buckets.List(project)
	for {
		page, err := req.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, b := range page.Items {
			out = append(out, ResourceImport{
				TerraformType: e.TerraformType(),
				TerraformName: SanitizeName(b.Name),
				ImportID:      b.Name,
			})
		}
		if page.NextPageToken == "" {
			break
		}
		req = req.PageToken(page.NextPageToken)
	}
	return out, nil
}
