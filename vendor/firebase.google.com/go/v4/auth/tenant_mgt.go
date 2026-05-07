// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"firebase.google.com/go/v4/internal"
	"google.golang.org/api/iterator"
)

// Tenant represents a tenant in a multi-tenant application.
//
// Multi-tenancy support requires Google Cloud's Identity Platform (GCIP). To learn more about GCIP,
// including pricing and features, see https://cloud.google.com/identity-platform.
//
// Before multi-tenancy can be used in a Google Cloud Identity Platform project, tenants must be
// enabled in that project via the Cloud Console UI.
//
// A tenant configuration provides information such as the display name, tenant identifier and email
// authentication configuration. For OIDC/SAML provider configuration management, TenantClient
// instances should be used instead of a Tenant to retrieve the list of configured IdPs on a tenant.
// When configuring these providers, note that tenants will inherit whitelisted domains and
// authenticated redirect URIs of their parent project.
//
// All other settings of a tenant will also be inherited. These will need to be managed from the
// Cloud Console UI.
type Tenant struct {
	ID                    string             `json:"name"`
	DisplayName           string             `json:"displayName"`
	AllowPasswordSignUp   bool               `json:"allowPasswordSignup"`
	EnableEmailLinkSignIn bool               `json:"enableEmailLinkSignin"`
	EnableAnonymousUsers  bool               `json:"enableAnonymousUser"`
	MultiFactorConfig     *MultiFactorConfig `json:"mfaConfig"`
}

// TenantClient is used for managing users, configuring SAML/OIDC providers, and generating email
// links for specific tenants.
//
// Before multi-tenancy can be used in a Google Cloud Identity Platform project, tenants must be
// enabled in that project via the Cloud Console UI.
//
// Each tenant contains its own identity providers, settings and users. TenantClient enables
// managing users and SAML/OIDC configurations of specific tenants. It also supports verifying ID
// tokens issued to users who are signed into specific tenants.
//
// TenantClient instances for a specific tenantID can be instantiated by calling
// [TenantManager.AuthForTenant(tenantID)].
type TenantClient struct {
	*baseClient
}

// TenantID returns the ID of the tenant to which this TenantClient instance belongs.
func (tc *TenantClient) TenantID() string {
	return tc.tenantID
}

// TenantManager is the interface used to manage tenants in a multi-tenant application.
//
// This supports creating, updating, listing, deleting the tenants of a Firebase project. It also
// supports creating new TenantClient instances scoped to specific tenant IDs.
type TenantManager struct {
	base       *baseClient
	endpoint   string
	projectID  string
	httpClient *internal.HTTPClient
}

func newTenantManager(client *internal.HTTPClient, conf *internal.AuthConfig, base *baseClient) *TenantManager {
	return &TenantManager{
		base:       base,
		endpoint:   base.tenantMgtEndpoint,
		projectID:  conf.ProjectID,
		httpClient: client,
	}
}

// AuthForTenant creates a new TenantClient scoped to a given tenantID.
func (tm *TenantManager) AuthForTenant(tenantID string) (*TenantClient, error) {
	if tenantID == "" {
		return nil, errors.New("tenantID must not be empty")
	}

	return &TenantClient{
		baseClient: tm.base.withTenantID(tenantID),
	}, nil
}

// Tenant returns the tenant with the given ID.
func (tm *TenantManager) Tenant(ctx context.Context, tenantID string) (*Tenant, error) {
	if tenantID == "" {
		return nil, errors.New("tenantID must not be empty")
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("/tenants/%s", tenantID),
	}
	var tenant Tenant
	if _, err := tm.makeRequest(ctx, req, &tenant); err != nil {
		return nil, err
	}

	tenant.ID = extractResourceID(tenant.ID)
	return &tenant, nil
}

// CreateTenant creates a new tenant with the given options.
func (tm *TenantManager) CreateTenant(ctx context.Context, tenant *TenantToCreate) (*Tenant, error) {
	if tenant == nil {
		return nil, errors.New("tenant must not be nil")
	}
	if err := tenant.validate(); err != nil {
		return nil, err
	}
	req := &internal.Request{
		Method: http.MethodPost,
		URL:    "/tenants",
		Body:   internal.NewJSONEntity(tenant.ensureParams()),
	}
	var result Tenant
	if _, err := tm.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	result.ID = extractResourceID(result.ID)
	return &result, nil
}

// UpdateTenant updates an existing tenant with the given options.
func (tm *TenantManager) UpdateTenant(ctx context.Context, tenantID string, tenant *TenantToUpdate) (*Tenant, error) {
	if tenantID == "" {
		return nil, errors.New("tenantID must not be empty")
	}
	if tenant == nil {
		return nil, errors.New("tenant must not be nil")
	}
	if err := tenant.validate(); err != nil {
		return nil, err
	}
	mask := tenant.params.UpdateMask()
	if len(mask) == 0 {
		return nil, errors.New("no parameters specified in the update request")
	}

	req := &internal.Request{
		Method: http.MethodPatch,
		URL:    fmt.Sprintf("/tenants/%s", tenantID),
		Body:   internal.NewJSONEntity(tenant.params),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("updateMask", strings.Join(mask, ",")),
		},
	}
	var result Tenant
	if _, err := tm.makeRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	result.ID = extractResourceID(result.ID)
	return &result, nil
}

// DeleteTenant deletes the tenant with the given ID.
func (tm *TenantManager) DeleteTenant(ctx context.Context, tenantID string) error {
	if tenantID == "" {
		return errors.New("tenantID must not be empty")
	}

	req := &internal.Request{
		Method: http.MethodDelete,
		URL:    fmt.Sprintf("/tenants/%s", tenantID),
	}
	_, err := tm.makeRequest(ctx, req, nil)
	return err
}

// Tenants returns an iterator over tenants in the project.
//
// If nextPageToken is empty, the iterator will start at the beginning. Otherwise,
// iterator starts after the token.
func (tm *TenantManager) Tenants(ctx context.Context, nextPageToken string) *TenantIterator {
	it := &TenantIterator{
		ctx: ctx,
		tm:  tm,
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(
		it.fetch,
		func() int { return len(it.tenants) },
		func() interface{} { b := it.tenants; it.tenants = nil; return b })
	it.pageInfo.MaxSize = maxConfigs
	it.pageInfo.Token = nextPageToken
	return it
}

func (tm *TenantManager) makeRequest(ctx context.Context, req *internal.Request, v interface{}) (*internal.Response, error) {
	if tm.projectID == "" {
		return nil, errors.New("project id not available")
	}

	req.URL = fmt.Sprintf("%s/projects/%s%s", tm.endpoint, tm.projectID, req.URL)
	return tm.httpClient.DoAndUnmarshal(ctx, req, v)
}

const (
	tenantDisplayNameKey       = "displayName"
	allowPasswordSignUpKey     = "allowPasswordSignup"
	enableEmailLinkSignInKey   = "enableEmailLinkSignin"
	enableAnonymousUser        = "enableAnonymousUser"
	multiFactorConfigTenantKey = "mfaConfig"
)

// TenantToCreate represents the options used to create a new tenant.
type TenantToCreate struct {
	params nestedMap
}

// DisplayName sets the display name of the new tenant.
func (t *TenantToCreate) DisplayName(name string) *TenantToCreate {
	return t.set(tenantDisplayNameKey, name)
}

// AllowPasswordSignUp enables or disables email sign-in provider.
func (t *TenantToCreate) AllowPasswordSignUp(allow bool) *TenantToCreate {
	return t.set(allowPasswordSignUpKey, allow)
}

// EnableEmailLinkSignIn enables or disables email link sign-in.
//
// Disabling this makes the password required for email sign-in.
func (t *TenantToCreate) EnableEmailLinkSignIn(enable bool) *TenantToCreate {
	return t.set(enableEmailLinkSignInKey, enable)
}

// EnableAnonymousUsers enables or disables anonymous authentication.
func (t *TenantToCreate) EnableAnonymousUsers(enable bool) *TenantToCreate {
	return t.set(enableAnonymousUser, enable)
}

// MultiFactorConfig configures the tenant's multi-factor settings
func (t *TenantToCreate) MultiFactorConfig(multiFactorConfig MultiFactorConfig) *TenantToCreate {
	return t.set(multiFactorConfigTenantKey, multiFactorConfig)
}

func (t *TenantToCreate) set(key string, value interface{}) *TenantToCreate {
	t.ensureParams().Set(key, value)
	return t
}

func (t *TenantToCreate) ensureParams() nestedMap {
	if t.params == nil {
		t.params = make(nestedMap)
	}

	return t.params
}
func (t *TenantToCreate) validate() error {
	req := make(map[string]interface{})
	for k, v := range t.params {
		req[k] = v
	}
	val, ok := req[multiFactorConfigTenantKey]
	if ok {
		multiFactorConfig, ok := val.(MultiFactorConfig)
		if !ok {
			return fmt.Errorf("invalid type for MultiFactorConfig: %s", req[multiFactorConfigProjectKey])
		}
		if err := multiFactorConfig.validate(); err != nil {
			return err
		}
	}
	return nil
}

// TenantToUpdate represents the options used to update an existing tenant.
type TenantToUpdate struct {
	params nestedMap
}

// DisplayName sets the display name of the new tenant.
func (t *TenantToUpdate) DisplayName(name string) *TenantToUpdate {
	return t.set(tenantDisplayNameKey, name)
}

// AllowPasswordSignUp enables or disables email sign-in provider.
func (t *TenantToUpdate) AllowPasswordSignUp(allow bool) *TenantToUpdate {
	return t.set(allowPasswordSignUpKey, allow)
}

// EnableEmailLinkSignIn enables or disables email link sign-in.
//
// Disabling this makes the password required for email sign-in.
func (t *TenantToUpdate) EnableEmailLinkSignIn(enable bool) *TenantToUpdate {
	return t.set(enableEmailLinkSignInKey, enable)
}

// EnableAnonymousUsers enables or disables anonymous authentication.
func (t *TenantToUpdate) EnableAnonymousUsers(enable bool) *TenantToUpdate {
	return t.set(enableAnonymousUser, enable)
}

// MultiFactorConfig configures the tenant's multi-factor settings
func (t *TenantToUpdate) MultiFactorConfig(multiFactorConfig MultiFactorConfig) *TenantToUpdate {
	return t.set(multiFactorConfigTenantKey, multiFactorConfig)
}

func (t *TenantToUpdate) set(key string, value interface{}) *TenantToUpdate {
	if t.params == nil {
		t.params = make(nestedMap)
	}

	t.params.Set(key, value)
	return t
}

func (t *TenantToUpdate) validate() error {
	req := make(map[string]interface{})
	for k, v := range t.params {
		req[k] = v
	}
	val, ok := req[multiFactorConfigTenantKey]
	if ok {
		multiFactorConfig, ok := val.(MultiFactorConfig)
		if !ok {
			return fmt.Errorf("invalid type for MultiFactorConfig: %s", req[multiFactorConfigProjectKey])
		}
		if err := multiFactorConfig.validate(); err != nil {
			return err
		}
	}
	return nil
}

// TenantIterator is an iterator over tenants.
type TenantIterator struct {
	tm       *TenantManager
	ctx      context.Context
	nextFunc func() error
	pageInfo *iterator.PageInfo
	tenants  []*Tenant
}

// PageInfo supports pagination.
func (it *TenantIterator) PageInfo() *iterator.PageInfo {
	return it.pageInfo
}

// Next returns the next Tenant. The error value of [iterator.Done] is
// returned if there are no more results. Once Next returns [iterator.Done], all
// subsequent calls will return [iterator.Done].
func (it *TenantIterator) Next() (*Tenant, error) {
	if err := it.nextFunc(); err != nil {
		return nil, err
	}

	tenant := it.tenants[0]
	it.tenants = it.tenants[1:]
	return tenant, nil
}

func (it *TenantIterator) fetch(pageSize int, pageToken string) (string, error) {
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
	}
	if pageToken != "" {
		params["pageToken"] = pageToken
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    "/tenants",
		Opts: []internal.HTTPOption{
			internal.WithQueryParams(params),
		},
	}

	var result struct {
		Tenants       []Tenant `json:"tenants"`
		NextPageToken string   `json:"nextPageToken"`
	}
	if _, err := it.tm.makeRequest(it.ctx, req, &result); err != nil {
		return "", err
	}

	for i := range result.Tenants {
		result.Tenants[i].ID = extractResourceID(result.Tenants[i].ID)
		it.tenants = append(it.tenants, &result.Tenants[i])
	}

	it.pageInfo.Token = result.NextPageToken
	return result.NextPageToken, nil
}
