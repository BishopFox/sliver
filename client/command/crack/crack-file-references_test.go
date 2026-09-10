package crack

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestResolveManagedCrackFileReferences(t *testing.T) {
	wordlistDigest := strings.Repeat("a", 64)
	rulesDigest := strings.Repeat("b", 64)
	hcstat2Digest := strings.Repeat("c", 64)
	files := map[clientpb.CrackFileType][]*clientpb.CrackFile{
		clientpb.CrackFileType_WORDLIST: {{ID: "wordlist-id", Name: "rockyou.txt", Sha2_256: wordlistDigest, Type: clientpb.CrackFileType_WORDLIST}},
		clientpb.CrackFileType_RULES:    {{ID: "rules-id", Name: "best64.rule", Sha2_256: rulesDigest, Type: clientpb.CrackFileType_RULES}},
		clientpb.CrackFileType_MARKOV_HCSTAT2: {{
			ID: "hcstat2-id", Name: "hashcat.hcstat2", Sha2_256: hcstat2Digest, Type: clientpb.CrackFileType_MARKOV_HCSTAT2,
		}},
	}
	requested := make([]clientpb.CrackFileType, 0)
	list := func(_ context.Context, request *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
		requested = append(requested, request.Type)
		return &clientpb.CrackFiles{Files: files[request.Type]}, nil
	}
	command := &clientpb.CrackCommand{
		AttackMode:          clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK,
		PositionalArguments: []string{"rockyou.txt", "?d?d?d"},
		RulesFile:           []byte("best64.rule"),
		RulesFilesV7:        [][]byte{[]byte("best64.rule"), []byte(":")},
		MarkovHcstat2:       []byte("hcstat2-id"),
	}

	if err := resolveManagedCrackFileReferences(context.Background(), command, list); err != nil {
		t.Fatalf("resolveManagedCrackFileReferences: %v", err)
	}
	if want := []string{"crackfile://wordlist/" + wordlistDigest, "?d?d?d"}; !slices.Equal(command.PositionalArguments, want) {
		t.Fatalf("PositionalArguments = %#v, want %#v", command.PositionalArguments, want)
	}
	wantRules := [][]byte{[]byte("crackfile://rules/" + rulesDigest), []byte(":")}
	if !slices.EqualFunc(command.RulesFilesV7, wantRules, func(left, right []byte) bool { return slices.Equal(left, right) }) {
		t.Fatalf("RulesFilesV7 = %#v, want %#v", command.RulesFilesV7, wantRules)
	}
	if !slices.Equal(command.RulesFile, wantRules[0]) {
		t.Fatalf("RulesFile = %q", command.RulesFile)
	}
	if want := "crackfile://hcstat2/" + hcstat2Digest; string(command.MarkovHcstat2) != want {
		t.Fatalf("MarkovHcstat2 = %q, want %q", command.MarkovHcstat2, want)
	}
	if want := []clientpb.CrackFileType{clientpb.CrackFileType_WORDLIST, clientpb.CrackFileType_RULES, clientpb.CrackFileType_MARKOV_HCSTAT2}; !slices.Equal(requested, want) {
		t.Fatalf("requested types = %#v, want %#v", requested, want)
	}
}

func TestResolveManagedCrackFileReferencesRespectsAttackOperandRoles(t *testing.T) {
	digestA := strings.Repeat("a", 64)
	digestB := strings.Repeat("b", 64)
	files := []*clientpb.CrackFile{
		{ID: "wordlist-a", Name: "first", Sha2_256: digestA, Type: clientpb.CrackFileType_WORDLIST},
		{ID: "wordlist-b", Name: "second", Sha2_256: digestB, Type: clientpb.CrackFileType_WORDLIST},
	}
	list := func(_ context.Context, request *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
		if request.Type != clientpb.CrackFileType_WORDLIST {
			t.Fatalf("requested file type = %s, want WORDLIST", request.Type)
		}
		return &clientpb.CrackFiles{Files: files}, nil
	}

	tests := []struct {
		name       string
		attackMode clientpb.CrackAttackMode
		inputs     []string
		want       []string
	}{
		{
			name:       "bruteforce mask collision",
			attackMode: clientpb.CrackAttackMode_BRUTEFORCE,
			inputs:     []string{"first"},
			want:       []string{"first"},
		},
		{
			name:       "wordlist mask hybrid",
			attackMode: clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK,
			inputs:     []string{"first", "second"},
			want:       []string{"crackfile://wordlist/" + digestA, "second"},
		},
		{
			name:       "mask wordlist hybrid",
			attackMode: clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST,
			inputs:     []string{"first", "second"},
			want:       []string{"first", "crackfile://wordlist/" + digestB},
		},
		{
			name:       "straight wordlists",
			attackMode: clientpb.CrackAttackMode_STRAIGHT,
			inputs:     []string{"first", "second"},
			want:       []string{"crackfile://wordlist/" + digestA, "crackfile://wordlist/" + digestB},
		},
		{
			name:       "combination wordlists",
			attackMode: clientpb.CrackAttackMode_COMBINATION,
			inputs:     []string{"first", "second"},
			want:       []string{"crackfile://wordlist/" + digestA, "crackfile://wordlist/" + digestB},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := &clientpb.CrackCommand{
				AttackMode:          test.attackMode,
				PositionalArguments: append([]string(nil), test.inputs...),
			}
			if err := resolveManagedCrackFileReferences(context.Background(), command, list); err != nil {
				t.Fatalf("resolveManagedCrackFileReferences: %v", err)
			}
			if !slices.Equal(command.PositionalArguments, test.want) {
				t.Fatalf("PositionalArguments = %#v, want %#v", command.PositionalArguments, test.want)
			}
		})
	}
}

func TestResolveManagedCrackFileReferencesPreservesInlineBytes(t *testing.T) {
	inline := []byte{0, 1, 2, 0xff}
	command := &clientpb.CrackCommand{RulesFilesV7: [][]byte{append([]byte(nil), inline...)}}
	list := func(_ context.Context, _ *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
		return &clientpb.CrackFiles{}, nil
	}
	if err := resolveManagedCrackFileReferences(context.Background(), command, list); err != nil {
		t.Fatal(err)
	}
	if len(command.RulesFilesV7) != 1 || !slices.Equal(command.RulesFilesV7[0], inline) {
		t.Fatalf("inline rules changed: %#v", command.RulesFilesV7)
	}
}

func TestResolveManagedCrackFileReferencesAcceptsCanonicalURIWithoutListing(t *testing.T) {
	uri := "crackfile://wordlist/" + strings.Repeat("d", 64)
	command := &clientpb.CrackCommand{PositionalArguments: []string{uri}}
	list := func(_ context.Context, _ *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
		t.Fatal("canonical URI should not require a manifest lookup")
		return nil, nil
	}
	if err := resolveManagedCrackFileReferences(context.Background(), command, list); err != nil {
		t.Fatal(err)
	}
	if command.PositionalArguments[0] != uri {
		t.Fatalf("URI changed to %q", command.PositionalArguments[0])
	}
}

func TestResolveManagedCrackFileReferencesRejectsWrongURIType(t *testing.T) {
	command := &clientpb.CrackCommand{RulesFilesV7: [][]byte{[]byte("crackfile://wordlist/" + strings.Repeat("e", 64))}}
	err := resolveManagedCrackFileReferences(context.Background(), command, func(context.Context, *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
		return nil, errors.New("must not be called")
	})
	if err == nil || !strings.Contains(err.Error(), "expected \"rules\"") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveManagedCrackFileReferencesRejectsAmbiguousName(t *testing.T) {
	command := &clientpb.CrackCommand{PositionalArguments: []string{"duplicate.txt"}}
	list := func(_ context.Context, request *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
		return &clientpb.CrackFiles{Files: []*clientpb.CrackFile{
			{Name: "duplicate.txt", Sha2_256: strings.Repeat("a", 64), Type: request.Type},
			{Name: "duplicate.txt", Sha2_256: strings.Repeat("b", 64), Type: request.Type},
		}}, nil
	}
	err := resolveManagedCrackFileReferences(context.Background(), command, list)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveManagedCrackFileReferencesRejectsInvalidDigest(t *testing.T) {
	command := &clientpb.CrackCommand{MarkovHcstat2: []byte("broken.hcstat2")}
	list := func(_ context.Context, request *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
		return &clientpb.CrackFiles{Files: []*clientpb.CrackFile{{
			Name: "broken.hcstat2", Sha2_256: "not-a-digest", Type: request.Type,
		}}}, nil
	}
	err := resolveManagedCrackFileReferences(context.Background(), command, list)
	if err == nil || !strings.Contains(err.Error(), "invalid SHA-256") {
		t.Fatalf("error = %v", err)
	}
}
