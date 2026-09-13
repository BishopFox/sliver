package top

import "time"

const crackTopHistoryLimit = 120

// crackTopHistorySample captures the real telemetry available at one refresh
// observation. Known fields distinguish an observed zero from telemetry that
// was unavailable, including when the refresh itself failed.
type crackTopHistorySample struct {
	At time.Time

	ClusterRate      uint64
	ClusterRateKnown bool
	ClusterComplete  bool
	Progress         float64
	ProgressKnown    bool
	Utilization      float64
	UtilizationKnown bool
	Temperature      float64
	TemperatureKnown bool
	CountsKnown      bool

	ActiveJobs    int
	OnlineWorkers int
	TotalWorkers  int
	Recovered     uint64
}

func crackTopHistorySampleFromDashboard(dashboard crackTopDashboard, at time.Time) crackTopHistorySample {
	sample := crackTopHistorySample{
		At:               at,
		ActiveJobs:       dashboard.ActiveJobs,
		OnlineWorkers:    dashboard.OnlineWorkers,
		TotalWorkers:     dashboard.OnlineWorkers + dashboard.Unavailable,
		Recovered:        dashboard.Recovered,
		ClusterRateKnown: dashboard.ClusterRateKnown,
		ClusterComplete:  dashboard.ClusterComplete,
		ProgressKnown:    dashboard.ProgressKnown,
		CountsKnown:      true,
	}
	if sample.ClusterRateKnown {
		sample.ClusterRate = dashboard.ClusterRate
	}
	if sample.ProgressKnown {
		sample.Progress = dashboard.Progress
	}

	var utilizationTotal float64
	utilizationSamples := 0
	for _, worker := range dashboard.Workers {
		if !worker.TelemetryLive {
			continue
		}
		if worker.HasUtil {
			utilizationTotal += worker.Utilization
			utilizationSamples++
		}
		if worker.HasTemp && (!sample.TemperatureKnown || worker.Temperature > sample.Temperature) {
			sample.Temperature = worker.Temperature
			sample.TemperatureKnown = true
		}
	}
	if utilizationSamples > 0 {
		sample.Utilization = utilizationTotal / float64(utilizationSamples)
		sample.UtilizationKnown = true
	}
	return sample
}

func appendCrackTopHistory(history []crackTopHistorySample, sample crackTopHistorySample) []crackTopHistorySample {
	if len(history) >= crackTopHistoryLimit {
		start := len(history) - (crackTopHistoryLimit - 1)
		retained := copy(history, history[start:])
		history = history[:retained]
	}
	return append(history, sample)
}
