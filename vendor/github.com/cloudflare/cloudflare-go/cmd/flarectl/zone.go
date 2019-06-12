package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/codegangsta/cli"
)

func zoneCerts(*cli.Context) {
}

func zoneKeyless(*cli.Context) {
}

func zoneRailgun(*cli.Context) {
}

func zoneCreate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "zone"); err != nil {
		return
	}
	zone := c.String("zone")
	jumpstart := c.Bool("jumpstart")
	orgID := c.String("org-id")
	zoneType := c.String("type")
	var org cloudflare.Organization
	if orgID != "" {
		org.ID = orgID
	}

	if zoneType != "partial" {
		zoneType = "full"
	}

	_, err := api.CreateZone(zone, jumpstart, org, zoneType)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("%s", err))
		return
	}
}

func zoneCheck(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "zone"); err != nil {
		return
	}
	zone := c.String("zone")

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	res, err := api.ZoneActivationCheck(zoneID)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", res.Messages[0].Message)
}

func zoneList(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	zones, err := api.ListZones()
	if err != nil {
		fmt.Println(err)
		return
	}
	output := make([][]string, 0, len(zones))
	for _, z := range zones {
		output = append(output, []string{
			z.ID,
			z.Name,
			z.Plan.Name,
			z.Status,
		})
	}
	writeTable(output, "ID", "Name", "Plan", "Status")
}

func zoneInfo(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	var zone string
	if len(c.Args()) > 0 {
		zone = c.Args()[0]
	} else if c.String("zone") != "" {
		zone = c.String("zone")
	} else {
		cli.ShowSubcommandHelp(c)
		return
	}
	zones, err := api.ListZones(zone)
	if err != nil {
		fmt.Println(err)
		return
	}
	output := make([][]string, 0, len(zones))
	for _, z := range zones {
		var nameservers []string
		if len(z.VanityNS) > 0 {
			nameservers = z.VanityNS
		} else {
			nameservers = z.NameServers
		}
		output = append(output, []string{
			z.ID,
			z.Name,
			z.Plan.Name,
			z.Status,
			strings.Join(nameservers, ", "),
			fmt.Sprintf("%t", z.Paused),
			z.Type,
		})
	}
	writeTable(output, "ID", "Zone", "Plan", "Status", "Name Servers", "Paused", "Type")
}

func zonePlan(*cli.Context) {
}

func zoneSettings(*cli.Context) {
}

func zoneCachePurge(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		cli.ShowSubcommandHelp(c)
		return
	}

	if err := checkFlags(c, "zone"); err != nil {
		cli.ShowSubcommandHelp(c)
		return
	}

	zoneName := c.String("zone")
	zoneID, err := api.ZoneIDByName(c.String("zone"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var resp cloudflare.PurgeCacheResponse

	// Purge everything
	if c.Bool("everything") {
		resp, err = api.PurgeEverything(zoneID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error purging all from zone %q: %s\n", zoneName, err)
			return
		}
	} else {
		var (
			files = c.StringSlice("files")
			tags  = c.StringSlice("tags")
			hosts = c.StringSlice("hosts")
		)

		if len(files) == 0 && len(tags) == 0 && len(hosts) == 0 {
			fmt.Fprintln(os.Stderr, "You must provide at least one of the --files, --tags or --hosts flags")
			return
		}

		// Purge selectively
		purgeReq := cloudflare.PurgeCacheRequest{
			Files: c.StringSlice("files"),
			Tags:  c.StringSlice("tags"),
			Hosts: c.StringSlice("hosts"),
		}

		resp, err = api.PurgeCache(zoneID, purgeReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error purging the cache from zone %q: %s\n", zoneName, err)
			return
		}
	}

	output := make([][]string, 0, 1)
	output = append(output, formatCacheResponse(resp))

	writeTable(output, "ID")
}

func zoneRecords(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	var zone string
	if len(c.Args()) > 0 {
		zone = c.Args()[0]
	} else if c.String("zone") != "" {
		zone = c.String("zone")
	} else {
		cli.ShowSubcommandHelp(c)
		return
	}

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Create a an empty record for searching for records
	rr := cloudflare.DNSRecord{}
	var records []cloudflare.DNSRecord
	if c.String("id") != "" {
		rec, err := api.DNSRecord(zoneID, c.String("id"))
		if err != nil {
			fmt.Println(err)
			return
		}
		records = append(records, rec)
	} else {
		if c.String("name") != "" {
			rr.Name = c.String("name")
		}
		if c.String("content") != "" {
			rr.Name = c.String("content")
		}
		var err error
		records, err = api.DNSRecords(zoneID, rr)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	output := make([][]string, 0, len(records))
	for _, r := range records {
		switch r.Type {
		case "MX":
			r.Content = fmt.Sprintf("%d %s", r.Priority, r.Content)
		case "SRV":
			dp := r.Data.(map[string]interface{})
			r.Content = fmt.Sprintf("%.f %s", dp["priority"], r.Content)
			// Cloudflare's API, annoyingly, automatically prepends the weight
			// and port into content, separated by tabs.
			// XXX: File this as a bug. LOC doesn't do this.
			r.Content = strings.Replace(r.Content, "\t", " ", -1)
		}
		output = append(output, []string{
			r.ID,
			r.Type,
			r.Name,
			r.Content,
			fmt.Sprintf("%t", r.Proxied),
			fmt.Sprintf("%d", r.TTL),
		})
	}
	writeTable(output, "ID", "Type", "Name", "Content", "Proxied", "TTL")
}

func formatCacheResponse(resp cloudflare.PurgeCacheResponse) []string {
	return []string{
		resp.Result.ID,
	}
}
