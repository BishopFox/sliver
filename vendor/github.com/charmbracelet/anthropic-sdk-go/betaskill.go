// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package anthropic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"slices"

	"github.com/charmbracelet/anthropic-sdk-go/internal/apiform"
	"github.com/charmbracelet/anthropic-sdk-go/internal/apijson"
	"github.com/charmbracelet/anthropic-sdk-go/internal/apiquery"
	"github.com/charmbracelet/anthropic-sdk-go/internal/requestconfig"
	"github.com/charmbracelet/anthropic-sdk-go/option"
	"github.com/charmbracelet/anthropic-sdk-go/packages/pagination"
	"github.com/charmbracelet/anthropic-sdk-go/packages/param"
	"github.com/charmbracelet/anthropic-sdk-go/packages/respjson"
)

// BetaSkillService contains methods and other services that help with interacting
// with the anthropic API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewBetaSkillService] method instead.
type BetaSkillService struct {
	Options  []option.RequestOption
	Versions BetaSkillVersionService
}

// NewBetaSkillService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewBetaSkillService(opts ...option.RequestOption) (r BetaSkillService) {
	r = BetaSkillService{}
	r.Options = opts
	r.Versions = NewBetaSkillVersionService(opts...)
	return
}

// Create Skill
func (r *BetaSkillService) New(ctx context.Context, params BetaSkillNewParams, opts ...option.RequestOption) (res *BetaSkillNewResponse, err error) {
	for _, v := range params.Betas {
		opts = append(opts, option.WithHeaderAdd("anthropic-beta", fmt.Sprintf("%v", v)))
	}
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithHeader("anthropic-beta", "skills-2025-10-02")}, opts...)
	path := "v1/skills?beta=true"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Get Skill
func (r *BetaSkillService) Get(ctx context.Context, skillID string, query BetaSkillGetParams, opts ...option.RequestOption) (res *BetaSkillGetResponse, err error) {
	for _, v := range query.Betas {
		opts = append(opts, option.WithHeaderAdd("anthropic-beta", fmt.Sprintf("%v", v)))
	}
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithHeader("anthropic-beta", "skills-2025-10-02")}, opts...)
	if skillID == "" {
		err = errors.New("missing required skill_id parameter")
		return
	}
	path := fmt.Sprintf("v1/skills/%s?beta=true", skillID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

// List Skills
func (r *BetaSkillService) List(ctx context.Context, params BetaSkillListParams, opts ...option.RequestOption) (res *pagination.PageCursor[BetaSkillListResponse], err error) {
	var raw *http.Response
	for _, v := range params.Betas {
		opts = append(opts, option.WithHeaderAdd("anthropic-beta", fmt.Sprintf("%v", v)))
	}
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithHeader("anthropic-beta", "skills-2025-10-02"), option.WithResponseInto(&raw)}, opts...)
	path := "v1/skills?beta=true"
	cfg, err := requestconfig.NewRequestConfig(ctx, http.MethodGet, path, params, &res, opts...)
	if err != nil {
		return nil, err
	}
	err = cfg.Execute()
	if err != nil {
		return nil, err
	}
	res.SetPageConfig(cfg, raw)
	return res, nil
}

// List Skills
func (r *BetaSkillService) ListAutoPaging(ctx context.Context, params BetaSkillListParams, opts ...option.RequestOption) *pagination.PageCursorAutoPager[BetaSkillListResponse] {
	return pagination.NewPageCursorAutoPager(r.List(ctx, params, opts...))
}

// Delete Skill
func (r *BetaSkillService) Delete(ctx context.Context, skillID string, body BetaSkillDeleteParams, opts ...option.RequestOption) (res *BetaSkillDeleteResponse, err error) {
	for _, v := range body.Betas {
		opts = append(opts, option.WithHeaderAdd("anthropic-beta", fmt.Sprintf("%v", v)))
	}
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithHeader("anthropic-beta", "skills-2025-10-02")}, opts...)
	if skillID == "" {
		err = errors.New("missing required skill_id parameter")
		return
	}
	path := fmt.Sprintf("v1/skills/%s?beta=true", skillID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodDelete, path, nil, &res, opts...)
	return
}

type BetaSkillNewResponse struct {
	// Unique identifier for the skill.
	//
	// The format and length of IDs may change over time.
	ID string `json:"id,required"`
	// ISO 8601 timestamp of when the skill was created.
	CreatedAt string `json:"created_at,required"`
	// Display title for the skill.
	//
	// This is a human-readable label that is not included in the prompt sent to the
	// model.
	DisplayTitle string `json:"display_title,required"`
	// The latest version identifier for the skill.
	//
	// This represents the most recent version of the skill that has been created.
	LatestVersion string `json:"latest_version,required"`
	// Source of the skill.
	//
	// This may be one of the following values:
	//
	// - `"custom"`: the skill was created by a user
	// - `"anthropic"`: the skill was created by Anthropic
	Source string `json:"source,required"`
	// Object type.
	//
	// For Skills, this is always `"skill"`.
	Type string `json:"type,required"`
	// ISO 8601 timestamp of when the skill was last updated.
	UpdatedAt string `json:"updated_at,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID            respjson.Field
		CreatedAt     respjson.Field
		DisplayTitle  respjson.Field
		LatestVersion respjson.Field
		Source        respjson.Field
		Type          respjson.Field
		UpdatedAt     respjson.Field
		ExtraFields   map[string]respjson.Field
		raw           string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaSkillNewResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaSkillNewResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaSkillGetResponse struct {
	// Unique identifier for the skill.
	//
	// The format and length of IDs may change over time.
	ID string `json:"id,required"`
	// ISO 8601 timestamp of when the skill was created.
	CreatedAt string `json:"created_at,required"`
	// Display title for the skill.
	//
	// This is a human-readable label that is not included in the prompt sent to the
	// model.
	DisplayTitle string `json:"display_title,required"`
	// The latest version identifier for the skill.
	//
	// This represents the most recent version of the skill that has been created.
	LatestVersion string `json:"latest_version,required"`
	// Source of the skill.
	//
	// This may be one of the following values:
	//
	// - `"custom"`: the skill was created by a user
	// - `"anthropic"`: the skill was created by Anthropic
	Source string `json:"source,required"`
	// Object type.
	//
	// For Skills, this is always `"skill"`.
	Type string `json:"type,required"`
	// ISO 8601 timestamp of when the skill was last updated.
	UpdatedAt string `json:"updated_at,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID            respjson.Field
		CreatedAt     respjson.Field
		DisplayTitle  respjson.Field
		LatestVersion respjson.Field
		Source        respjson.Field
		Type          respjson.Field
		UpdatedAt     respjson.Field
		ExtraFields   map[string]respjson.Field
		raw           string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaSkillGetResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaSkillGetResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaSkillListResponse struct {
	// Unique identifier for the skill.
	//
	// The format and length of IDs may change over time.
	ID string `json:"id,required"`
	// ISO 8601 timestamp of when the skill was created.
	CreatedAt string `json:"created_at,required"`
	// Display title for the skill.
	//
	// This is a human-readable label that is not included in the prompt sent to the
	// model.
	DisplayTitle string `json:"display_title,required"`
	// The latest version identifier for the skill.
	//
	// This represents the most recent version of the skill that has been created.
	LatestVersion string `json:"latest_version,required"`
	// Source of the skill.
	//
	// This may be one of the following values:
	//
	// - `"custom"`: the skill was created by a user
	// - `"anthropic"`: the skill was created by Anthropic
	Source string `json:"source,required"`
	// Object type.
	//
	// For Skills, this is always `"skill"`.
	Type string `json:"type,required"`
	// ISO 8601 timestamp of when the skill was last updated.
	UpdatedAt string `json:"updated_at,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID            respjson.Field
		CreatedAt     respjson.Field
		DisplayTitle  respjson.Field
		LatestVersion respjson.Field
		Source        respjson.Field
		Type          respjson.Field
		UpdatedAt     respjson.Field
		ExtraFields   map[string]respjson.Field
		raw           string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaSkillListResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaSkillListResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaSkillDeleteResponse struct {
	// Unique identifier for the skill.
	//
	// The format and length of IDs may change over time.
	ID string `json:"id,required"`
	// Deleted object type.
	//
	// For Skills, this is always `"skill_deleted"`.
	Type string `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaSkillDeleteResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaSkillDeleteResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaSkillNewParams struct {
	// Display title for the skill.
	//
	// This is a human-readable label that is not included in the prompt sent to the
	// model.
	DisplayTitle param.Opt[string] `json:"display_title,omitzero"`
	// Files to upload for the skill.
	//
	// All files must be in the same top-level directory and must include a SKILL.md
	// file at the root of that directory.
	Files []io.Reader `json:"files,omitzero" format:"binary"`
	// Optional header to specify the beta version(s) you want to use.
	Betas []AnthropicBeta `header:"anthropic-beta,omitzero" json:"-"`
	paramObj
}

func (r BetaSkillNewParams) MarshalMultipart() (data []byte, contentType string, err error) {
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	err = apiform.MarshalRoot(r, writer)
	if err == nil {
		err = apiform.WriteExtras(writer, r.ExtraFields())
	}
	if err != nil {
		writer.Close()
		return nil, "", err
	}
	err = writer.Close()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), writer.FormDataContentType(), nil
}

type BetaSkillGetParams struct {
	// Optional header to specify the beta version(s) you want to use.
	Betas []AnthropicBeta `header:"anthropic-beta,omitzero" json:"-"`
	paramObj
}

type BetaSkillListParams struct {
	// Pagination token for fetching a specific page of results.
	//
	// Pass the value from a previous response's `next_page` field to get the next page
	// of results.
	Page param.Opt[string] `query:"page,omitzero" json:"-"`
	// Filter skills by source.
	//
	// If provided, only skills from the specified source will be returned:
	//
	// - `"custom"`: only return user-created skills
	// - `"anthropic"`: only return Anthropic-created skills
	Source param.Opt[string] `query:"source,omitzero" json:"-"`
	// Number of results to return per page.
	//
	// Maximum value is 100. Defaults to 20.
	Limit param.Opt[int64] `query:"limit,omitzero" json:"-"`
	// Optional header to specify the beta version(s) you want to use.
	Betas []AnthropicBeta `header:"anthropic-beta,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [BetaSkillListParams]'s query parameters as `url.Values`.
func (r BetaSkillListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type BetaSkillDeleteParams struct {
	// Optional header to specify the beta version(s) you want to use.
	Betas []AnthropicBeta `header:"anthropic-beta,omitzero" json:"-"`
	paramObj
}
