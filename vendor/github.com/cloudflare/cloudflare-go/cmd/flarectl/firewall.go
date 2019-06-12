package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"

	"github.com/cloudflare/cloudflare-go"
	"github.com/codegangsta/cli"
)

func formatAccessRule(rule cloudflare.AccessRule) []string {
	return []string{
		rule.ID,
		rule.Configuration.Value,
		rule.Scope.Type,
		rule.Mode,
		rule.Notes,
	}
}

func firewallAccessRules(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}

	organizationID, zoneID, err := getScope(c)
	if err != nil {
		return
	}

	// Create an empty access rule for searching for rules
	rule := cloudflare.AccessRule{
		Configuration: getConfiguration(c),
	}
	if c.String("scope-type") != "" {
		rule.Scope.Type = c.String("scope-type")
	}
	if c.String("notes") != "" {
		rule.Notes = c.String("notes")
	}
	if c.String("mode") != "" {
		rule.Mode = c.String("mode")
	}

	var response *cloudflare.AccessRuleListResponse
	switch {
	case organizationID != "":
		response, err = api.ListOrganizationAccessRules(organizationID, rule, 1)
	case zoneID != "":
		response, err = api.ListZoneAccessRules(zoneID, rule, 1)
	default:
		response, err = api.ListUserAccessRules(rule, 1)
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	totalPages := response.ResultInfo.TotalPages
	rules := make([]cloudflare.AccessRule, 0, response.ResultInfo.Total)
	rules = append(rules, response.Result...)
	if totalPages > 1 {
		for page := 2; page <= totalPages; page++ {
			switch {
			case organizationID != "":
				response, err = api.ListOrganizationAccessRules(organizationID, rule, page)
			case zoneID != "":
				response, err = api.ListZoneAccessRules(zoneID, rule, page)
			default:
				response, err = api.ListUserAccessRules(rule, page)
			}
			if err != nil {
				fmt.Println(err)
				return
			}
			rules = append(rules, response.Result...)
		}
	}

	output := make([][]string, 0, len(rules))
	for _, rule := range rules {
		output = append(output, formatAccessRule(rule))
	}
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")
}

func firewallAccessRuleCreate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "mode", "value"); err != nil {
		fmt.Println(err)
		return
	}
	organizationID, zoneID, err := getScope(c)
	if err != nil {
		return
	}
	configuration := getConfiguration(c)
	mode := c.String("mode")
	notes := c.String("notes")

	rule := cloudflare.AccessRule{
		Configuration: configuration,
		Mode:          mode,
		Notes:         notes,
	}

	var (
		rules       []cloudflare.AccessRule
		errCreating = "error creating firewall access rule"
	)

	switch {
	case organizationID != "":
		resp, err := api.CreateOrganizationAccessRule(organizationID, rule)
		if err != nil {
			errors.Wrap(err, errCreating)
		}
		rules = append(rules, resp.Result)
	case zoneID != "":
		resp, err := api.CreateZoneAccessRule(zoneID, rule)
		if err != nil {
			errors.Wrap(err, errCreating)
		}
		rules = append(rules, resp.Result)
	default:
		resp, err := api.CreateUserAccessRule(rule)
		if err != nil {
			errors.Wrap(err, errCreating)
		}
		rules = append(rules, resp.Result)

	}

	output := make([][]string, 0, len(rules))
	for _, rule := range rules {
		output = append(output, formatAccessRule(rule))
	}
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")
}

func firewallAccessRuleUpdate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "id"); err != nil {
		fmt.Println(err)
		return
	}
	id := c.String("id")
	organizationID, zoneID, err := getScope(c)
	if err != nil {
		return
	}
	mode := c.String("mode")
	notes := c.String("notes")

	rule := cloudflare.AccessRule{
		Mode:  mode,
		Notes: notes,
	}

	var (
		rules       []cloudflare.AccessRule
		errUpdating = "error updating firewall access rule"
	)
	switch {
	case organizationID != "":
		resp, err := api.UpdateOrganizationAccessRule(organizationID, id, rule)
		if err != nil {
			errors.Wrap(err, errUpdating)
		}
		rules = append(rules, resp.Result)
	case zoneID != "":
		resp, err := api.UpdateZoneAccessRule(zoneID, id, rule)
		if err != nil {
			errors.Wrap(err, errUpdating)
		}
		rules = append(rules, resp.Result)
	default:
		resp, err := api.UpdateUserAccessRule(id, rule)
		if err != nil {
			errors.Wrap(err, errUpdating)
		}
		rules = append(rules, resp.Result)
	}

	output := make([][]string, 0, len(rules))
	for _, rule := range rules {
		output = append(output, formatAccessRule(rule))
	}
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")

}

func firewallAccessRuleCreateOrUpdate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "mode", "value"); err != nil {
		fmt.Println(err)
		return
	}
	organizationID, zoneID, err := getScope(c)
	if err != nil {
		return
	}
	configuration := getConfiguration(c)
	mode := c.String("mode")
	notes := c.String("notes")

	// Look for an existing record
	rule := cloudflare.AccessRule{
		Configuration: configuration,
	}
	var response *cloudflare.AccessRuleListResponse
	switch {
	case organizationID != "":
		response, err = api.ListOrganizationAccessRules(organizationID, rule, 1)
	case zoneID != "":
		response, err = api.ListZoneAccessRules(zoneID, rule, 1)
	default:
		response, err = api.ListUserAccessRules(rule, 1)
	}
	if err != nil {
		fmt.Println("Error creating or updating firewall access rule:", err)
		return
	}

	rule.Mode = mode
	rule.Notes = notes
	if len(response.Result) > 0 {
		for _, r := range response.Result {
			if mode == "" {
				rule.Mode = r.Mode
			}
			if notes == "" {
				rule.Notes = r.Notes
			}
			switch {
			case organizationID != "":
				_, err = api.UpdateOrganizationAccessRule(organizationID, r.ID, rule)
			case zoneID != "":
				_, err = api.UpdateZoneAccessRule(zoneID, r.ID, rule)
			default:
				_, err = api.UpdateUserAccessRule(r.ID, rule)
			}
			if err != nil {
				fmt.Println("Error updating firewall access rule:", err)
			}
		}
	} else {
		switch {
		case organizationID != "":
			_, err = api.CreateOrganizationAccessRule(organizationID, rule)
		case zoneID != "":
			_, err = api.CreateZoneAccessRule(zoneID, rule)
		default:
			_, err = api.CreateUserAccessRule(rule)
		}
		if err != nil {
			fmt.Println("Error creating firewall access rule:", err)
		}
	}
}

func firewallAccessRuleDelete(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "id"); err != nil {
		fmt.Println(err)
		return
	}
	ruleID := c.String("id")

	organizationID, zoneID, err := getScope(c)
	if err != nil {
		return
	}

	var (
		rules       []cloudflare.AccessRule
		errDeleting = "error deleting firewall access rule"
	)
	switch {
	case organizationID != "":
		resp, err := api.DeleteOrganizationAccessRule(organizationID, ruleID)
		if err != nil {
			errors.Wrap(err, errDeleting)
		}
		rules = append(rules, resp.Result)
	case zoneID != "":
		resp, err := api.DeleteZoneAccessRule(zoneID, ruleID)
		if err != nil {
			errors.Wrap(err, errDeleting)
		}
		rules = append(rules, resp.Result)
	default:
		resp, err := api.DeleteUserAccessRule(ruleID)
		if err != nil {
			errors.Wrap(err, errDeleting)
		}
		rules = append(rules, resp.Result)
	}
	if err != nil {
		fmt.Println("Error deleting firewall access rule:", err)
	}

	output := make([][]string, 0, len(rules))
	for _, rule := range rules {
		output = append(output, formatAccessRule(rule))
	}
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")
}

func getScope(c *cli.Context) (string, string, error) {
	var organization, organizationID string
	if c.String("organization") != "" {
		organization = c.String("organization")
		organizations, _, err := api.ListOrganizations()
		if err != nil {
			fmt.Println(err)
			return "", "", err
		}
		for _, org := range organizations {
			if org.Name == organization {
				organizationID = org.ID
				break
			}
		}
		if organizationID == "" {
			err := errors.New("Organization could not be found")
			fmt.Println(err)
			return "", "", err
		}
	}

	var zone, zoneID string
	if c.String("zone") != "" {
		zone = c.String("zone")
		id, err := api.ZoneIDByName(zone)
		if err != nil {
			fmt.Println(err)
			return "", "", err
		}
		zoneID = id
	}

	if zoneID != "" && organizationID != "" {
		err := errors.New("Cannot specify both --zone and --organization")
		fmt.Println(err)
		return "", "", err
	}

	return organizationID, zoneID, nil
}

func getConfiguration(c *cli.Context) cloudflare.AccessRuleConfiguration {
	configuration := cloudflare.AccessRuleConfiguration{}
	if c.String("value") != "" {
		ip := net.ParseIP(c.String("value"))
		_, cidr, cidrErr := net.ParseCIDR(c.String("value"))
		_, asnErr := strconv.ParseInt(c.String("value"), 10, 32)
		if ip != nil {
			configuration.Target = "ip"
			configuration.Value = ip.String()
		} else if cidrErr == nil {
			cidr.IP = cidr.IP.Mask(cidr.Mask)
			configuration.Target = "ip_range"
			configuration.Value = cidr.String()
		} else if asnErr == nil {
			configuration.Target = "asn"
			configuration.Value = c.String("value")
		} else {
			configuration.Target = "country"
			configuration.Value = c.String("value")
		}
	}
	return configuration
}
