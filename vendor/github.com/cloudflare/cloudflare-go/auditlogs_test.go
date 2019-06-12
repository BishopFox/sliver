package cloudflare

import (
	"strings"
	"testing"
)

func TestAuditLogFilterStringify(t *testing.T) {
	filter := AuditLogFilter{
		ID: "aaaa",
	}
	if !strings.Contains(filter.String(), "&id=aaaa") {
		t.Fatalf("Did not properly stringify the id field: %s", filter.String())
	}

	filter.ActorIP = "1.1.1.1"
	if !strings.Contains(filter.String(), "&actor.ip=1.1.1.1") {
		t.Fatalf("Did not properly stringify the actorip field: %s", filter.String())
	}

	filter.ZoneName = "ejj.io"
	if !strings.Contains(filter.String(), "&zone.name=ejj.io") {
		t.Fatalf("Did not properly stringify the zone.name field: %s", filter.String())
	}

	filter.ActorEmail = "admin@admin.com"
	if !strings.Contains(filter.String(), "&actor.email=admin@admin.com") {
		t.Fatalf("Did not properly stringify the actor.email field: %s", filter.String())
	}

	filter.Direction = "direction"
	if !strings.Contains(filter.String(), "&direction=direction") {
		t.Fatalf("Did not properly stringify the direction field: %s", filter.String())
	}

	filter.Since = "10-2-2018"
	if !strings.Contains(filter.String(), "&since=10-2-2018") {
		t.Fatalf("Did not properly stringify the since field: %s", filter.String())
	}

	filter.Before = "10-2-2018"
	if !strings.Contains(filter.String(), "&before=10-2-2018") {
		t.Fatalf("Did not properly stringify the before field: %s", filter.String())
	}

	filter.PerPage = 10000
	if !strings.Contains(filter.String(), "&per_page=10000") {
		t.Fatalf("Did not properly stringify the per_page field: %s", filter.String())
	}

	filter.Page = 3
	if !strings.Contains(filter.String(), "&page=3") {
		t.Fatalf("Did not properly stringify the page field: %s", filter.String())
	}
}
