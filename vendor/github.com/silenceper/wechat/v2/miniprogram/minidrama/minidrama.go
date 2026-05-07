/*
 *   Copyright silenceper/wechat Author(https://silenceper.com/wechat/). All Rights Reserved.
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 *
 *    You can obtain one at https://github.com/silenceper/wechat.
 *
 */

package minidrama

import (
	"context"
	"strconv"

	"github.com/silenceper/wechat/v2/util"
)

// SingleFileUpload 单文件上传
func (s *MiniDrama) SingleFileUpload(ctx context.Context, in *SingleFileUploadRequest) (out SingleFileUploadResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, singleFileUpload); err != nil {
		return
	}
	var (
		fields = []util.MultipartFormField{
			{
				IsFile:    true,
				Fieldname: "media_data",
				Filename:  string(in.MediaData),
			}, {
				IsFile:    false,
				Fieldname: "media_name",
				Value:     []byte(in.MediaName),
			}, {
				IsFile:    false,
				Fieldname: "media_type",
				Value:     []byte(in.MediaType),
			},
		}
		response []byte
	)

	if in.CoverType != "" && in.CoverData != nil {
		fields = append(fields, util.MultipartFormField{
			IsFile:    false,
			Fieldname: "cover_type",
			Value:     []byte(in.CoverType),
		})
		fields = append(fields, util.MultipartFormField{
			IsFile:    true,
			Fieldname: "cover_data",
			Filename:  string(in.CoverData),
		})
	}

	if in.SourceContext != "" {
		fields = append(fields, util.MultipartFormField{
			IsFile:    false,
			Fieldname: "source_context",
			Value:     []byte(in.SourceContext),
		})
	}

	if response, err = util.PostMultipartForm(fields, address); err != nil {
		return
	}
	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "SingleFileUpload")
	return
}

// PullUpload 拉取上传
func (s *MiniDrama) PullUpload(ctx context.Context, in *PullUploadRequest) (out PullUploadResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, pullUpload); err != nil {
		return
	}
	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "PullUpload")
	return
}

// GetTask 查询任务状态
func (s *MiniDrama) GetTask(ctx context.Context, in *GetTaskRequest) (out GetTaskResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, getTask); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetTask")
	return
}

// ApplyUpload 申请分片上传
func (s *MiniDrama) ApplyUpload(ctx context.Context, in *ApplyUploadRequest) (out ApplyUploadResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, applyUpload); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "ApplyUpload")
	return
}

// UploadPart 上传分片
// Content-Type 需要指定为 multipart/form-data; boundary=<delimiter>，<箭头括号>表示必须替换为有效值的变量。
func (s *MiniDrama) UploadPart(ctx context.Context, in *UploadPartRequest) (out UploadPartResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, uploadPart); err != nil {
		return
	}

	var (
		fields = []util.MultipartFormField{
			{
				IsFile:    true,
				Fieldname: "data",
				Filename:  string(in.Data),
			}, {
				IsFile:    false,
				Fieldname: "upload_id",
				Value:     []byte(in.UploadID),
			}, {
				IsFile:    false,
				Fieldname: "part_number",
				Value:     []byte(strconv.Itoa(in.PartNumber)),
			}, {
				IsFile:    false,
				Fieldname: "resource_type",
				Value:     []byte(strconv.Itoa(in.PartNumber)),
			},
		}
		response []byte
	)
	if response, err = util.PostMultipartForm(fields, address); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "UploadPart")
	return
}

// CommitUpload 确认上传
func (s *MiniDrama) CommitUpload(ctx context.Context, in *CommitUploadRequest) (out CommitUploadResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, commitUpload); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "CommitUpload")
	return
}

// ListMedia 获取媒体列表
func (s *MiniDrama) ListMedia(ctx context.Context, in *ListMediaRequest) (out ListMediaResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, listMedia); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "ListMedia")
	return
}

// GetMedia 获取媒资详细信息
func (s *MiniDrama) GetMedia(ctx context.Context, in *GetMediaRequest) (out GetMediaResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, getMedia); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetMedia")
	return
}

// GetMediaLink 获取媒资播放链接
func (s *MiniDrama) GetMediaLink(ctx context.Context, in *GetMediaLinkRequest) (out GetMediaLinkResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, getMediaLink); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetMediaLink")
	return
}

// DeleteMedia 删除媒体
func (s *MiniDrama) DeleteMedia(ctx context.Context, in *DeleteMediaRequest) (out DeleteMediaResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, deleteMedia); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "DeleteMedia")
	return
}

// AuditDrama 审核剧本
func (s *MiniDrama) AuditDrama(ctx context.Context, in *AuditDramaRequest) (out AuditDramaResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, auditDrama); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "AuditDrama")
	return
}

// ListDramas 获取剧目列表
func (s *MiniDrama) ListDramas(ctx context.Context, in *ListDramasRequest) (out ListDramasResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, listDramas); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "ListDramas")
	return
}

// GetDrama 获取剧目信息
func (s *MiniDrama) GetDrama(ctx context.Context, in *GetDramaRequest) (out GetDramaResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, getDrama); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}
	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetDrama")
	return
}

// GetCdnUsageData 查询 CDN 用量数据
func (s *MiniDrama) GetCdnUsageData(ctx context.Context, in *GetCdnUsageDataRequest) (out GetCdnUsageDataResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, getCdnUsageData); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}
	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetCdnUsageData")
	return
}

// GetCdnLogs 查询 CDN 日志
func (s *MiniDrama) GetCdnLogs(ctx context.Context, in *GetCdnLogsRequest) (out GetCdnLogsResponse, err error) {
	var address string
	if address, err = s.requestAddress(ctx, getCdnLogs); err != nil {
		return
	}

	var response []byte
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}
	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetCdnLogs")
	return
}

// requestAddress 请求地址
func (s *MiniDrama) requestAddress(_ context.Context, url string) (string, error) {
	accessToken, err := s.ctx.GetAccessToken()
	if err != nil {
		return "", err
	}
	return url + accessToken, nil
}
