package gcp

import (
	"context"
	"strings"

	iamapi "google.golang.org/api/iam/v1"
)

type serviceAccountEnumerator struct{ svc *iamapi.Service }

func newServiceAccountEnumerator(svc *iamapi.Service) *serviceAccountEnumerator {
	return &serviceAccountEnumerator{svc: svc}
}

func (e *serviceAccountEnumerator) TerraformType() string { return "google_service_account" }

func (e *serviceAccountEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	var out []ResourceImport
	req := e.svc.Projects.ServiceAccounts.List("projects/" + project)
	for {
		page, err := req.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, sa := range page.Accounts {
			// Email format: "name@project.iam.gserviceaccount.com"
			name := strings.SplitN(sa.Email, "@", 2)[0]
			out = append(out, ResourceImport{
				TerraformType: e.TerraformType(),
				TerraformName: SanitizeName(name),
				ImportID:      sa.Name, // already "projects/{project}/serviceAccounts/{email}"
			})
		}
		if page.NextPageToken == "" {
			break
		}
		req = req.PageToken(page.NextPageToken)
	}
	return out, nil
}
