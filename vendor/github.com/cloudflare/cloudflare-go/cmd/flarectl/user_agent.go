package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cloudflare/cloudflare-go"
	"github.com/codegangsta/cli"
)

func formatUserAgentRule(rule cloudflare.UserAgentRule) []string {
	return []string{
		rule.ID,
		rule.Description,
		rule.Mode,
		rule.Configuration.Value,
		strconv.FormatBool(rule.Paused),
	}
}

func userAgentCreate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if err := checkFlags(c, "zone", "mode", "value"); err != nil {
		fmt.Println(err)
		return
	}

	zoneID, err := api.ZoneIDByName(c.String("zone"))
	if err != nil {
		fmt.Println(err)
		return
	}

	userAgentRule := cloudflare.UserAgentRule{
		Description: c.String("description"),
		Mode:        c.String("mode"),
		Paused:      c.Bool("paused"),
		Configuration: cloudflare.UserAgentRuleConfig{
			Target: "ua",
			Value:  c.String("value"),
		},
	}

	resp, err := api.CreateUserAgentRule(zoneID, userAgentRule)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating User-Agent block rule: ", err)
		return
	}

	output := [][]string{
		formatUserAgentRule(resp.Result),
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
}

func userAgentUpdate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}

	if err := checkFlags(c, "zone", "id", "mode", "value"); err != nil {
		return
	}

	zoneID, err := api.ZoneIDByName(c.String("zone"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	userAgentRule := cloudflare.UserAgentRule{
		Description: c.String("description"),
		Mode:        c.String("mode"),
		Paused:      c.Bool("paused"),
		Configuration: cloudflare.UserAgentRuleConfig{
			Target: "ua",
			Value:  c.String("value"),
		},
	}

	resp, err := api.UpdateUserAgentRule(zoneID, c.String("id"), userAgentRule)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error updating User-Agent block rule: ", err)
		return
	}

	output := [][]string{
		formatUserAgentRule(resp.Result),
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
}

func userAgentDelete(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}

	if err := checkFlags(c, "zone", "id"); err != nil {
		return
	}

	zoneID, err := api.ZoneIDByName(c.String("zone"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	resp, err := api.DeleteUserAgentRule(zoneID, c.String("id"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error deleting User-Agent block rule: ", err)
		return
	}

	output := [][]string{
		formatUserAgentRule(resp.Result),
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
}

func userAgentList(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}

	if err := checkFlags(c, "zone", "page"); err != nil {
		return
	}

	zoneID, err := api.ZoneIDByName(c.String("zone"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	resp, err := api.ListUserAgentRules(zoneID, c.Int("page"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error listing User-Agent block rules: ", err)
		return
	}

	output := make([][]string, 0, len(resp.Result))
	for _, rule := range resp.Result {
		output = append(output, formatUserAgentRule(rule))
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
}
