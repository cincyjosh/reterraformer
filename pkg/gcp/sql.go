package gcp

import (
	"context"

	sqladminapi "google.golang.org/api/sqladmin/v1beta4"
)

type sqlInstanceEnumerator struct{ svc *sqladminapi.Service }

func newSQLInstanceEnumerator(svc *sqladminapi.Service) *sqlInstanceEnumerator {
	return &sqlInstanceEnumerator{svc: svc}
}

func (e *sqlInstanceEnumerator) TerraformType() string { return "google_sql_database_instance" }

func (e *sqlInstanceEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	var out []ResourceImport
	req := e.svc.Instances.List(project)
	for {
		page, err := req.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, inst := range page.Items {
			out = append(out, ResourceImport{
				TerraformType: e.TerraformType(),
				TerraformName: SanitizeName(inst.Name),
				ImportID:      project + "/" + inst.Name,
			})
		}
		if page.NextPageToken == "" {
			break
		}
		req = req.PageToken(page.NextPageToken)
	}
	return out, nil
}
