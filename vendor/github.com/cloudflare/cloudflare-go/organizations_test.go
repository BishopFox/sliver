package cloudflare

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestOrganizations_ListOrganizations(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/user/organizations", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
"success": true,
"errors": [],
"messages": [],
"result": [
    {
      "id": "01a7362d577a6c3019a474fd6f485823",
      "name": "Cloudflare, Inc.",
      "status": "member",
      "permissions": [
        "#zones:read"
      ],
      "roles": [
        "All Privileges - Super Administrator"
      ]
    }
  ],
"result_info": {
  "page": 1,
  "per_page": 20,
  "count": 1,
  "total_count": 2000
  }
}`)
	})

	user, paginator, err := client.ListOrganizations()

	want := []Organization{{
		ID:          "01a7362d577a6c3019a474fd6f485823",
		Name:        "Cloudflare, Inc.",
		Status:      "member",
		Permissions: []string{"#zones:read"},
		Roles:       []string{"All Privileges - Super Administrator"},
	}}

	if assert.NoError(t, err) {
		assert.Equal(t, user, want)
	}

	wantPagination := ResultInfo{
		Page:    1,
		PerPage: 20,
		Count:   1,
		Total:   2000,
	}
	assert.Equal(t, paginator, wantPagination)
}

func TestOrganizations_OrganizationDetails(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/organizations/01a7362d577a6c3019a474fd6f485823", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": {
    "id": "01a7362d577a6c3019a474fd6f485823",
    "name": "Cloudflare, Inc.",
    "members": [
      {
        "id": "7c5dae5552338874e5053f2534d2767a",
        "name": "John Smith",
        "email": "user@example.com",
        "status": "accepted",
        "roles": [
          {
            "id": "3536bcfad5faccb999b47003c79917fb",
            "name": "Organization Admin",
            "description": "Administrative access to the entire Organization",
            "permissions": [
              "#zones:read"
            ]
          }
        ]
      }
    ],
    "invites": [
      {
        "id": "4f5f0c14a2a41d5063dd301b2f829f04",
        "invited_member_id": "5a7805061c76ada191ed06f989cc3dac",
        "invited_member_email": "user@example.com",
        "organization_id": "5a7805061c76ada191ed06f989cc3dac",
        "organization_name": "Cloudflare, Inc.",
        "roles": [
          {
            "id": "3536bcfad5faccb999b47003c79917fb",
            "name": "Organization Admin",
            "description": "Administrative access to the entire Organization",
            "permissions": [
              "#zones:read"
            ]
          }
        ],
        "invited_by": "user@example.com",
        "invited_on": "2014-01-01T05:20:00Z",
        "expires_on": "2014-01-01T05:20:00Z",
        "status": "accepted"
      }
    ],
    "roles": [
      {
        "id": "3536bcfad5faccb999b47003c79917fb",
        "name": "Organization Admin",
        "description": "Administrative access to the entire Organization",
        "permissions": [
          "#zones:read"
        ]
      }
    ]
  }
}`)
	})

	organizationDetails, err := client.OrganizationDetails("01a7362d577a6c3019a474fd6f485823")

	invitedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	expiresOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")

	want := OrganizationDetails{
		ID:   "01a7362d577a6c3019a474fd6f485823",
		Name: "Cloudflare, Inc.",
		Members: []OrganizationMember{{
			ID:     "7c5dae5552338874e5053f2534d2767a",
			Name:   "John Smith",
			Email:  "user@example.com",
			Status: "accepted",
			Roles: []OrganizationRole{{
				ID:          "3536bcfad5faccb999b47003c79917fb",
				Name:        "Organization Admin",
				Description: "Administrative access to the entire Organization",
				Permissions: []string{
					"#zones:read",
				},
			}},
		}},
		Invites: []OrganizationInvite{{

			ID:                 "4f5f0c14a2a41d5063dd301b2f829f04",
			InvitedMemberID:    "5a7805061c76ada191ed06f989cc3dac",
			InvitedMemberEmail: "user@example.com",
			OrganizationID:     "5a7805061c76ada191ed06f989cc3dac",
			OrganizationName:   "Cloudflare, Inc.",
			Roles: []OrganizationRole{{

				ID:          "3536bcfad5faccb999b47003c79917fb",
				Name:        "Organization Admin",
				Description: "Administrative access to the entire Organization",
				Permissions: []string{
					"#zones:read",
				}}},
			InvitedBy: "user@example.com",
			InvitedOn: &invitedOn,
			ExpiresOn: &expiresOn,
			Status:    "accepted",
		}},
		Roles: []OrganizationRole{{
			ID:          "3536bcfad5faccb999b47003c79917fb",
			Name:        "Organization Admin",
			Description: "Administrative access to the entire Organization",
			Permissions: []string{"#zones:read"},
		}},
	}

	if assert.NoError(t, err) {
		assert.Equal(t, organizationDetails, want)
	}
}

func TestOrganizations_OrganizationMembers(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/organizations/01a7362d577a6c3019a474fd6f485823/members", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": [
    {
      "id": "7c5dae5552338874e5053f2534d2767a",
      "name": "John Smith",
      "email": "user@example.com",
      "status": "accepted",
      "roles": [
        {
          "id": "3536bcfad5faccb999b47003c79917fb",
          "name": "Organization Admin",
          "description": "Administrative access to the entire Organization",
          "permissions": [
            "#zones:read"
          ]
        }
      ]
    }
  ],
  "result_info": {
    "page": 1,
    "per_page": 20,
    "count": 1,
    "total_count": 2000
  }
}`)
	})

	members, paginator, err := client.OrganizationMembers("01a7362d577a6c3019a474fd6f485823")

	want := []OrganizationMember{{
		ID:     "7c5dae5552338874e5053f2534d2767a",
		Name:   "John Smith",
		Email:  "user@example.com",
		Status: "accepted",
		Roles: []OrganizationRole{{
			ID:          "3536bcfad5faccb999b47003c79917fb",
			Name:        "Organization Admin",
			Description: "Administrative access to the entire Organization",
			Permissions: []string{
				"#zones:read",
			},
		}},
	}}

	if assert.NoError(t, err) {
		assert.Equal(t, members, want)
	}

	wantPagination := ResultInfo{
		Page:    1,
		PerPage: 20,
		Count:   1,
		Total:   2000,
	}
	assert.Equal(t, paginator, wantPagination)
}

func TestOrganizations_OrganizationInvites(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/organizations/01a7362d577a6c3019a474fd6f485823/invites", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": [
    {
      "id": "4f5f0c14a2a41d5063dd301b2f829f04",
      "invited_member_id": "5a7805061c76ada191ed06f989cc3dac",
      "invited_member_email": "user@example.com",
      "organization_id": "5a7805061c76ada191ed06f989cc3dac",
      "organization_name": "Cloudflare, Inc.",
      "roles": [
        {
          "id": "3536bcfad5faccb999b47003c79917fb",
          "name": "Organization Admin",
          "description": "Administrative access to the entire Organization",
          "permissions": [
            "#zones:read"
          ]
        }
      ],
      "invited_by": "user@example.com",
      "invited_on": "2014-01-01T05:20:00Z",
      "expires_on": "2014-01-01T05:20:00Z",
      "status": "accepted"
    }
  ],
  "result_info": {
    "page": 1,
    "per_page": 20,
    "count": 1,
    "total_count": 2000
  }
}`)
	})

	members, paginator, err := client.OrganizationInvites("01a7362d577a6c3019a474fd6f485823")

	invitedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	expiresOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")

	want := []OrganizationInvite{{
		ID:                 "4f5f0c14a2a41d5063dd301b2f829f04",
		InvitedMemberID:    "5a7805061c76ada191ed06f989cc3dac",
		InvitedMemberEmail: "user@example.com",
		OrganizationID:     "5a7805061c76ada191ed06f989cc3dac",
		OrganizationName:   "Cloudflare, Inc.",
		Roles: []OrganizationRole{{
			ID:          "3536bcfad5faccb999b47003c79917fb",
			Name:        "Organization Admin",
			Description: "Administrative access to the entire Organization",
			Permissions: []string{
				"#zones:read",
			}}},
		InvitedBy: "user@example.com",
		InvitedOn: &invitedOn,
		ExpiresOn: &expiresOn,
		Status:    "accepted",
	}}

	if assert.NoError(t, err) {
		assert.Equal(t, members, want)
	}

	wantPagination := ResultInfo{
		Page:    1,
		PerPage: 20,
		Count:   1,
		Total:   2000,
	}
	assert.Equal(t, paginator, wantPagination)
}

func TestOrganizations_OrganizationRoles(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/organizations/01a7362d577a6c3019a474fd6f485823/roles", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": [
    {
      "id": "3536bcfad5faccb999b47003c79917fb",
      "name": "Organization Admin",
      "description": "Administrative access to the entire Organization",
      "permissions": [
        "#zones:read"
      ]
    }
  ],
  "result_info": {
    "page": 1,
    "per_page": 20,
    "count": 1,
    "total_count": 2000
  }
}`)
	})

	members, paginator, err := client.OrganizationRoles("01a7362d577a6c3019a474fd6f485823")

	want := []OrganizationRole{{
		ID:          "3536bcfad5faccb999b47003c79917fb",
		Name:        "Organization Admin",
		Description: "Administrative access to the entire Organization",
		Permissions: []string{
			"#zones:read",
		}}}

	if assert.NoError(t, err) {
		assert.Equal(t, members, want)
	}

	wantPagination := ResultInfo{
		Page:    1,
		PerPage: 20,
		Count:   1,
		Total:   2000,
	}
	assert.Equal(t, paginator, wantPagination)
}
