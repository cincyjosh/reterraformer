package gcp

import (
	"context"
	"strings"

	computeapi "google.golang.org/api/compute/v1"
)

// --- Networks ---

type networkEnumerator struct{ svc *computeapi.Service }

func newNetworkEnumerator(svc *computeapi.Service) *networkEnumerator {
	return &networkEnumerator{svc: svc}
}

func (e *networkEnumerator) TerraformType() string { return "google_compute_network" }

func (e *networkEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	var out []ResourceImport
	err := e.svc.Networks.List(project).Pages(ctx, func(page *computeapi.NetworkList) error {
		for _, n := range page.Items {
			out = append(out, ResourceImport{
				TerraformType: e.TerraformType(),
				TerraformName: SanitizeName(n.Name),
				ImportID:      project + "/" + n.Name,
			})
		}
		return nil
	})
	return out, err
}

// --- Subnetworks ---

type subnetworkEnumerator struct{ svc *computeapi.Service }

func newSubnetworkEnumerator(svc *computeapi.Service) *subnetworkEnumerator {
	return &subnetworkEnumerator{svc: svc}
}

func (e *subnetworkEnumerator) TerraformType() string { return "google_compute_subnetwork" }

func (e *subnetworkEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	var out []ResourceImport
	err := e.svc.Subnetworks.AggregatedList(project).Pages(ctx, func(page *computeapi.SubnetworkAggregatedList) error {
		for key, item := range page.Items {
			if !strings.HasPrefix(key, "regions/") {
				continue
			}
			region := strings.TrimPrefix(key, "regions/")
			for _, s := range item.Subnetworks {
				out = append(out, ResourceImport{
					TerraformType: e.TerraformType(),
					TerraformName: SanitizeName(s.Name),
					ImportID:      project + "/" + region + "/" + s.Name,
				})
			}
		}
		return nil
	})
	return out, err
}

// --- Instances ---

type instanceEnumerator struct{ svc *computeapi.Service }

func newInstanceEnumerator(svc *computeapi.Service) *instanceEnumerator {
	return &instanceEnumerator{svc: svc}
}

func (e *instanceEnumerator) TerraformType() string { return "google_compute_instance" }

func (e *instanceEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	var out []ResourceImport
	err := e.svc.Instances.AggregatedList(project).Pages(ctx, func(page *computeapi.InstanceAggregatedList) error {
		for key, item := range page.Items {
			if !strings.HasPrefix(key, "zones/") {
				continue
			}
			zone := strings.TrimPrefix(key, "zones/")
			for _, inst := range item.Instances {
				out = append(out, ResourceImport{
					TerraformType: e.TerraformType(),
					TerraformName: SanitizeName(inst.Name),
					ImportID:      project + "/" + zone + "/" + inst.Name,
				})
			}
		}
		return nil
	})
	return out, err
}

// --- Firewalls ---

type firewallEnumerator struct{ svc *computeapi.Service }

func newFirewallEnumerator(svc *computeapi.Service) *firewallEnumerator {
	return &firewallEnumerator{svc: svc}
}

func (e *firewallEnumerator) TerraformType() string { return "google_compute_firewall" }

func (e *firewallEnumerator) Enumerate(ctx context.Context, project string) ([]ResourceImport, error) {
	var out []ResourceImport
	err := e.svc.Firewalls.List(project).Pages(ctx, func(page *computeapi.FirewallList) error {
		for _, f := range page.Items {
			out = append(out, ResourceImport{
				TerraformType: e.TerraformType(),
				TerraformName: SanitizeName(f.Name),
				ImportID:      project + "/" + f.Name,
			})
		}
		return nil
	})
	return out, err
}
