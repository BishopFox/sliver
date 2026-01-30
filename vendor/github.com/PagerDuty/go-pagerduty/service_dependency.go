package pagerduty

import (
	"context"
)

// ServiceDependency represents a relationship between a business and technical service
type ServiceDependency struct {
	ID                string      `json:"id,omitempty"`
	Type              string      `json:"type,omitempty"`
	SupportingService *ServiceObj `json:"supporting_service,omitempty"`
	DependentService  *ServiceObj `json:"dependent_service,omitempty"`
}

// ServiceObj represents a service object in service relationship
type ServiceObj struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

// ListServiceDependencies represents a list of dependencies for a service
type ListServiceDependencies struct {
	Relationships []*ServiceDependency `json:"relationships,omitempty"`
}

// ListBusinessServiceDependencies lists dependencies of a business service.
//
// Deprecated: Use ListBusinessServiceDependenciesWithContext instead.
func (c *Client) ListBusinessServiceDependencies(businessServiceID string) (*ListServiceDependencies, error) {
	return c.ListBusinessServiceDependenciesWithContext(context.Background(), businessServiceID)
}

// ListBusinessServiceDependenciesWithContext lists dependencies of a business service.
func (c *Client) ListBusinessServiceDependenciesWithContext(ctx context.Context, businessServiceID string) (*ListServiceDependencies, error) {
	resp, err := c.get(ctx, "/service_dependencies/business_services/"+businessServiceID, nil)
	if err != nil {
		return nil, err
	}

	var result ListServiceDependencies
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListTechnicalServiceDependencies lists dependencies of a technical service.
//
// Deprecated: Use ListTechnicalServiceDependenciesWithContext instead.
func (c *Client) ListTechnicalServiceDependencies(serviceID string) (*ListServiceDependencies, error) {
	return c.ListTechnicalServiceDependenciesWithContext(context.Background(), serviceID)
}

// ListTechnicalServiceDependenciesWithContext lists dependencies of a technical service.
func (c *Client) ListTechnicalServiceDependenciesWithContext(ctx context.Context, serviceID string) (*ListServiceDependencies, error) {
	resp, err := c.get(ctx, "/service_dependencies/technical_services/"+serviceID, nil)
	if err != nil {
		return nil, err
	}

	var result ListServiceDependencies
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AssociateServiceDependencies Create new dependencies between two services.
//
// Deprecated: Use AssociateServiceDependenciesWithContext instead.
func (c *Client) AssociateServiceDependencies(dependencies *ListServiceDependencies) (*ListServiceDependencies, error) {
	return c.AssociateServiceDependenciesWithContext(context.Background(), dependencies)
}

// AssociateServiceDependenciesWithContext Create new dependencies between two services.
func (c *Client) AssociateServiceDependenciesWithContext(ctx context.Context, dependencies *ListServiceDependencies) (*ListServiceDependencies, error) {
	resp, err := c.post(ctx, "/service_dependencies/associate", dependencies, nil)
	if err != nil {
		return nil, err
	}

	var result ListServiceDependencies
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DisassociateServiceDependencies Disassociate dependencies between two services.
//
// Deprecated: Use DisassociateServiceDependenciesWithContext instead.
func (c *Client) DisassociateServiceDependencies(dependencies *ListServiceDependencies) (*ListServiceDependencies, error) {
	return c.DisassociateServiceDependenciesWithContext(context.Background(), dependencies)
}

// DisassociateServiceDependenciesWithContext Disassociate dependencies between two services.
func (c *Client) DisassociateServiceDependenciesWithContext(ctx context.Context, dependencies *ListServiceDependencies) (*ListServiceDependencies, error) {
	resp, err := c.post(ctx, "/service_dependencies/disassociate", dependencies, nil)
	if err != nil {
		return nil, err
	}

	var result ListServiceDependencies
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
