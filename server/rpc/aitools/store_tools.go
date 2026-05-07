package aitools

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"

	clientcredentials "github.com/bishopfox/sliver/client/credentials"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	serverai "github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	serverloot "github.com/bishopfox/sliver/server/loot"
	"github.com/gofrs/uuid"
)

const aiLootReadDefaultMaxBytes = 64 * 1024

var (
	hashTypeAliases = map[string]string{
		"NTLM":   "MD4",
		"SHA224": "SHA2_224",
		"SHA256": "SHA2_256",
		"SHA384": "SHA2_384",
		"SHA512": "SHA2_512",
	}
	fileTypeAliases = map[string]string{
		"BIN": "BINARY",
		"TXT": "TEXT",
	}
)

type lootAddToolArgs struct {
	Name           string `json:"name,omitempty"`
	FileName       string `json:"file_name,omitempty"`
	FileTypeName   string `json:"file_type_name,omitempty"`
	FileTypeValue  *int32 `json:"file_type_value,omitempty"`
	OriginHostUUID string `json:"origin_host_uuid,omitempty"`
	Text           string `json:"text,omitempty"`
	DataBase64     string `json:"data_base64,omitempty"`
}

type lootReadToolArgs struct {
	LootID   string `json:"loot_id,omitempty"`
	MaxBytes int64  `json:"max_bytes,omitempty"`
}

type lootDeleteToolArgs struct {
	LootID string `json:"loot_id,omitempty"`
}

type credentialsAddToolArgs struct {
	Collection     string `json:"collection,omitempty"`
	Username       string `json:"username,omitempty"`
	Plaintext      string `json:"plaintext,omitempty"`
	Hash           string `json:"hash,omitempty"`
	HashTypeName   string `json:"hash_type_name,omitempty"`
	HashTypeValue  *int32 `json:"hash_type_value,omitempty"`
	OriginHostUUID string `json:"origin_host_uuid,omitempty"`
}

type credentialReadToolArgs struct {
	CredentialID string `json:"credential_id,omitempty"`
}

type credentialDeleteToolArgs struct {
	CredentialID string `json:"credential_id,omitempty"`
}

type lootSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	FileTypeName   string `json:"file_type_name"`
	FileTypeValue  int32  `json:"file_type_value"`
	OriginHostUUID string `json:"origin_host_uuid,omitempty"`
	Size           int64  `json:"size"`
	HasFile        bool   `json:"has_file"`
	FileName       string `json:"file_name,omitempty"`
}

type lootListResult struct {
	Loot  []lootSummary `json:"loot"`
	Count int           `json:"count"`
}

type lootReadResult struct {
	RequestedID       string `json:"requested_id"`
	RequestedMaxBytes int64  `json:"requested_max_bytes"`
	TotalByteLen      int    `json:"total_byte_len"`
	ReturnedByteLen   int    `json:"returned_byte_len"`
	Truncated         bool   `json:"truncated"`
	SHA256            string `json:"sha256,omitempty"`
	DataBase64        string `json:"data_base64,omitempty"`
	Text              string `json:"text,omitempty"`
	lootSummary
}

type lootDeleteResult struct {
	RequestedID string      `json:"requested_id"`
	Deleted     lootSummary `json:"deleted"`
}

type credentialSummary struct {
	ID             string `json:"id"`
	Collection     string `json:"collection,omitempty"`
	Username       string `json:"username,omitempty"`
	OriginHostUUID string `json:"origin_host_uuid,omitempty"`
	IsCracked      bool   `json:"is_cracked"`
	HasPlaintext   bool   `json:"has_plaintext"`
	HasHash        bool   `json:"has_hash"`
	HashTypeName   string `json:"hash_type_name,omitempty"`
	HashTypeValue  *int32 `json:"hash_type_value,omitempty"`
}

type credentialsListResult struct {
	Credentials []credentialSummary `json:"credentials"`
	Count       int                 `json:"count"`
}

type credentialReadResult struct {
	Plaintext string `json:"plaintext,omitempty"`
	Hash      string `json:"hash,omitempty"`
	credentialSummary
}

type credentialDeleteResult struct {
	RequestedID string            `json:"requested_id"`
	Deleted     credentialSummary `json:"deleted"`
}

func storeToolDefinitions() []serverai.AgenticToolDefinition {
	return []serverai.AgenticToolDefinition{
		{
			Name:        "loot_list",
			Description: "List loot entries from the server loot store. Results include metadata only, not file content.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "loot_read",
			Description: "Read one loot entry from the server loot store and return metadata plus up to max_bytes of file content. Defaults to 65536 bytes when max_bytes is omitted.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"loot_id":   map[string]any{"type": "string", "description": "Loot ID or unique ID prefix to read."},
					"max_bytes": map[string]any{"type": "integer", "description": "Optional maximum number of bytes to return from the loot file content."},
				},
				"required":             []string{"loot_id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "loot_add",
			Description: "Add a new entry to the server loot store from inline text or base64 bytes. Provide either text or data_base64. file_name defaults to name, and file_type defaults to TEXT for text or BINARY for base64 content.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data_base64":      map[string]any{"type": "string", "description": "Binary loot content encoded as base64. Provide either text or data_base64."},
					"file_name":        map[string]any{"type": "string", "description": "Optional filename stored with the loot entry. Defaults to name."},
					"file_type_name":   map[string]any{"type": "string", "description": "Optional file type name such as TEXT or BINARY."},
					"file_type_value":  map[string]any{"type": "integer", "description": "Optional numeric file type enum value."},
					"name":             map[string]any{"type": "string", "description": "Display name for the loot entry."},
					"origin_host_uuid": map[string]any{"type": "string", "description": "Optional origin host UUID for the loot entry."},
					"text":             map[string]any{"type": "string", "description": "Text loot content. Provide either text or data_base64."},
				},
				"required":             []string{"name"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "loot_delete",
			Description: "Delete one loot entry from the server loot store using a full ID or unique prefix.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"loot_id": map[string]any{"type": "string", "description": "Loot ID or unique ID prefix to delete."},
				},
				"required":             []string{"loot_id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "credentials_list",
			Description: "List credentials from the server credentials store. Results include metadata only, not plaintext or hash bodies.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "credentials_read",
			Description: "Read one credential from the server credentials store using a full ID or unique prefix. Returns the stored plaintext and hash values when present.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"credential_id": map[string]any{"type": "string", "description": "Credential ID or unique ID prefix to read."},
				},
				"required":             []string{"credential_id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "credentials_add",
			Description: "Add one credential to the server credentials store. If hash_type is omitted and hash is provided, the server sniffs the hash type automatically.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"collection":       map[string]any{"type": "string", "description": "Optional credential collection or source label."},
					"hash":             map[string]any{"type": "string", "description": "Optional password hash."},
					"hash_type_name":   map[string]any{"type": "string", "description": "Optional hash type name such as MD4, NTLM, or SHA2_256."},
					"hash_type_value":  map[string]any{"type": "integer", "description": "Optional numeric hash type enum value."},
					"origin_host_uuid": map[string]any{"type": "string", "description": "Optional origin host UUID for the credential."},
					"plaintext":        map[string]any{"type": "string", "description": "Optional plaintext password or secret."},
					"username":         map[string]any{"type": "string", "description": "Optional username or account identifier."},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "credentials_delete",
			Description: "Delete one credential from the server credentials store using a full ID or unique prefix.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"credential_id": map[string]any{"type": "string", "description": "Credential ID or unique ID prefix to delete."},
				},
				"required":             []string{"credential_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (e *executor) callStoreTool(ctx context.Context, name string, arguments string) (string, bool, error) {
	switch strings.TrimSpace(name) {
	case "loot_list":
		result, err := e.callLootList(ctx)
		return result, true, err
	case "loot_read":
		var args lootReadToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callLootRead(ctx, args)
		return result, true, err
	case "loot_add":
		var args lootAddToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callLootAdd(ctx, args)
		return result, true, err
	case "loot_delete":
		var args lootDeleteToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callLootDelete(ctx, args)
		return result, true, err
	case "credentials_list":
		result, err := e.callCredentialsList(ctx)
		return result, true, err
	case "credentials_read":
		var args credentialReadToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callCredentialsRead(ctx, args)
		return result, true, err
	case "credentials_add":
		var args credentialsAddToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callCredentialsAdd(ctx, args)
		return result, true, err
	case "credentials_delete":
		var args credentialDeleteToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callCredentialsDelete(ctx, args)
		return result, true, err
	default:
		return "", false, nil
	}
}

func (e *executor) callLootList(_ context.Context) (string, error) {
	allLoot := serverloot.GetLootStore().All()
	if allLoot == nil {
		return "", fmt.Errorf("failed to load loot store")
	}

	summaries := make([]lootSummary, 0, len(allLoot.GetLoot()))
	for _, entry := range allLoot.GetLoot() {
		if entry == nil {
			continue
		}
		summaries = append(summaries, lootSummaryFromProtobuf(entry))
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Name == summaries[j].Name {
			return summaries[i].ID < summaries[j].ID
		}
		return summaries[i].Name < summaries[j].Name
	})

	return marshalToolResult(lootListResult{
		Loot:  summaries,
		Count: len(summaries),
	})
}

func (e *executor) callLootAdd(_ context.Context, args lootAddToolArgs) (string, error) {
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	data, err := decodeLootContent(args.Text, args.DataBase64)
	if err != nil {
		return "", err
	}
	fileType, err := resolveLootFileType(args.FileTypeName, args.FileTypeValue, args.Text, args.DataBase64)
	if err != nil {
		return "", err
	}

	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		fileName = name
	}

	entry, err := serverloot.GetLootStore().Add(&clientpb.Loot{
		Name:           name,
		FileType:       fileType,
		OriginHostUUID: strings.TrimSpace(args.OriginHostUUID),
		File: &commonpb.File{
			Name: fileName,
			Data: data,
		},
	})
	if err != nil {
		return "", err
	}
	return marshalToolResult(lootSummaryFromProtobuf(entry))
}

func (e *executor) callLootRead(_ context.Context, args lootReadToolArgs) (string, error) {
	record, err := resolveLootRecord(args.LootID)
	if err != nil {
		return "", err
	}

	entry, err := serverloot.GetLootStore().GetContent(record.ID.String(), true)
	if err != nil {
		return "", err
	}

	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = aiLootReadDefaultMaxBytes
	}

	data := []byte(nil)
	if entry.GetFile() != nil {
		data = entry.GetFile().GetData()
	}
	totalLen := len(data)
	returned := data
	if int64(len(returned)) > maxBytes {
		returned = returned[:maxBytes]
	}

	result := lootReadResult{
		RequestedID:       strings.TrimSpace(args.LootID),
		RequestedMaxBytes: maxBytes,
		TotalByteLen:      totalLen,
		ReturnedByteLen:   len(returned),
		Truncated:         len(returned) < totalLen,
		lootSummary:       lootSummaryFromProtobuf(entry),
	}
	if totalLen > 0 {
		result.SHA256 = sha256Hex(data)
		result.DataBase64 = base64.StdEncoding.EncodeToString(returned)
		if len(returned) > 0 {
			result.Text, _ = bytesToTextAndBase64(returned)
		}
	}
	return marshalToolResult(result)
}

func (e *executor) callLootDelete(_ context.Context, args lootDeleteToolArgs) (string, error) {
	record, err := resolveLootRecord(args.LootID)
	if err != nil {
		return "", err
	}

	entry, err := serverloot.GetLootStore().GetContent(record.ID.String(), false)
	if err != nil {
		return "", err
	}
	if err := serverloot.GetLootStore().Rm(record.ID.String()); err != nil {
		return "", err
	}

	return marshalToolResult(lootDeleteResult{
		RequestedID: strings.TrimSpace(args.LootID),
		Deleted:     lootSummaryFromProtobuf(entry),
	})
}

func (e *executor) callCredentialsList(_ context.Context) (string, error) {
	dbCreds := []*models.Credential{}
	if err := db.Session().Where(&models.Credential{}).Find(&dbCreds).Error; err != nil {
		return "", err
	}

	summaries := make([]credentialSummary, 0, len(dbCreds))
	for _, cred := range dbCreds {
		if cred == nil {
			continue
		}
		summaries = append(summaries, credentialSummaryFromProtobuf(cred.ToProtobuf()))
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Collection == summaries[j].Collection {
			if summaries[i].Username == summaries[j].Username {
				return summaries[i].ID < summaries[j].ID
			}
			return summaries[i].Username < summaries[j].Username
		}
		return summaries[i].Collection < summaries[j].Collection
	})

	return marshalToolResult(credentialsListResult{
		Credentials: summaries,
		Count:       len(summaries),
	})
}

func (e *executor) callCredentialsAdd(_ context.Context, args credentialsAddToolArgs) (string, error) {
	if strings.TrimSpace(args.Username) == "" && args.Plaintext == "" && args.Hash == "" {
		return "", fmt.Errorf("at least one of username, plaintext, or hash is required")
	}

	hashType, err := resolveHashType(args.HashTypeName, args.HashTypeValue, args.Hash)
	if err != nil {
		return "", err
	}

	record := &models.Credential{
		Collection:     strings.TrimSpace(args.Collection),
		Username:       strings.TrimSpace(args.Username),
		Plaintext:      args.Plaintext,
		Hash:           args.Hash,
		HashType:       int32(hashType),
		IsCracked:      args.Plaintext != "" && args.Hash != "",
		OriginHostUUID: uuid.FromStringOrNil(strings.TrimSpace(args.OriginHostUUID)),
	}
	if err := db.Session().Create(record).Error; err != nil {
		return "", err
	}

	return marshalToolResult(credentialReadResultFromProtobuf(record.ToProtobuf()))
}

func (e *executor) callCredentialsRead(_ context.Context, args credentialReadToolArgs) (string, error) {
	record, err := resolveCredentialRecord(args.CredentialID)
	if err != nil {
		return "", err
	}
	return marshalToolResult(credentialReadResultFromProtobuf(record.ToProtobuf()))
}

func (e *executor) callCredentialsDelete(_ context.Context, args credentialDeleteToolArgs) (string, error) {
	record, err := resolveCredentialRecord(args.CredentialID)
	if err != nil {
		return "", err
	}

	summary := credentialSummaryFromProtobuf(record.ToProtobuf())
	if err := db.Session().Delete(record).Error; err != nil {
		return "", err
	}

	return marshalToolResult(credentialDeleteResult{
		RequestedID: strings.TrimSpace(args.CredentialID),
		Deleted:     summary,
	})
}

func decodeLootContent(text string, dataBase64 string) ([]byte, error) {
	hasText := text != ""
	hasData := strings.TrimSpace(dataBase64) != ""
	switch {
	case hasText && hasData:
		return nil, fmt.Errorf("provide only one of text or data_base64")
	case hasText:
		return []byte(text), nil
	case hasData:
		data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(dataBase64))
		if err != nil {
			return nil, fmt.Errorf("invalid data_base64: %w", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("text or data_base64 is required")
	}
}

func resolveLootFileType(name string, value *int32, text string, dataBase64 string) (clientpb.FileType, error) {
	valueType, hasValue, err := parseFileTypeValue(value)
	if err != nil {
		return 0, err
	}
	nameType, hasName, err := parseFileTypeName(name)
	if err != nil {
		return 0, err
	}
	if hasValue && hasName {
		if valueType != nameType {
			return 0, fmt.Errorf("file_type_name and file_type_value refer to different file types")
		}
		return valueType, nil
	}
	if hasValue {
		return valueType, nil
	}
	if hasName {
		return nameType, nil
	}
	if text != "" {
		return clientpb.FileType_TEXT, nil
	}
	if strings.TrimSpace(dataBase64) != "" {
		return clientpb.FileType_BINARY, nil
	}
	return 0, fmt.Errorf("file content is required")
}

func resolveHashType(name string, value *int32, hash string) (clientpb.HashType, error) {
	valueType, hasValue, err := parseHashTypeValue(value)
	if err != nil {
		return 0, err
	}
	nameType, hasName, err := parseHashTypeName(name)
	if err != nil {
		return 0, err
	}
	if hasValue && hasName {
		if valueType != nameType {
			return 0, fmt.Errorf("hash_type_name and hash_type_value refer to different hash types")
		}
		return valueType, nil
	}
	if hasValue {
		return valueType, nil
	}
	if hasName {
		return nameType, nil
	}
	if hash != "" {
		return clientcredentials.SniffHashType(hash), nil
	}
	return clientpb.HashType_MD5, nil
}

func parseFileTypeName(name string) (clientpb.FileType, bool, error) {
	normalized := normalizeEnumName(name)
	if normalized == "" {
		return 0, false, nil
	}
	if alias, ok := fileTypeAliases[normalized]; ok {
		normalized = alias
	}
	value, ok := clientpb.FileType_value[normalized]
	if !ok {
		return 0, false, fmt.Errorf("unsupported file_type_name %q", name)
	}
	return clientpb.FileType(value), true, nil
}

func parseFileTypeValue(value *int32) (clientpb.FileType, bool, error) {
	if value == nil {
		return 0, false, nil
	}
	if _, ok := clientpb.FileType_name[*value]; !ok {
		return 0, false, fmt.Errorf("unsupported file_type_value %d", *value)
	}
	return clientpb.FileType(*value), true, nil
}

func parseHashTypeName(name string) (clientpb.HashType, bool, error) {
	normalized := normalizeEnumName(name)
	if normalized == "" {
		return 0, false, nil
	}
	if alias, ok := hashTypeAliases[normalized]; ok {
		normalized = alias
	}
	value, ok := clientpb.HashType_value[normalized]
	if !ok {
		return 0, false, fmt.Errorf("unsupported hash_type_name %q", name)
	}
	return clientpb.HashType(value), true, nil
}

func parseHashTypeValue(value *int32) (clientpb.HashType, bool, error) {
	if value == nil {
		return 0, false, nil
	}
	if _, ok := clientpb.HashType_name[*value]; !ok {
		return 0, false, fmt.Errorf("unsupported hash_type_value %d", *value)
	}
	return clientpb.HashType(*value), true, nil
}

func normalizeEnumName(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	replacer := strings.NewReplacer("-", "_", " ", "_")
	return replacer.Replace(name)
}

func resolveLootRecord(id string) (*models.Loot, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("loot_id is required")
	}

	if lootID := uuid.FromStringOrNil(id); lootID != uuid.Nil {
		record := &models.Loot{}
		err := db.Session().Where(&models.Loot{ID: lootID}).First(record).Error
		if err == nil {
			return record, nil
		}
		if !errors.Is(err, db.ErrRecordNotFound) {
			return nil, err
		}
	}

	records := []*models.Loot{}
	if err := db.Session().Where(&models.Loot{}).Find(&records).Error; err != nil {
		return nil, err
	}

	matches := make([]*models.Loot, 0, 1)
	for _, record := range records {
		if record != nil && strings.HasPrefix(record.ID.String(), id) {
			matches = append(matches, record)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("loot %q not found", id)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("loot ID prefix %q is ambiguous: %s", id, joinRecordIDsLoot(matches))
	}
}

func resolveCredentialRecord(id string) (*models.Credential, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("credential_id is required")
	}

	if credentialID := uuid.FromStringOrNil(id); credentialID != uuid.Nil {
		record := &models.Credential{}
		err := db.Session().Where(&models.Credential{ID: credentialID}).First(record).Error
		if err == nil {
			return record, nil
		}
		if !errors.Is(err, db.ErrRecordNotFound) {
			return nil, err
		}
	}

	records := []*models.Credential{}
	if err := db.Session().Where(&models.Credential{}).Find(&records).Error; err != nil {
		return nil, err
	}

	matches := make([]*models.Credential, 0, 1)
	for _, record := range records {
		if record != nil && strings.HasPrefix(record.ID.String(), id) {
			matches = append(matches, record)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("credential %q not found", id)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("credential ID prefix %q is ambiguous: %s", id, joinRecordIDsCredential(matches))
	}
}

func joinRecordIDsLoot(records []*models.Loot) string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		if record == nil || record.ID == uuid.Nil {
			continue
		}
		ids = append(ids, record.ID.String())
	}
	return joinRecordIDs(ids)
}

func joinRecordIDsCredential(records []*models.Credential) string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		if record == nil || record.ID == uuid.Nil {
			continue
		}
		ids = append(ids, record.ID.String())
	}
	return joinRecordIDs(ids)
}

func joinRecordIDs(ids []string) string {
	sort.Strings(ids)
	if len(ids) > 5 {
		ids = append(ids[:5], "...")
	}
	return strings.Join(ids, ", ")
}

func lootSummaryFromProtobuf(entry *clientpb.Loot) lootSummary {
	summary := lootSummary{
		ID:             strings.TrimSpace(entry.GetID()),
		Name:           strings.TrimSpace(entry.GetName()),
		FileTypeName:   fileTypeName(entry.GetFileType()),
		FileTypeValue:  int32(entry.GetFileType()),
		OriginHostUUID: cleanUUIDString(entry.GetOriginHostUUID()),
		Size:           entry.GetSize(),
		HasFile:        entry.GetFile() != nil,
	}
	if entry.GetFile() != nil {
		summary.FileName = strings.TrimSpace(entry.GetFile().GetName())
	}
	return summary
}

func credentialSummaryFromProtobuf(cred *clientpb.Credential) credentialSummary {
	summary := credentialSummary{
		ID:             strings.TrimSpace(cred.GetID()),
		Collection:     strings.TrimSpace(cred.GetCollection()),
		Username:       strings.TrimSpace(cred.GetUsername()),
		OriginHostUUID: cleanUUIDString(cred.GetOriginHostUUID()),
		IsCracked:      cred.GetIsCracked(),
		HasPlaintext:   cred.GetPlaintext() != "",
		HasHash:        cred.GetHash() != "",
	}
	if summary.HasHash {
		value := int32(cred.GetHashType())
		summary.HashTypeName = hashTypeName(cred.GetHashType())
		summary.HashTypeValue = &value
	}
	return summary
}

func credentialReadResultFromProtobuf(cred *clientpb.Credential) credentialReadResult {
	return credentialReadResult{
		Plaintext:         cred.GetPlaintext(),
		Hash:              cred.GetHash(),
		credentialSummary: credentialSummaryFromProtobuf(cred),
	}
}

func fileTypeName(fileType clientpb.FileType) string {
	if name, ok := clientpb.FileType_name[int32(fileType)]; ok {
		return name
	}
	return fmt.Sprintf("%d", fileType)
}

func hashTypeName(hashType clientpb.HashType) string {
	if name, ok := clientpb.HashType_name[int32(hashType)]; ok {
		return name
	}
	return fmt.Sprintf("%d", hashType)
}

func cleanUUIDString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == uuid.Nil.String() {
		return ""
	}
	return raw
}
