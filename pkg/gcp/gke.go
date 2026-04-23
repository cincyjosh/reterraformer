package gcp

import (
	"context"

	containerapi "google.golang.org/api/container/v1"
)

type clusterEnumerator struct{ svc *containerapi.Service }

func newClusterEnumerator(svc *containerapi.Service) *clusterEnumerator {
	return &clusterEnumerator{svc: svc}
}

func (e *clusterEnumerator) TerraformType() string { return "google_container_cluster" }

func (e *clusterEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	// "-" as location lists clusters across all regions and zones.
	resp, err := e.svc.Projects.Locations.Clusters.List("projects/" + project + "/locations/-").Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	var out []ResourceImport
	for _, c := range resp.Clusters {
		out = append(out, ResourceImport{
			TerraformType: e.TerraformType(),
			TerraformName: SanitizeName(c.Name),
			ImportID:      project + "/" + c.Location + "/" + c.Name,
		})
	}
	return out, nil
}
