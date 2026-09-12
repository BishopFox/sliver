package top

import (
	"testing"
	"time"
)

func TestCrackTopHistorySampleFromDashboardDerivesLiveTelemetry(t *testing.T) {
	at := time.Unix(1_700_000_000, 123)
	dashboard := crackTopDashboard{
		ActiveJobs:       2,
		OnlineWorkers:    3,
		Unavailable:      2,
		Recovered:        9,
		ClusterRate:      12_345,
		ClusterRateKnown: true,
		ClusterComplete:  true,
		Progress:         0.625,
		ProgressKnown:    true,
		Workers: []crackTopWorkerRow{
			{TelemetryLive: true, Utilization: 20, HasUtil: true, Temperature: 60, HasTemp: true},
			{TelemetryLive: true, Utilization: 80, HasUtil: true, Temperature: 75, HasTemp: true},
			{TelemetryLive: true},
			{TelemetryLive: false, Utilization: 100, HasUtil: true, Temperature: 99, HasTemp: true},
		},
	}

	sample := crackTopHistorySampleFromDashboard(dashboard, at)
	if !sample.At.Equal(at) {
		t.Fatalf("sample time = %s, want %s", sample.At, at)
	}
	if !sample.ClusterRateKnown || sample.ClusterRate != 12_345 {
		t.Fatalf("cluster rate = %d known %v, want 12345 known", sample.ClusterRate, sample.ClusterRateKnown)
	}
	if !sample.ClusterComplete {
		t.Fatal("complete cluster snapshot was recorded as incomplete")
	}
	if !sample.ProgressKnown || sample.Progress != 0.625 {
		t.Fatalf("progress = %v known %v, want 0.625 known", sample.Progress, sample.ProgressKnown)
	}
	if !sample.UtilizationKnown || sample.Utilization != 50 {
		t.Fatalf("utilization = %v known %v, want live-worker mean 50", sample.Utilization, sample.UtilizationKnown)
	}
	if !sample.TemperatureKnown || sample.Temperature != 75 {
		t.Fatalf("temperature = %v known %v, want live-worker maximum 75", sample.Temperature, sample.TemperatureKnown)
	}
	if sample.ActiveJobs != 2 || sample.OnlineWorkers != 3 || sample.TotalWorkers != 5 || sample.Recovered != 9 {
		t.Fatalf("sample counts = active:%d online:%d total:%d recovered:%d", sample.ActiveJobs, sample.OnlineWorkers, sample.TotalWorkers, sample.Recovered)
	}
	if !sample.CountsKnown {
		t.Fatal("successful dashboard sample marked counts unknown")
	}
}

func TestCrackTopHistorySampleFromDashboardPreservesUnknowns(t *testing.T) {
	dashboard := crackTopDashboard{
		ClusterRate: 999,
		Progress:    0.9,
		Workers: []crackTopWorkerRow{
			{TelemetryLive: false, Utilization: 88, HasUtil: true, Temperature: 70, HasTemp: true},
			{TelemetryLive: true},
		},
	}

	sample := crackTopHistorySampleFromDashboard(dashboard, time.Time{})
	if sample.ClusterRateKnown || sample.ClusterRate != 0 {
		t.Fatalf("unknown cluster rate = %d known %v, want zero unknown", sample.ClusterRate, sample.ClusterRateKnown)
	}
	if sample.ClusterComplete {
		t.Fatal("incomplete cluster snapshot was recorded as complete")
	}
	if sample.ProgressKnown || sample.Progress != 0 {
		t.Fatalf("unknown progress = %v known %v, want zero unknown", sample.Progress, sample.ProgressKnown)
	}
	if sample.UtilizationKnown || sample.Utilization != 0 {
		t.Fatalf("unknown utilization = %v known %v, want zero unknown", sample.Utilization, sample.UtilizationKnown)
	}
	if sample.TemperatureKnown || sample.Temperature != 0 {
		t.Fatalf("unknown temperature = %v known %v, want zero unknown", sample.Temperature, sample.TemperatureKnown)
	}
}

func TestAppendCrackTopHistoryCapsAndRetainsNewestSamples(t *testing.T) {
	history := make([]crackTopHistorySample, 0, crackTopHistoryLimit)
	for index := 0; index < crackTopHistoryLimit+5; index++ {
		history = appendCrackTopHistory(history, crackTopHistorySample{ActiveJobs: index})
	}

	if len(history) != crackTopHistoryLimit {
		t.Fatalf("history length = %d, want %d", len(history), crackTopHistoryLimit)
	}
	if history[0].ActiveJobs != 5 {
		t.Fatalf("oldest retained sample = %d, want 5", history[0].ActiveJobs)
	}
	if history[len(history)-1].ActiveJobs != crackTopHistoryLimit+4 {
		t.Fatalf("newest retained sample = %d, want %d", history[len(history)-1].ActiveJobs, crackTopHistoryLimit+4)
	}
}
