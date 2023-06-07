package dnsclient

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	insecureRand "math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/dnspb"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"
)

const (
	// Do not change these without updating the tests
	parent1 = ".1.example.com."
	parent2 = ".something-longer.example.com."
	parent3 = ".something-even-longer.example.computer."
)

var (
	// Max NS subdomain + Max Domain + Max TLD = 153 chars
	parentMax = fmt.Sprintf(".%s.%s.%s.", strings.Repeat("a", 63), strings.Repeat("b", 63), strings.Repeat("c", 24))

	opts = &DNSOptions{
		QueryTimeout:       time.Duration(time.Second * 3),
		RetryWait:          time.Duration(time.Second * 3),
		RetryCount:         1,
		WorkersPerResolver: 1,
	}
)

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
}

func randomDataRandomSize(maxSize int) []byte {
	buf := make([]byte, insecureRand.Intn(maxSize))
	rand.Read(buf)
	return buf
}

func randomData(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
	return buf
}

func TestSplitBufferBase58(t *testing.T) {

	t.Logf("Testing with client1")
	client1 := NewDNSClient(parent1, opts)
	testData := randomData(2048)
	clientSplitBuffer(t, client1, encoders.Base58{}, testData)

	t.Logf("Testing with client2")
	client2 := NewDNSClient(parent2, opts)
	testData2 := randomData(2048)
	clientSplitBuffer(t, client2, encoders.Base58{}, testData2)

	t.Logf("Testing with client3")
	client3 := NewDNSClient(parent3, opts)
	testData3 := randomData(2048)
	clientSplitBuffer(t, client3, encoders.Base58{}, testData3)

	t.Logf("Testing with client max")
	clientMax := NewDNSClient(parentMax, opts)
	testDataMax := randomData(2048)
	clientSplitBuffer(t, clientMax, encoders.Base58{}, testDataMax)

	t.Logf("Testing all clients with randomly sized data")
	for _, client := range []*SliverDNSClient{client1, client2, client3} {
		for count := 0; count < 10; count++ {
			testData := randomDataRandomSize(2 * 1024 * 1024)
			clientSplitBuffer(t, client, encoders.Base58{}, testData)
		}
	}
}

func TestSplitBufferBase32(t *testing.T) {

	t.Logf("Testing with client1")
	client1 := NewDNSClient(parent1, opts)
	testData := randomData(2048)
	clientSplitBuffer(t, client1, encoders.Base32{}, testData)

	t.Logf("Testing with client2")
	client2 := NewDNSClient(parent2, opts)
	testData2 := randomData(2048)
	clientSplitBuffer(t, client2, encoders.Base32{}, testData2)

	t.Logf("Testing with client3")
	client3 := NewDNSClient(parent3, opts)
	testData3 := randomData(2048)
	clientSplitBuffer(t, client3, encoders.Base32{}, testData3)

	t.Logf("Testing with client max")
	clientMax := NewDNSClient(parentMax, opts)
	testDataMax := randomData(2048)
	clientSplitBuffer(t, clientMax, encoders.Base32{}, testDataMax)

	t.Logf("Testing all clients with randomly sized data")
	for _, client := range []*SliverDNSClient{client1, client2, client3} {
		for count := 0; count < 10; count++ {
			testData := randomDataRandomSize(2 * 1024 * 1024)
			clientSplitBuffer(t, client, encoders.Base32{}, testData)
		}
	}
}

func clientSplitBuffer(t *testing.T, client *SliverDNSClient, encoder encoders.Encoder, testData []byte) {
	msg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_DATA_FROM_IMPLANT,
		Size: uint32(len(testData)),
	}
	domains, err := client.SplitBuffer(msg, encoder, testData)
	if err != nil {
		t.Fatalf("Unexpected error splitting buffer: %s", err)
	}
	t.Logf("Domains: %v", domains)
	allData := []byte{}
	for _, domain := range domains {
		subdata := strings.TrimSuffix(domain, client.parent)
		subdata = strings.ReplaceAll(subdata, ".", "")
		data, err := encoder.Decode([]byte(subdata))
		if err != nil {
			t.Fatalf("Unexpected error decoding subdata: %s", err)
		}
		msg := &dnspb.DNSMessage{}
		err = proto.Unmarshal(data, msg)
		if err != nil {
			t.Fatalf("Unexpected error un-marshaling subdata: %s", err)
		}
		allData = append(allData, msg.Data...)
	}
	if !bytes.Equal(allData, testData) {
		t.Fatalf("Unexpected data returned from splitting buffer\nSample: %v\nData: %v\n", testData, allData)
	}
}

// func addTestResolver(client *SliverDNSClient, enableBase58 bool) {
// 	client.resolvers = []DNSResolver{
// 		&GenericResolver{
// 			address:   "127.0.0.1:53",
// 			retries:   1,
// 			retryWait: retryWait,
// 			resolver: &dns.Client{
// 				ReadTimeout:  timeout,
// 				WriteTimeout: timeout,
// 			},
// 			base64: encoders.Base64{},
// 		},
// 	}
// 	client.metadata["127.0.0.1:53"] = &ResolverMetadata{
// 		Address:      "127.0.0.1:53",
// 		EnableBase58: enableBase58,
// 		Errors:       0,
// 	}
// }

func TestSubdataSpace(t *testing.T) {

	// 1.example.com. (14 chars parent, 240 chars subdata)
	// Grand Total: 254 chars
	//       parent |  subdata with '.'    | subdata without '.'
	// 254 -   15   -  [64 - 64 - 64 - 47] = 63 + 63 + 63 + 46 (235)
	// expected value is thus 235 (max chars without '.'), rounded down if applicable
	client1 := NewDNSClient(parent1, opts)
	if client1.subdataSpace != 235 {
		t.Fatalf("Unexpected subdata space for parent %s: %d", parent1, client1.subdataSpace)
	}

	// .something-longer.example.com. (30 chars parent, 224 chars subdata)
	// Grand Total: 254 chars
	//       parent |  subdata with '.'    | subdata without '.'
	// 254 -   30   -  [64 - 64 - 64 - 32] = 63 + 63 + 63 + 31 (220)
	// expected value is thus 235 (max chars without '.'), rounded down if applicable
	client2 := NewDNSClient(parent2, opts)
	if client2.subdataSpace != 220 {
		t.Fatalf("Unexpected subdata space for parent %s: %d", parent2, client2.subdataSpace)
	}

	// .something-even-longer.example.computer. (40 chars parent, 214 chars subdata)
	// Grand Total: 254 chars
	//       parent |  subdata with '.'    | subdata without '.'
	// 254 -   40   -  [64 - 64 - 64 - 22] = 63 + 63 + 63 + 21 (210)
	// expected value is thus 235 (max chars without '.'), rounded down if applicable
	client3 := NewDNSClient(parent3, opts)
	if client3.subdataSpace != 210 {
		t.Fatalf("Unexpected subdata space for parent %s: %d", parent3, client3.subdataSpace)
	}

	// "maxParent" (154 chars parent, 100 chars subdata)
	// Grand Total: 254 chars
	//       parent  |  subdata with '.'    | subdata without '.'
	// 254 -   154   -  [64 - 36]           = 63 + 35 (98)
	// expected value is thus 98 (max chars without '.'), rounded down if applicable
	clientMax := NewDNSClient(parentMax, opts)
	if clientMax.subdataSpace != 98 {
		t.Fatalf("Unexpected subdata space for parent %s: %d", parentMax, clientMax.subdataSpace)
	}
}

func TestJoinSubdata(t *testing.T) {
	subdata := strings.Repeat("1234567890", 9) // 90 chars

	client1 := NewDNSClient(parent1, opts)
	domain, err := client1.joinSubdataToParent(subdata)
	if err != nil {
		t.Fatalf("Error joining subdata to parent: %s", err)
	}
	if domain != "123456789012345678901234567890123456789012345678901234567890123.456789012345678901234567890.1.example.com." {
		t.Fatalf("Unexpected domain value: %s", domain)
	}

	client2 := NewDNSClient(parent2, opts)
	domain, err = client2.joinSubdataToParent(subdata)
	if err != nil {
		t.Fatalf("Error joining subdata to parent: %s", err)
	}
	if domain != "123456789012345678901234567890123456789012345678901234567890123.456789012345678901234567890.something-longer.example.com." {
		t.Fatalf("Unexpected domain value: %s", domain)
	}

	client3 := NewDNSClient(parent3, opts)
	domain, err = client3.joinSubdataToParent(subdata)
	if err != nil {
		t.Fatalf("Error joining subdata to parent: %s", err)
	}
	if domain != "123456789012345678901234567890123456789012345678901234567890123.456789012345678901234567890.something-even-longer.example.computer." {
		t.Fatalf("Unexpected domain value: %s", domain)
	}

	clientMax := NewDNSClient(parentMax, opts)
	domain, err = clientMax.joinSubdataToParent(subdata)
	if err != nil {
		t.Fatalf("Error joining subdata to parent: %s", err)
	}
	if domain != fmt.Sprintf("123456789012345678901234567890123456789012345678901234567890123.456789012345678901234567890%s", parentMax) {
		t.Fatalf("Unexpected domain value: %s", domain)
	}

	subdataTooLong := strings.Repeat("1234567890", 10)
	_, err = clientMax.joinSubdataToParent(subdataTooLong)
	if err != errMsgTooLong {
		t.Fatalf("Expected error: %s", errMsgTooLong)
	}
}
