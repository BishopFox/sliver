package db

import (
	"errors"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/db/models"
)

func TestCanaryByDomain(t *testing.T) {
	canary := &models.DNSCanary{
		ImplantName: "canary_lookup_" + models.NewUUID().String(),
		Domain:      "lookup-" + models.NewUUID().String() + ".Canary.Example.",
	}
	if err := Session().Create(canary).Error; err != nil {
		t.Fatalf("create canary: %v", err)
	}
	t.Cleanup(func() {
		if err := Session().Delete(canary).Error; err != nil {
			t.Errorf("delete canary: %v", err)
		}
	})

	for _, domain := range []string{
		canary.Domain,
		strings.ToLower(canary.Domain),
		strings.ToUpper(canary.Domain),
		" " + strings.TrimSuffix(canary.Domain, ".") + " ",
	} {
		t.Run(domain, func(t *testing.T) {
			found, err := CanaryByDomain(domain)
			if err != nil {
				t.Fatalf("lookup canary: %v", err)
			}
			if found.ID != canary.ID.String() || found.Domain != canary.Domain {
				t.Errorf("lookup returned %v, want ID %s and domain %s", found, canary.ID, canary.Domain)
			}
		})
	}
}

func TestCanaryByDomainNotFound(t *testing.T) {
	for _, domain := range []string{"", " \t", "missing-" + models.NewUUID().String() + ".invalid."} {
		t.Run(domain, func(t *testing.T) {
			found, err := CanaryByDomain(domain)
			if !errors.Is(err, ErrRecordNotFound) || found != nil {
				t.Fatalf("lookup returned (%v, %v), want (nil, record not found)", found, err)
			}
		})
	}
}
