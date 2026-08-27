package c2

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type recordingWGPeerConfigurator struct {
	events *[]string
	config string
	err    error
}

func (configurator *recordingWGPeerConfigurator) IpcSetOperation(reader io.Reader) error {
	*configurator.events = append(*configurator.events, "apply-peer")
	config, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	configurator.config = string(config)
	return configurator.err
}

type recordingWGKeyExchangeWriter struct {
	events     *[]string
	response   bytes.Buffer
	err        error
	shortWrite bool
}

func (writer *recordingWGKeyExchangeWriter) Write(data []byte) (int, error) {
	*writer.events = append(*writer.events, "write-response")
	if writer.err != nil {
		return 0, writer.err
	}
	if writer.shortWrite {
		return len(data) - 1, nil
	}
	return writer.response.Write(data)
}

func TestWriteWGKeyExchangeResponseAppliesPeerBeforeSendingCredentials(t *testing.T) {
	events := []string{}
	configurator := &recordingWGPeerConfigurator{events: &events}
	writer := &recordingWGKeyExchangeWriter{events: &events}

	err := writeWGKeyExchangeResponse(
		writer,
		configurator,
		func() (string, string, string, error) {
			events = append(events, "generate-peer")
			return "100.64.0.42", "implant-private-key", "implant-public-key", nil
		},
		func() (string, string, error) {
			events = append(events, "get-server-keys")
			return "server-private-key", "server-public-key", nil
		},
	)
	if err != nil {
		t.Fatalf("write key exchange response: %v", err)
	}

	wantEvents := []string{"get-server-keys", "generate-peer", "apply-peer", "write-response"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("unexpected operation order: got=%v want=%v", events, wantEvents)
	}
	if want := "public_key=implant-public-key\nallowed_ip=100.64.0.42/32\n"; configurator.config != want {
		t.Fatalf("unexpected peer config: got=%q want=%q", configurator.config, want)
	}
	if want := "implant-private-key|server-public-key|100.64.0.42"; writer.response.String() != want {
		t.Fatalf("unexpected key exchange response: got=%q want=%q", writer.response.String(), want)
	}
}

func TestWriteWGKeyExchangeResponseDoesNotSendCredentialsBeforePeerApply(t *testing.T) {
	errApply := errors.New("apply failed")
	events := []string{}
	configurator := &recordingWGPeerConfigurator{events: &events, err: errApply}
	writer := &recordingWGKeyExchangeWriter{events: &events}

	err := writeWGKeyExchangeResponse(
		writer,
		configurator,
		func() (string, string, string, error) {
			events = append(events, "generate-peer")
			return "100.64.0.42", "implant-private-key", "implant-public-key", nil
		},
		func() (string, string, error) {
			events = append(events, "get-server-keys")
			return "server-private-key", "server-public-key", nil
		},
	)
	if !errors.Is(err, errApply) {
		t.Fatalf("expected peer apply error, got %v", err)
	}
	if strings.Contains(strings.Join(events, ","), "write-response") {
		t.Fatalf("credentials were written after peer apply failed: events=%v", events)
	}
	if writer.response.Len() != 0 {
		t.Fatalf("unexpected key exchange response after peer apply failed: %q", writer.response.String())
	}
}

func TestWriteWGKeyExchangeResponseErrors(t *testing.T) {
	errServerKeys := errors.New("server keys failed")
	errGenerate := errors.New("generate failed")
	errWrite := errors.New("write failed")

	tests := []struct {
		name             string
		serverKeysErr    error
		generateErr      error
		writerErr        error
		shortWrite       bool
		wantErr          error
		wantConfigCalled bool
		wantWriteCalled  bool
	}{
		{
			name:          "server keys",
			serverKeysErr: errServerKeys,
			wantErr:       errServerKeys,
		},
		{
			name:        "peer generation",
			generateErr: errGenerate,
			wantErr:     errGenerate,
		},
		{
			name:             "response write",
			writerErr:        errWrite,
			wantErr:          errWrite,
			wantConfigCalled: true,
			wantWriteCalled:  true,
		},
		{
			name:             "short response write",
			shortWrite:       true,
			wantErr:          io.ErrShortWrite,
			wantConfigCalled: true,
			wantWriteCalled:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			configurator := &recordingWGPeerConfigurator{events: &events}
			writer := &recordingWGKeyExchangeWriter{
				events:     &events,
				err:        test.writerErr,
				shortWrite: test.shortWrite,
			}

			err := writeWGKeyExchangeResponse(
				writer,
				configurator,
				func() (string, string, string, error) {
					events = append(events, "generate-peer")
					if test.generateErr != nil {
						return "", "", "", test.generateErr
					}
					return "100.64.0.42", "implant-private-key", "implant-public-key", nil
				},
				func() (string, string, error) {
					events = append(events, "get-server-keys")
					if test.serverKeysErr != nil {
						return "", "", test.serverKeysErr
					}
					return "server-private-key", "server-public-key", nil
				},
			)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}

			configCalled := strings.Contains(strings.Join(events, ","), "apply-peer")
			if configCalled != test.wantConfigCalled {
				t.Fatalf("unexpected peer config call state: got=%v want=%v events=%v", configCalled, test.wantConfigCalled, events)
			}
			writeCalled := strings.Contains(strings.Join(events, ","), "write-response")
			if writeCalled != test.wantWriteCalled {
				t.Fatalf("unexpected response write state: got=%v want=%v events=%v", writeCalled, test.wantWriteCalled, events)
			}
		})
	}
}
