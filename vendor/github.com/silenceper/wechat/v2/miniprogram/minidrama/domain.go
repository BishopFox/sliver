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
	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

// MiniDrama mini program entertainment live broadcast related
type MiniDrama struct {
	ctx *context.Context
}

// ErrCode error code
type ErrCode int

// SingleFileUploadRequest 单文件上传请求
// Content-Type 需要指定为 multipart/form-data; boundary=<delimiter>
// <箭头括号> 表示必须替换为有效值的变量。
// 不填写 cover_type，cover_data 字段时默认截取视频首帧作为视频封面。
type SingleFileUploadRequest struct {
	MediaName     string `json:"media_name"`               // 媒体文件名称 文件名，需按照“剧目名 - 对应剧集数”格式命名文件，示例值："我的演艺 - 第 1 集"。
	MediaType     string `json:"media_type"`               // 媒体文件类型 视频格式，支持：MP4，TS，MOV，MXF，MPG，FLV，WMV，AVI，M4V，F4V，MPEG，3GP，ASF，MKV，示例值："MP4"。
	MediaData     []byte `json:"media_data"`               // 媒体文件数据 视频文件内容，二进制。
	CoverType     string `json:"cover_type,omitempty"`     // 视频封面图片格式，支持：JPG、JPEG、PNG、BMP、TIFF、AI、CDR、EPS、TIF，示例值："JPG"。
	CoverData     []byte `json:"cover_data,omitempty"`     // 视频封面图片内容，二进制。
	SourceContext string `json:"source_context,omitempty"` // 来源上下文，会在上传完成事件中透传给开发者。
}

// SingleFileUploadResponse 单文件上传响应
type SingleFileUploadResponse struct {
	util.CommonError
	MediaID int64 `json:"media_id"` // 媒体文件唯一标识，用于发布视频。
}

// PullUploadRequest 拉取上传请求
// 不填写 cover_url 字段时默认截取视频首帧作为封面。
// Content-Type 需要指定为 application/json
// 该接口为异步接口，上传完成会推送上传完成事件到开发者服务器，开发者也可以调用"查询任务"接口来轮询上传结果。
type PullUploadRequest struct {
	MediaName     string `json:"media_name"`               // 媒体文件名称 文件名，需按照“剧目名 - 对应剧集数”格式命名文件，示例值："我的演艺 - 第 1 集"。
	MediaURL      string `json:"media_url"`                // 视频 URL，示例值："https://developers.weixin.qq.com/test.mp4"。
	CoverURL      string `json:"cover_url,omitempty"`      // 视频封面 URL，示例值："https://developers.weixin.qq.com/test.jpg"。
	SourceContext string `json:"source_context,omitempty"` // 来源上下文，会在上传完成事件中透传给开发者。
}

// PullUploadResponse 拉取上传响应
type PullUploadResponse struct {
	util.CommonError
	TaskID int64 `json:"task_id"` // 任务 ID，用于查询拉取上传任务的结果。
}

// GetTaskRequest 查询任务请求
// 该接口用于查询拉取上传的任务状态。
// Content-Type 需要指定为 application/json。
type GetTaskRequest struct {
	TaskID int64 `json:"task_id"` // 任务 ID，用于查询拉取上传任务的结果。
}

// GetTaskResponse 查询任务响应
type GetTaskResponse struct {
	util.CommonError
	TaskInfo TaskInfo `json:"task_info"` // 任务信息。
}

// TaskInfo 任务信息
type TaskInfo struct {
	ID         int64  `json:"id"`          // 任务 ID。
	TaskType   int    `json:"task_type"`   // 任务类型，1：拉取上传任务。
	Status     int    `json:"status"`      // 任务状态枚举值：1. 等待中；2. 正在处理；3. 已完成；4. 失败。
	ErrCode    int    `json:"errcode"`     // 任务错误码，0 表示成功，其它表示失败。
	ErrMsg     string `json:"errmsg"`      // 任务错误原因。
	CreateTime int64  `json:"create_time"` // 任务创建时间，时间戳，单位：秒。
	FinishTime int64  `json:"finish_time"` // 任务完成时间，时间戳，单位：秒。
	MediaID    int64  `json:"media_id"`    // 媒体文件唯一标识，用于发布视频。
}

// ApplyUploadRequest 申请上传请求
// 上传大文件时需使用分片上传方式，分为 3 个步骤：
//
// 申请分片上传，确定文件名、格式类型，返回 upload_id，唯一标识本次分片上传。
// 上传分片，多次调用上传文件分片，需要携带 part_number 和 upload_id，其中 part_number 为分片的编号，支持乱序上传。当传入 part_number 和 upload_id 都相同的时候，后发起上传请求的分片将覆盖之前的分片。
// 确认分片上传，当上传完所有分片后，需要完成整个文件的合并。请求体中需要给出每一个分片的 part_number 和 etag，用来校验分片的准确性，最后返回文件的 media_id。
// 如果填写了 cover_type，表明本次分片上传除上传媒体文件外还需要上传封面图片，不填写 cover_type 则默认截取视频首帧作为封面。
// Content-Type 需要指定为 application/json。
type ApplyUploadRequest struct {
	MediaName     string `json:"media_name"`               // 媒体文件名称 文件名，需按照“剧目名 - 对应剧集数”格式命名文件，示例值："我的演艺 - 第 1 集"。
	MediaType     string `json:"media_type"`               // 媒体文件类型 视频格式，支持：MP4，TS，MOV，MXF，MPG，FLV，WMV，AVI，M4V，F4V，MPEG，3GP，ASF，MKV，示例值："MP4"。
	CoverType     string `json:"cover_type,omitempty"`     // 视频封面图片格式，支持：JPG、JPEG、PNG、BMP、TIFF、AI、CDR、EPS、TIF，示例值："JPG"。
	SourceContext string `json:"source_context,omitempty"` // 来源上下文，会在上传完成事件中透传给开发者。
}

// ApplyUploadResponse 申请上传响应
type ApplyUploadResponse struct {
	util.CommonError
	UploadID string `json:"upload_id"` // 本次分片上传的唯一标识。
}

// UploadPartRequest 上传分片请求
// 将文件的其中一个分片上传到平台，最多支持 100 个分片，每个分片大小为 5MB，最后一个分片可以小于 5MB。该接口适用于视频和封面图片。视频最大支持 500MB，封面图片最大支持 10MB。
// 调用该接口之前必须先调用申请分片上传接口。
// 在申请分片上传时，如果不填写 cover_type，则默认截取视频首帧作为封面。
// Content-Type 需要指定为 multipart/form-data; boundary=<delimiter>，<箭头括号>表示必须替换为有效值的变量。
// part_number 从 1 开始。如除了上传视频外还需要上传封面图片，则封面图片的 part_number 需重新从 1 开始编号。
type UploadPartRequest struct {
	UploadID     string `json:"upload_id"`     // 一次分片上传的唯一标识，由申请分片上传接口返回。
	PartNumber   int    `json:"part_number"`   // 本次上传的分片的编号，范围在 1 - 100。
	ResourceType int    `json:"resource_type"` // 指定该分片属于视频还是封面图片的枚举值：1. 视频，2. 封面图片。
	Data         []byte `json:"data"`          // 分片内容，二进制。
}

// UploadPartResponse 上传分片响应
type UploadPartResponse struct {
	util.CommonError
	ETag string `json:"etag"` // 上传分片成功后返回的分片标识，用于后续确认分片上传接口。
}

// CommitUploadRequest 确认分片上传请求
// 该接口用于完成整个分片上传流程，合并所有文件分片，确认媒体文件（和封面图片文件）上传到平台的结果，返回文件的 ID。请求中需要给出每一个分片的 part_number 和 etag，用来校验分片的准确性。
// 注意事项
// Content-Type 需要指定为 application/json。
// 调用该接口之前必须先调用申请分片上传接口以及上传分片接口。
// 如本次分片上传除上传媒体文件外还需要上传封面图片，则请求中还需提供 cover_part_infos 字段以用于合并封面图片文件分片。
// 请求中 media_part_infos 和 cover_part_infos 字段必须按 part_number 从小到大排序，part_number 必须从 1 开始，连续且不重复。
type CommitUploadRequest struct {
	UploadID       string      `json:"upload_id"`
	MediaPartInfos []*PartInfo `json:"media_part_infos"`
	CoverPartInfos []*PartInfo `json:"cover_part_infos,omitempty"`
}

// PartInfo 分片信息
type PartInfo struct {
	PartNumber int    `json:"part_number"` // 分片编号。
	Etag       string `json:"etag"`        // 使用上传分片接口上传成功后返回的 etag 的值
}

// CommitUploadResponse 确认分片上传响应
type CommitUploadResponse struct {
	util.CommonError
	MediaID int64 `json:"media_id"` // 媒体文件唯一标识，用于发布视频。
}

// ListMediaRequest 查询媒体列表请求
// 该接口用于查询已经上传到平台的媒体文件列表。
// 注意事项
// Content-Type 需要指定为 application/json。
// 本接口返回的视频或图片链接均为临时链接，不应将其保存下来。
// media_name 参数支持模糊匹配，当需要模糊匹配时可以在前面或后面加上 %，否则为精确匹配。例如 "test%" 可以匹配到 "test123", "testxxx", "test"。
// 调用方式
type ListMediaRequest struct {
	DramaID   int64  `json:"drama_id,omitempty"`   // 剧目 ID，可通过查询剧目列表接口获取。
	MediaName string `json:"media_name,omitempty"` // 媒体文件名称，可通过查询媒体列表接口获取，模糊匹配。
	StartTime int64  `json:"start_time,omitempty"` // 媒资上传时间>=start_time，Unix 时间戳，单位：秒。
	EndTime   int64  `json:"end_time,omitempty"`   // 媒资上传时间<end_time，Unix 时间戳，单位：秒。
	Limit     int    `json:"limit,omitempty"`      // 分页拉取的最大返回结果数。默认值：100；最大值：100。
	Offset    int    `json:"offset,omitempty"`     // 分页拉取的起始偏移量。默认值：0。
}

// MediaInfo 媒体信息
type MediaInfo struct {
	MediaID     int64             `json:"media_id"`     // 媒资文件 id。
	CreateTime  int64             `json:"create_time"`  // 	上传时间，时间戳。
	ExpireTime  int64             `json:"expire_time"`  // 过期时间，时间戳。
	DramaID     int64             `json:"drama_id"`     // 所属剧目 id。
	FileSize    int64             `json:"file_size"`    // 媒资文件大小，单位：字节。
	Duration    int64             `json:"duration"`     // 播放时长，单位：秒。
	Name        string            `json:"name"`         // 媒资文件名。
	Description string            `json:"description"`  // 描述。
	CoverURL    string            `json:"cover_url"`    // 封面图临时链接。
	OriginalURL string            `json:"original_url"` // 原始视频临时链接。
	Mp4URL      string            `json:"mp4_url"`      // mp4 格式临时链接。
	HlsURL      string            `json:"hls_url"`      // hls 格式临时链接。
	AuditDetail *MediaAuditDetail `json:"audit_detail"` // 审核信息。
}

// MediaAuditDetail 媒体审核详情
type MediaAuditDetail struct {
	Status                 int      `json:"status"`                    // 审核状态 0 为无效值；1 为审核中；2 为审核驳回；3 为审核通过；4 为驳回重填。需要注意可能存在单个剧集的状态为审核通过，但是剧目整体是未通过的情况，而能不能获取播放链接取决于剧目的审核状态。
	CreateTime             int      `json:"create_time"`               // 提审时间戳。
	AuditTime              int      `json:"audit_time"`                // 审核时间戳。
	Reason                 string   `json:"reason"`                    // 审核备注。该值可能为空。
	EvidenceMaterialIDList []string `json:"evidence_material_id_list"` // 审核证据截图 id 列表，截图 id 可以用作 get_material 接口的参数来获得截图内容。
}

// ListMediaResponse 查询媒体列表响应
type ListMediaResponse struct {
	util.CommonError
	MediaInfoList []*MediaInfo `json:"media_info_list"` // 媒体信息列表。
}

// GetMediaRequest 获取媒体请求
// 该接口用于获取已上传到平台的指定媒资信息，用于开发者后台管理使用。用于给用户客户端播放的链接应该使用 getmedialink 接口获取。
// Content-Type 需要指定为 application/json。
// 本接口返回的视频或图片链接均为临时链接，不应将其保存下来。
type GetMediaRequest struct {
	MediaID int64 `json:"media_id"` // 媒资文件 id。
}

// GetMediaResponse 获取媒体响应
type GetMediaResponse struct {
	util.CommonError
	MediaInfo MediaInfo `json:"media_info"` // 媒体信息。
}

// GetMediaLinkRequest 获取媒体链接请求
// 该接口用于获取视频临时播放链接，用于给用户的播放使用。只有审核通过的视频才能通过该接口获取播放链接。
// 注意事项
// Content-Type 需要指定为 application/json。
// 本接口返回的视频或图片链接均为临时链接，不应将其保存下来。
// 能不能获取播放链接取决于剧目审核状态，可能存在单个剧集的状态为审核通过，但是剧目整体是未通过的情况，这种情况也没法获取播放链接。
// 开发者如需区分不同渠道的播放流量或次数，可以在 us 参数中传入渠道标识，这样得到的播放链接中 us 参数的前半部分就包含有渠道标识。开发者把这个带有渠道标识的链接分发给对应的渠道播放，就能统计到不同渠道播放情况。统计的数据来源为 CDN 日志（从 getcdnlogs 接口得到），CDN 日志中“文件路径”列中的参数也带有该标识，再结合日志中“字节数”列的流量数值，估算每个渠道所消耗的流量。另需注意日志统计的流量和扣费流量的差异，详情参考 getcdnlogs 接口中的注意事项。
type GetMediaLinkRequest struct {
	MediaID int64  `json:"media_id"`         // 媒资文件 id。
	T       int64  `json:"t"`                // 播放地址的过期时间戳。有效的时间最长不能超过 2 小时后。
	US      string `json:"us,omitempty"`     // 链接标识。平台默认会生成一个仅包含小写字母和数字的字符串用于增强链接的唯一性 (如 us=647488c4792c15185b8fd2a6)。如开发者需要增加自己的标识，比如区分播放的渠道，可使用该参数，该参数最终的值是"开发者标识 - 平台标识"（如开发者传入 abcd，则最终的临时链接中 us=abcd-647488c4792c15185b8fd2a6）
	Expr    int    `json:"expr,omitempty"`   // 试看时长，单位：秒，最大值不能超过视频长度
	RLimit  int    `json:"rlimit,omitempty"` // 最多允许多少个不同 IP 的终端播放，以十进制表示，最大值为 9，不填表示不做限制。当限制 URL 只能被 1 个人播放时，建议 rlimit 不要严格限制成 1（例如可设置为 3），因为移动端断网后重连 IP 可能改变。
	WHref   string `json:"whref,omitempty"`  // 允许访问的域名列表，支持 1 条 - 10 条，用半角逗号分隔。域名前不要带协议名（http://和 https://），域名为前缀匹配（如填写 abc.com，则 abc.com/123 和 abc.com.cn 也会匹配），且支持通配符（如 *.abc.com）
	BkRef   string `json:"bkref,omitempty"`  // 禁止访问的域名列表，支持 1 条 - 10 条，用半角逗号分隔。域名前不要带协议名（http://和 https://），域名为前缀匹配（如填写 abc.com，则 abc.com/123 和 abc.com.cn 也会匹配），且支持通配符（如 *.abc.com）。
}

// GetMediaLinkResponse 获取媒体链接响应
type GetMediaLinkResponse struct {
	util.CommonError
	MediaInfo MediaPlaybackInfo `json:"media_info"` // 媒体播放信息。
}

// MediaPlaybackInfo 媒体播放信息
type MediaPlaybackInfo struct {
	MediaID     int64  `json:"media_id"`    // 媒资文件 id。
	Duration    int64  `json:"duration"`    // 播放时长，单位：秒。
	Name        string `json:"name"`        // 媒资文件名。
	Description string `json:"description"` // 描述。
	CoverURL    string `json:"cover_url"`   // 封面图临时链接。
	Mp4URL      string `json:"mp4_url"`     // mp4 格式临时链接。
	HlsURL      string `json:"hls_url"`     // hls 格式临时链接。
}

// DeleteMediaRequest 删除媒体请求
// 该接口用于删除已上传到平台的指定媒资文件，用于开发者后台管理使用。
// Content-Type 需要指定为 application/json。
type DeleteMediaRequest struct {
	MediaID int64 `json:"media_id"` // 媒资文件 id。
}

// DeleteMediaResponse 删除媒体响应
type DeleteMediaResponse struct {
	util.CommonError
}

// AuditDramaRequest 审核剧目请求
// 该接口用于审核剧目，审核通过后，剧目下所有剧集都会被审核通过。
// 注意事项
// Content-Type 需要指定为 application/json。
// 剧目信息与审核材料在首次提审时为必填，重新提审时根据是否需要修改选填，
// 本接口中使用的临时图片 material_id 可通过新增临时素材接口上传得到，对应临时素材接口中的 media_id，本文档中为避免与剧集的 media_id 混淆，称其为 material_id。
// 新增临时素材接口可以被小程序调用，调用的小程序账号和剧目提审的小程序账号必须是同一个，否则提交审核时会无法识别素材 id。
type AuditDramaRequest struct {
	DramaID                  int64          `json:"drama_id,omitempty"`                    // 剧目 ID，可通过查询剧目列表接口获取。首次提审不需要填该参数，重新提审时必填
	Name                     string         `json:"name,omitempty"`                        // 剧名，首次提审时必填，重新提审时根据是否需要修改选填。
	MediaCount               int            `json:"media_count,omitempty"`                 // 剧集数目。首次提审时必填，重新提审时可不填，如要填写也要和第一次提审时一样。
	MediaIDList              []int64        `json:"media_id_list,omitempty"`               // 剧集媒资 media_id 列表。首次提审时必填，而且元素个数必须与 media_count 一致。重新提审时为可选，如果剧集有内容有变化，可以通过新的列表替换未通过的剧集（推荐使用 replace_media_list 进行替换，避免顺序和原列表不一致）。
	Producer                 string         `json:"producer,omitempty"`                    // 制作方。首次提审时必填，重新提审时根据是否需要修改选填。
	Description              string         `json:"description,omitempty"`                 // 剧描述。首次提审时必填，重新提审时根据是否需要修改选填。
	CoverMaterialID          string         `json:"cover_material_id,omitempty"`           // 封面图片临时 media_id。首次提审时必填，重新提审时根据是否需要修改选填。
	RegistrationNumber       string         `json:"registration_number,omitempty"`         // 剧目备案号。首次提审时剧目备案号与网络剧片发行许可证编号二选一。重新提审时根据是否需要修改选填
	AuthorizedMaterialID     string         `json:"authorized_material_id,omitempty"`      // 剧目播放授权材料 material_id。如果小程序主体名称和制作方完全一致，则不需要填，否则必填
	PublishLicense           string         `json:"publish_license,omitempty"`             // 网络剧片发行许可证编号。首次提审时剧目备案号与网络剧片发行许可证编号二选一。重新提审时根据是否需要修改选填
	PublishLicenseMaterialID string         `json:"publish_license_material_id,omitempty"` // 网络剧片发行许可证图片，首次提审时如果网络剧片发行许可证编号非空，则该改字段也非空。重新提审时根据是否变化选填
	Expedited                int            `json:"expedited,omitempty"`                   // 是否加急审核，填 1 表示审核加急，0 或不填为不加急。每天有 5 次加急机会。该字段在首次提审时才有效，重新提审时会沿用首次提审时的属性，重新提审不会扣次数。最终是否为加急单，可以根据 DramaInfo.expedited 属性判断
	ReplaceMediaList         []*ReplaceInfo `json:"replace_media_list,omitempty"`          // 重新提审时，如果剧目内容有变化，可以通过该字段替换未通过的剧集。用于重新提审时替换审核不通过的剧集。
}

// ReplaceInfo 替换信息
type ReplaceInfo struct {
	Old int64 `json:"old"` // 旧的 media_id
	New int64 `json:"new"` // 新的 media_id
}

// AuditDramaResponse 审核剧目响应
type AuditDramaResponse struct {
	util.CommonError
	DramaID int64 `json:"drama_id"` // 剧目 ID。
}

// ListDramasRequest 查询剧目列表请求
// 该接口用于获取已提交的剧目列表。
// 注意事项
// Content-Type 需要指定为 application/json。
// 本接口返回的图片链接均为临时链接，不应将其保存下来。
// 如果剧目审核结果为失败或驳回，则具体每一集的具体驳回理由及证据截图可通过“获取媒资列表”或者“获取媒资详细信息”接口来获取。
type ListDramasRequest struct {
	Limit  int `json:"limit,omitempty"`  // 分页拉取的最大返回结果数。默认值：100；最大值：100。
	Offset int `json:"offset,omitempty"` // 分页拉取的起始偏移量。默认值：0。
}

// DramaInfo 剧目信息
type DramaInfo struct {
	DramaID           int64             `json:"drama_id"`           // 剧目 id。
	CreateTime        int64             `json:"create_time"`        // 创建时间，时间戳。
	Name              string            `json:"name"`               // 	剧名。
	Playwright        string            `json:"playwright"`         // 编剧。
	Producer          string            `json:"producer"`           // 制作方。
	ProductionLicense string            `json:"production_license"` // 广播电视节目制作经营许可证。
	CoverURL          string            `json:"cover_url"`          // 封面图临时链接，根据提审时提交的 cover_material_id 转存得到。
	MediaCount        int               `json:"media_count"`        // 剧集数目。
	Description       string            `json:"description"`        // 剧描述。
	MediaList         []*DramaMediaInfo `json:"media_list"`         // 剧集信息列表。
	AuditDetail       *DramaAuditDetail `json:"audit_detail"`       // 审核状态。
	Expedited         int               `json:"expedited"`          // 是否加急审核，1 表示审核加急，0 或空为非加急审核。
}

// DramaMediaInfo 剧目媒体信息
type DramaMediaInfo struct {
	MediaID int64 `json:"media_id"`
}

// DramaAuditDetail 剧目审核详情
type DramaAuditDetail struct {
	Status     int   `json:"status"`      // 审核状态 0 为无效值；1 为审核中；2 为审核驳回；3 为审核通过；4 为驳回重填。
	CreateTime int64 `json:"create_time"` // 提审时间戳。
	AuditTime  int64 `json:"audit_time"`  // 审核时间戳。
}

// ListDramasResponse 查询剧目列表响应
type ListDramasResponse struct {
	util.CommonError
	DramaInfoList []*DramaInfo `json:"drama_info_list"` // 剧目信息列表。
}

// GetDramaRequest 获取剧目请求
// 该接口用于查询已提交的剧目。
// 注意事项
// Content-Type 需要指定为 application/json。
// 本接口返回的图片链接均为临时链接，不应将其保存下来。
// 如果剧目审核结果为失败或驳回，则具体每一集的具体驳回理由及证据截图可通过“获取媒资列表”或者“获取媒资详细信息”接口来获取。
type GetDramaRequest struct {
	DramaID int64 `json:"drama_id"` // 剧目 id。
}

// GetDramaResponse 获取剧目响应
type GetDramaResponse struct {
	util.CommonError
	DramaInfo *DramaInfo `json:"drama_info"` // 剧目信息。
}

// GetCdnUsageDataRequest 获取 CDN 用量数据请求
// 该接口用于查询点播 CDN 的流量数据。
// 注意事项
// 可以查询最近 365 天内的 CDN 用量数据。
// 查询时间跨度不超过 90 天。
// 可以指定用量数据的时间粒度，支持 5 分钟、1 小时、1 天的时间粒度。
// 流量为查询时间粒度内的总流量。
type GetCdnUsageDataRequest struct {
	StartTime    int64 `json:"start_time"`    // 查询起始时间，Unix 时间戳，单位：秒。
	EndTime      int64 `json:"end_time"`      // 查询结束时间，Unix 时间戳，单位：秒。
	DataInterval int   `json:"data_interval"` // 用量数据的时间粒度，单位：分钟，取值有：5:5 分钟粒度，返回指定查询时间内 5 分钟粒度的明细数据。60：小时粒度，返回指定查询时间内 1 小时粒度的数据。1440：天粒度，返回指定查询时间内 1 天粒度的数据。默认值为 1440，返回天粒度的数据。
}

// DataItem 数据项
type DataItem struct {
	Time  int64 `json:"time"`  // 时间戳，单位：秒。
	Value int64 `json:"value"` // 用量数值。
}

// GetCdnUsageDataResponse 获取 CDN 用量数据响应
type GetCdnUsageDataResponse struct {
	util.CommonError
	DataInterval int         `json:"data_interval"`
	ItemList     []*DataItem `json:"item_list"`
}

// GetCdnLogsRequest 获取 CDN 日志下载链接请求
// 该接口用于获取点播 CDN 日志下载链接。
// 注意事项
// 可以查询最近 30 天内的 CDN 日志下载链接。
// 默认情况下 CDN 每小时生成一个日志文件，如果某一个小时没有 CDN 访问，不会生成日志文件。
// CDN 日志下载链接的有效期为 24 小时。
// 日志字段依次为：请求时间、客户端 IP、访问域名、文件路径、字节数、省级编码、运营商编码、HTTP 状态码、referer、Request-Time、UA、range、HTTP Method、协议标识、缓存 HIT / MISS，日志数据打包存在延迟，正常情况下 3 小时后数据包趋于完整日志中的字节数为应用层数据大小，未考虑网络协议包头、加速重传等开销，因此与计费数据存在一定差异。
// CDN 日志中记录的下行字节数统计而来的流量数据，是应用层数据。在实际网络传输中，产生的网络流量要比纯应用层流量多 5%-15%，比如 TCP/IP 协议的包头消耗、网络丢包重传等，这些无法被应用层统计到。在业内标准中，计费用流量一般在应用层流量的基础上加上上述开销，媒资管理服务中计费的加速流量约为日志计算加速流量的 110%。
// 省份映射
// 22：北京；86：内蒙古；146：山西；1069：河北；1177：天津；119：宁夏；152：陕西；1208：甘肃；1467：青海；1468：新疆；145：黑龙江；1445：吉林；1464：辽宁；2：福建；120：江苏；121：安徽；122：山东；1050：上海；1442：浙江；182：河南；1135：湖北；1465：江西；1466：湖南；118：贵州；153：云南；1051：重庆；1068：四川；1155：西藏；4：广东；173：广西；1441：海南；0：其他；1：港澳台；-1：海外。
// 运营商映射
// 2：中国电信；26：中国联通；38：教育网；43：长城宽带；1046：中国移动；3947：中国铁通；-1：海外运营商；0：其他运营商。
type GetCdnLogsRequest struct {
	StartTime int64 `json:"start_time"`       // 查询起始时间，Unix 时间戳，单位：秒。
	EndTime   int64 `json:"end_time"`         // 查询结束时间，Unix 时间戳，单位：秒。
	Limit     int   `json:"limit,omitempty"`  // 分页拉取的最大返回结果数。默认值：100；最大值：100。
	Offset    int   `json:"offset,omitempty"` // 分页拉取的起始偏移量。默认值：0。
}

// CdnLogInfo CDN 日志信息
type CdnLogInfo struct {
	Date      int64  `json:"date"`       // 日志日期，格式为 YYYYMMDD。
	Name      string `json:"name"`       // 日志文件名
	URL       string `json:"url"`        // 日志下载链接，24 小时内下载有效。
	StartTime int64  `json:"start_time"` // 查询起始时间，Unix 时间戳，单位：秒。
	EndTime   int64  `json:"end_time"`   // 查询结束时间，Unix 时间戳，单位：秒。
}

// GetCdnLogsResponse 获取 CDN 日志下载链接响应
type GetCdnLogsResponse struct {
	util.CommonError
	TotalCount      int           `json:"total_count"`       // 日志总条数。
	DomesticCdnLogs []*CdnLogInfo `json:"domestic_cdn_logs"` // 日志信息列表，国内 CDN 节点的日志下载列表。
}

// AsyncMediaUploadEvent 异步媒体上传事件
// see: https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/mini-drama/mini_drama.html#_5-1-%E5%AA%92%E8%B5%84%E4%B8%8A%E4%BC%A0%E5%AE%8C%E6%88%90%E4%BA%8B%E4%BB%B6
type AsyncMediaUploadEvent struct {
	util.CommonError
	MediaID       int64  `json:"media_id"`       // 媒资文件 id。
	SourceContext string `json:"source_context"` // 来源上下文，开发者可自定义该字段内容。
}

// AsyncMediaAuditEvent 异步媒体审核事件
// see: https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/mini-drama/mini_drama.html#_5-2-%E5%AE%A1%E6%A0%B8%E7%8A%B6%E6%80%81%E4%BA%8B%E4%BB%B6
type AsyncMediaAuditEvent struct {
	DramaID       int64             `json:"drama_id"`       // 剧目 id。
	SourceContext string            `json:"source_context"` // 来源上下文，开发者可自定义该字段内容。
	AuditDetail   *DramaAuditDetail `json:"audit_detail"`   // 审核状态。
}
