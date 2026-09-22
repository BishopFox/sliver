package crack

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

const crackFileScheme = "crackfile://"

type crackFilesListFunc func(context.Context, *clientpb.CrackFile) (*clientpb.CrackFiles, error)

// resolveManagedCrackFileReferences replaces exact server-side crack file
// names, IDs, and digests with the content-addressed URI understood by a
// crackstation. Inputs that do not match a managed file remain ordinary
// Hashcat operands, masks, paths, or inline rule content.
//
//nolint:gocyclo // Resolve every managed-file role together in one deterministic pass.
func resolveManagedCrackFileReferences(ctx context.Context, command *clientpb.CrackCommand, list crackFilesListFunc) error {
	if command == nil {
		return fmt.Errorf("missing crack command")
	}
	if list == nil {
		return fmt.Errorf("missing crack file lister")
	}

	wordlistIndexes := make([]int, 0, len(command.PositionalArguments))
	wordlistInputs := make([]string, 0, len(command.PositionalArguments))
	for index, argument := range command.PositionalArguments {
		// Only wordlist-role operands may be resolved by alias. An inline mask can
		// legitimately have the same text as a managed file name and must remain
		// an inline mask. Explicit crackfile URIs are still validated here so a
		// mistyped managed reference fails at the client boundary.
		if !crackOperandIsWordlist(command.AttackMode, index) && !strings.HasPrefix(argument, crackFileScheme) {
			continue
		}
		wordlistIndexes = append(wordlistIndexes, index)
		wordlistInputs = append(wordlistInputs, argument)
	}
	if len(wordlistInputs) > 0 {
		resolved, err := resolveCrackFileValues(ctx, wordlistInputs, clientpb.CrackFileType_WORDLIST, list)
		if err != nil {
			return err
		}
		for index, argument := range resolved {
			command.PositionalArguments[wordlistIndexes[index]] = argument
		}
	}
	legacyIdentify := command.Identify //nolint:staticcheck // Identify remains required for legacy wire compatibility.
	if len(command.PositionalArguments) == 0 && legacyIdentify != "" &&
		(crackOperandIsWordlist(command.AttackMode, 0) || strings.HasPrefix(legacyIdentify, crackFileScheme)) {
		resolved, err := resolveCrackFileValues(ctx, []string{legacyIdentify}, clientpb.CrackFileType_WORDLIST, list)
		if err != nil {
			return err
		}
		command.Identify = resolved[0] //nolint:staticcheck // Keep the resolved legacy operand synchronized for older servers.
	}

	rules := command.RulesFilesV7
	if len(rules) == 0 && len(command.RulesFile) > 0 {
		rules = [][]byte{command.RulesFile}
	}
	if len(rules) > 0 {
		values := make([]string, 0, len(rules))
		for _, rule := range rules {
			values = append(values, string(rule))
		}
		resolved, err := resolveCrackFileValues(ctx, values, clientpb.CrackFileType_RULES, list)
		if err != nil {
			return err
		}
		command.RulesFilesV7 = make([][]byte, 0, len(resolved))
		for _, rule := range resolved {
			command.RulesFilesV7 = append(command.RulesFilesV7, []byte(rule))
		}
		command.RulesFile = append([]byte(nil), command.RulesFilesV7[0]...)
	}

	if len(command.MarkovHcstat2) > 0 {
		resolved, err := resolveCrackFileValues(ctx, []string{string(command.MarkovHcstat2)}, clientpb.CrackFileType_MARKOV_HCSTAT2, list)
		if err != nil {
			return err
		}
		command.MarkovHcstat2 = []byte(resolved[0])
	}
	return nil
}

func crackOperandIsWordlist(attackMode clientpb.CrackAttackMode, index int) bool {
	switch attackMode {
	case clientpb.CrackAttackMode_STRAIGHT, clientpb.CrackAttackMode_COMBINATION:
		return true
	case clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK:
		return index == 0
	case clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST:
		return index == 1
	default:
		return false
	}
}

//nolint:gocyclo // Validation and alias disambiguation intentionally share the manifest lookup and preserve input ordering.
func resolveCrackFileValues(ctx context.Context, values []string, fileType clientpb.CrackFileType, list crackFilesListFunc) ([]string, error) {
	needsManifest := false
	for _, value := range values {
		if strings.HasPrefix(value, crackFileScheme) {
			if err := validateCrackFileURI(value, fileType); err != nil {
				return nil, err
			}
			continue
		}
		needsManifest = true
	}
	if !needsManifest {
		return append([]string(nil), values...), nil
	}

	manifest, err := list(ctx, &clientpb.CrackFile{Type: fileType})
	if err != nil {
		return nil, fmt.Errorf("list managed %s files: %w", crackFileTypeToken(fileType), err)
	}
	files := []*clientpb.CrackFile(nil)
	if manifest != nil {
		files = manifest.Files
	}
	resolved := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, crackFileScheme) {
			resolved = append(resolved, value)
			continue
		}
		matches := make([]*clientpb.CrackFile, 0, 1)
		for _, file := range files {
			if file == nil || file.Type != fileType {
				continue
			}
			if value == file.Name || value == file.ID || strings.EqualFold(value, file.Sha2_256) {
				matches = append(matches, file)
			}
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("managed %s file reference %q is ambiguous", crackFileTypeToken(fileType), value)
		}
		if len(matches) == 0 {
			resolved = append(resolved, value)
			continue
		}
		uri, err := crackFileURI(matches[0])
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, uri)
	}
	return resolved, nil
}

func crackFileURI(file *clientpb.CrackFile) (string, error) {
	if file == nil {
		return "", fmt.Errorf("missing managed crack file")
	}
	digest := strings.ToLower(strings.TrimSpace(file.Sha2_256))
	if err := validateCrackFileDigest(digest); err != nil {
		return "", fmt.Errorf("managed crack file %q has invalid SHA-256: %w", file.Name, err)
	}
	fileType := crackFileTypeToken(file.Type)
	if fileType == "" {
		return "", fmt.Errorf("managed crack file %q has unsupported type %s", file.Name, file.Type.String())
	}
	return crackFileScheme + fileType + "/" + digest, nil
}

func validateCrackFileURI(value string, expectedType clientpb.CrackFileType) error {
	remainder := strings.TrimPrefix(value, crackFileScheme)
	parts := strings.Split(remainder, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid managed crack file URI %q", value)
	}
	if parts[0] != crackFileTypeToken(expectedType) {
		return fmt.Errorf("managed crack file URI %q has type %q, expected %q", value, parts[0], crackFileTypeToken(expectedType))
	}
	if parts[1] != strings.ToLower(parts[1]) {
		return fmt.Errorf("managed crack file URI %q must use a lowercase SHA-256", value)
	}
	if err := validateCrackFileDigest(parts[1]); err != nil {
		return fmt.Errorf("invalid managed crack file URI %q: %w", value, err)
	}
	return nil
}

func validateCrackFileDigest(value string) error {
	if len(value) != 64 {
		return fmt.Errorf("SHA-256 must contain 64 hexadecimal characters")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("SHA-256 must contain 64 hexadecimal characters")
	}
	return nil
}

func crackFileTypeToken(fileType clientpb.CrackFileType) string {
	switch fileType {
	case clientpb.CrackFileType_WORDLIST:
		return "wordlist"
	case clientpb.CrackFileType_RULES:
		return "rules"
	case clientpb.CrackFileType_MARKOV_HCSTAT2:
		return "hcstat2"
	default:
		return ""
	}
}
