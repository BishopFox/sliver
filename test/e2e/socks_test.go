package e2e

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestRequireSocksE2EAuthRejectionOnConn(t *testing.T) {
	const username = "range-user"
	const password = "wrong-password"
	for _, testCase := range []struct {
		name    string
		status  byte
		wantErr bool
	}{
		{name: "explicit rejection", status: 0x01},
		{name: "incorrect password accepted", status: 0x00, wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer func() { _ = client.Close() }()
			defer func() { _ = server.Close() }()
			deadline := time.Now().Add(time.Second)
			_ = client.SetDeadline(deadline)
			_ = server.SetDeadline(deadline)
			serverDone := make(chan error, 1)
			go func() {
				greeting := make([]byte, 3)
				if _, err := io.ReadFull(server, greeting); err != nil {
					serverDone <- err
					return
				}
				if !bytes.Equal(greeting, []byte{0x05, 0x01, 0x02}) {
					serverDone <- fmt.Errorf("greeting = %x", greeting)
					return
				}
				if _, err := server.Write([]byte{0x05, 0x02}); err != nil {
					serverDone <- err
					return
				}
				authRequest := make([]byte, 3+len(username)+len(password))
				if _, err := io.ReadFull(server, authRequest); err != nil {
					serverDone <- err
					return
				}
				wantRequest := append([]byte{0x01, byte(len(username))}, username...)
				wantRequest = append(wantRequest, byte(len(password)))
				wantRequest = append(wantRequest, password...)
				if !bytes.Equal(authRequest, wantRequest) {
					serverDone <- fmt.Errorf("auth request = %x, want %x", authRequest, wantRequest)
					return
				}
				_, err := server.Write([]byte{0x01, testCase.status})
				serverDone <- err
			}()

			err := requireSocksE2EAuthRejectionOnConn(client, username, password)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("auth rejection error = %v, wantErr=%t", err, testCase.wantErr)
			}
			if err := <-serverDone; err != nil {
				t.Fatalf("fake SOCKS5 server: %v", err)
			}
		})
	}
}

//nolint:gocyclo // A single table-driven assertion compares every field of the deterministic corpus.
func TestSocksE2EBinaryCorpusIsDeterministic(t *testing.T) {
	first := socksE2EBinaryCases()
	second := socksE2EBinaryCases()
	wantCount := len(socksE2EBoundarySizes) + 8
	if len(first) != wantCount || len(second) != wantCount {
		t.Fatalf("binary corpus lengths = %d and %d, want %d", len(first), len(second), wantCount)
	}

	for index := range first {
		left := first[index]
		right := second[index]
		if left.Index != index || right.Index != index {
			t.Fatalf("case %d indexes = %d and %d", index, left.Index, right.Index)
		}
		if left.Length != len(left.Payload) || right.Length != len(right.Payload) {
			t.Fatalf("case %d declared lengths = %d and %d, payload lengths = %d and %d", index, left.Length, right.Length, len(left.Payload), len(right.Payload))
		}
		if left.Length <= 0 {
			t.Fatalf("case %d has invalid length %d", index, left.Length)
		}
		if left.Length != right.Length || !bytes.Equal(left.Payload, right.Payload) {
			t.Fatalf("seed %#x case %d is not reproducible", socksE2EBinarySeed, index)
		}
	}

	for index, wantSize := range socksE2EBoundarySizes {
		got := first[index]
		if got.Length != wantSize {
			t.Fatalf("boundary case %d length = %d, want %d", index, got.Length, wantSize)
		}
		if got.Payload[0] != 0 {
			t.Fatalf("boundary case %d does not begin with a NUL byte", index)
		}
		if len(got.Payload) > 1 && got.Payload[len(got.Payload)-1] != 0xff {
			t.Fatalf("boundary case %d does not end with 0xff", index)
		}
	}
}

func TestSocksE2EMutationCorpusIsDeterministicAndReplayable(t *testing.T) {
	first := socksE2EMutations(socksE2EMutationSeed, socksE2EMalformedCases)
	second := socksE2EMutations(socksE2EMutationSeed, socksE2EMalformedCases)
	if len(first) != socksE2EMalformedCases || len(second) != socksE2EMalformedCases {
		t.Fatalf("mutation corpus lengths = %d and %d, want %d", len(first), len(second), socksE2EMalformedCases)
	}

	mutated := 0
	for index := range first {
		left := first[index]
		right := second[index]
		if left.Index != index || right.Index != index {
			t.Fatalf("case %d indexes = %d and %d", index, left.Index, right.Index)
		}
		if left.Kind == "" {
			t.Fatalf("seed %#x case %d has no mutation kind", socksE2EMutationSeed, index)
		}
		if left.Kind != right.Kind || !bytes.Equal(left.Data, right.Data) {
			t.Fatalf("seed %#x case %d is not reproducible", socksE2EMutationSeed, index)
		}
		if strings.HasPrefix(left.Kind, "mutated-") {
			mutated++
		}
		if err := validateSocksE2EMutationEgress(left.Data); err != nil {
			t.Fatalf("seed %#x case %d can escape loopback: %v", socksE2EMutationSeed, index, err)
		}
	}
	if mutated == 0 {
		t.Fatal("mutation corpus contains no seeded mutations")
	}
}

func TestSocksE2EMutationCorpusCannotCreateExternalConnect(t *testing.T) {
	for _, seed := range []int64{0, 1, socksE2EMutationSeed, -1, 0x7fff_ffff_ffff_ffff} {
		for _, mutation := range socksE2EMutations(seed, 4096) {
			if err := validateSocksE2EMutationEgress(mutation.Data); err != nil {
				t.Fatalf("seed=%#x case=%d kind=%s data=%x: %v", seed, mutation.Index, mutation.Kind, mutation.Data, err)
			}
		}
	}

	unsafe := []byte{0x05, 0x01, 0x00, 0x05, 0x01, 0x00, 0x01, 192, 0, 2, 1, 0, 80}
	if err := validateSocksE2EMutationEgress(unsafe); err == nil {
		t.Fatal("egress guard accepted a complete non-loopback CONNECT")
	}
	for name, networkCommand := range map[string][]byte{
		"loopback CONNECT": {0x05, 0x01, 0x00, 0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1, 0x13, 0x37},
		"loopback BIND":    {0x05, 0x01, 0x00, 0x05, 0x02, 0x00, 0x01, 127, 0, 0, 1, 0x13, 0x37},
		"UDP ASSOCIATE":    {0x05, 0x01, 0x00, 0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0, 0},
	} {
		if err := validateSocksE2EMutationEgress(networkCommand); err == nil {
			t.Errorf("egress guard accepted complete %s", name)
		}
	}
}

func TestValidateSocksFuzzOptionsBoundsCampaignAndReplay(t *testing.T) {
	for _, testCase := range []struct {
		cases  int
		replay int
		valid  bool
	}{
		{cases: 1, replay: -1, valid: true},
		{cases: socksE2EMaxFuzzCases, replay: socksE2EMaxFuzzCases - 1, valid: true},
		{cases: 0, replay: -1},
		{cases: socksE2EMaxFuzzCases + 1, replay: -1},
		{cases: 1, replay: -2},
		{cases: 1, replay: socksE2EMaxFuzzCases},
	} {
		err := validateSocksFuzzOptions(testCase.cases, testCase.replay)
		if (err == nil) != testCase.valid {
			t.Errorf("validate cases=%d replay=%d error=%v, valid=%v", testCase.cases, testCase.replay, err, testCase.valid)
		}
	}
}

func TestSocksE2EMutationCorpusHonorsSeedAndCount(t *testing.T) {
	short := socksE2EMutations(17, 3)
	if len(short) != 3 {
		t.Fatalf("short corpus length = %d, want 3", len(short))
	}
	if got := socksE2EMutations(17, 0); got != nil {
		t.Fatalf("zero-length corpus = %#v, want nil", got)
	}

	first := socksE2EMutations(17, socksE2EMalformedCases)
	second := socksE2EMutations(18, socksE2EMalformedCases)
	if len(first) != len(second) {
		t.Fatalf("seeded corpus lengths = %d and %d", len(first), len(second))
	}
	// The fixed protocol seeds intentionally match. At least one generated
	// mutation must differ so a new seed explores a distinct live corpus.
	different := false
	for index := 10; index < len(first); index++ {
		if first[index].Kind != second[index].Kind || !bytes.Equal(first[index].Data, second[index].Data) {
			different = true
			break
		}
	}
	if !different {
		t.Fatal("different fuzz seeds generated identical mutation corpora")
	}
}

func TestSocksE2EReplayCanAddressCaseBeyondCampaignCount(t *testing.T) {
	if got := socksE2EGeneratedCaseCount(32, 127); got != 128 {
		t.Fatalf("generated replay corpus count = %d, want 128", got)
	}
	if got := socksE2EGeneratedCaseCount(128, 7); got != 128 {
		t.Fatalf("generated in-range replay corpus count = %d, want 128", got)
	}
	short := socksE2EMutations(0x7eed5eed, 32)
	long := socksE2EMutations(0x7eed5eed, 128)
	for index := range short {
		if short[index].Kind != long[index].Kind || !bytes.Equal(short[index].Data, long[index].Data) {
			t.Fatalf("extending the replay corpus changed case %d", index)
		}
	}
}

func TestSocksE2EMalformedScenarioTimeoutScalesAndReplayIsBounded(t *testing.T) {
	defaultTimeout := socksE2EMalformedScenarioTimeout(32, -1)
	stressTimeout := socksE2EMalformedScenarioTimeout(128, -1)
	replayTimeout := socksE2EMalformedScenarioTimeout(32, 127)
	if defaultTimeout <= 2*time.Minute {
		t.Fatalf("default malformed timeout = %s, want more than the ordinary two-minute command timeout", defaultTimeout)
	}
	if stressTimeout <= defaultTimeout {
		t.Fatalf("128-case timeout = %s, want greater than 32-case timeout %s", stressTimeout, defaultTimeout)
	}
	if replayTimeout >= defaultTimeout || replayTimeout < 2*time.Second+2*socksE2ECaseTimeout {
		t.Fatalf("replay timeout = %s, want one-case bounded timeout below %s", replayTimeout, defaultTimeout)
	}
}

func TestRequireSocksE2EMalformedResponse(t *testing.T) {
	tests := []struct {
		name     string
		mutation socksE2EMutation
		response []byte
		wantErr  bool
	}{
		{name: "truncated timeout without response", mutation: socksE2EMutation{Kind: "truncated-request"}},
		{name: "unsupported method", mutation: socksE2EMutation{Kind: "unsupported-method"}, response: []byte{0x05, 0xff}},
		{name: "unsupported command", mutation: socksE2EMutation{Kind: "unsupported-command"}, response: []byte{0x05, 0x00, 0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}},
		{name: "rule rejects UDP", mutation: socksE2EMutation{Kind: "unsupported-udp-associate"}, response: []byte{0x05, 0x00, 0x05, 0x02, 0x00, 0x01, 0, 0, 0, 0, 0, 0}},
		{name: "successful connect rejected", mutation: socksE2EMutation{Kind: "mutated-random"}, response: []byte{0x05, 0x00, 0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0x13, 0x37}, wantErr: true},
		{name: "wrong unsupported reply rejected", mutation: socksE2EMutation{Kind: "unsupported-command"}, response: []byte{0x05, 0x00, 0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := requireSocksE2EMalformedResponse(test.mutation, test.response)
			if test.wantErr && err == nil {
				t.Fatal("malformed response oracle accepted invalid response")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("malformed response oracle: %v", err)
			}
		})
	}
}

func TestSocksE2EHostnameHelpersPreservePortAndPath(t *testing.T) {
	address, err := socksE2EHostnameAddress("127.0.0.1:31337")
	if err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	if host != "localhost" || port != "31337" {
		t.Fatalf("hostname address = %q, want localhost:31337", address)
	}

	gotURL, err := socksE2EHostnameURL("http://127.0.0.1:8080/ignored?keep=yes", "/socks-fixture")
	if err != nil {
		t.Fatal(err)
	}
	if gotURL != "http://localhost:8080/socks-fixture?keep=yes" {
		t.Fatalf("hostname URL = %q", gotURL)
	}

	if _, err := socksE2EHostnameAddress("missing-port"); err == nil {
		t.Fatal("hostname address helper accepted an address without a port")
	}
	if _, err := socksE2EHostnameURL("http://127.0.0.1/path", "/socks-fixture"); err == nil {
		t.Fatal("hostname URL helper accepted a URL without a port")
	}
}
