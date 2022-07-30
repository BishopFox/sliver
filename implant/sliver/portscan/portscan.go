package portscan

import (
	"fmt"
	"sync"
)

func Scan(hostSpec string, portSpec string, numThreads int32) (string, error) {
	var probes []*Probe
	var output string

	hosts, err := parseHostSpec(hostSpec)
	if err != nil {
		return "", fmt.Errorf("No hosts could be loaded")
	}

	ports, err := parsePortSpec(portSpec)
	if err != nil {
		return "", fmt.Errorf("No ports could be loaded")
	}

	for _, host := range hosts {
		for _, port := range ports {
			probes = append(probes, NewProbe(host, port))
		}
	}

	input := make(chan *Probe, numThreads)
	results := make(chan *Probe)

	numProducers := int(numThreads)
	numConsumers := 1
	var wgProducers sync.WaitGroup
	var wgConsumers sync.WaitGroup
	wgProducers.Add(numProducers)
	wgConsumers.Add(numConsumers)

	for i := 0; i < numProducers; i++ {
		go func() {
			defer wgProducers.Done()

			for {
				probe := <-input
				if probe == nil {
					break
				}

				probe.Probe()
				results <- probe
			}
		}()
	}

	for i := 0; i < numConsumers; i++ {
		go func() {
			defer wgConsumers.Done()
			for result := range results {
				if result.open == true {
					output += result.Report() + "\n"
				}
			}
		}()
	}

	for _, probe := range probes {
		input <- probe
	}

	close(input)
	wgProducers.Wait()
	close(results)
	wgConsumers.Wait()

	if output == "" {
		output = "No open ports were found"
	}

	return output, nil
}
