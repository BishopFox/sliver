package portscan

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func parseHostSpec(hostSpec string) ([]string, error) {
	var hosts []string

	if fileExists(hostSpec) {
		file, err := os.Open(hostSpec)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			hosts = append(hosts, strings.Trim(scanner.Text(), " "))
		}
	} else if strings.Contains(hostSpec, "/") {
		for _, host := range explodeCidr(hostSpec) {
			hosts = append(hosts, host.String())
		}
	} else {
		hosts = append(hosts, hostSpec)
	}

	return hosts, nil
}

func atoi(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}

func isNumeric(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func parsePortSpec(portSpec string) ([]int, error) {
	var ports []int

	for _, commas := range strings.Split(portSpec, ",") {
		if strings.Contains(commas, "-") {
			dashes := strings.Split(commas, "-")
			start := atoi(dashes[0])
			end := atoi(dashes[1])
			for i := start; i < end+1; i++ {
				ports = append(ports, i)
			}
		} else {
			ports = append(ports, atoi(commas))
		}
	}

	return ports, nil
}
