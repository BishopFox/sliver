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

const (
	// Success 错误码 0、成功
	Success ErrCode = 0
	// SystemError 错误码 -1、系统错误
	SystemError ErrCode = -1
	// InitError 错误码 -2 初始化未完成，请稍后再试
	InitError ErrCode = -2
	// FormatError 错误码 47001	输入格式错误
	FormatError ErrCode = 47001
	// ParamError 错误码 47003	参数不符合要求
	ParamError ErrCode = 47003
	// PostError 错误码 44002	POST 内容为空
	PostError ErrCode = 44002
	// MethodError 错误码 43002	HTTP 请求必须使用 POST 方法
	MethodError ErrCode = 43002
	// VideoTypeError 错误码 10090001	视频类型不支持
	VideoTypeError ErrCode = 10090001
	// ImageTypeError 错误码 10090002	图片类型不支持
	ImageTypeError ErrCode = 10090002
	// ImageURLError 错误码 10090003	图片 URL 无效
	ImageURLError ErrCode = 10090003
	// ResourceType 错误码 10090005	resource_type 无效
	ResourceType ErrCode = 10090005
	// OperationError 错误码 10093011	操作失败
	OperationError ErrCode = 10093011
	// ParamError2 错误码 10093014	参数错误（包括参数格式、类型等错误）
	ParamError2 ErrCode = 10093014
	// OperationFrequentError 错误码 10093023	操作过于频繁
	OperationFrequentError ErrCode = 10093023
	// ResourceNotExistError 错误码 10093030	资源不存在
	ResourceNotExistError ErrCode = 10093030
)

const (
	// singleFileUpload 单个文件上传，上传媒体（和封面）文件，上传小文件（小于 10MB）时使用。上传大文件请使用分片上传接口。
	singleFileUpload = "https://api.weixin.qq.com/wxa/sec/vod/singlefileupload?access_token="

	// pullUpload 拉取上传，该接口用于将一个网络上的视频拉取上传到平台。
	pullUpload = "https://api.weixin.qq.com/wxa/sec/vod/pullupload?access_token="

	// getTask 查询任务，该接口用于查询拉取上传的任务状态。
	getTask = "https://api.weixin.qq.com/wxa/sec/vod/gettask?access_token="

	// applyUpload 申请分片上传
	applyUpload = "https://api.weixin.qq.com/wxa/sec/vod/applyupload?access_token="

	// uploadPart 上传分片
	uploadPart = "https://api.weixin.qq.com/wxa/sec/vod/uploadpart?access_token="

	// commitUpload 确认上传，该接口用于完成整个分片上传流程，合并所有文件分片，确认媒体文件（和封面图片文件）上传到平台的结果，返回文件的 ID。请求中需要给出每一个分片的 part_number 和 etag，用来校验分片的准确性。
	commitUpload = "https://api.weixin.qq.com/wxa/sec/vod/commitupload?access_token="

	// listMedia 获取媒体列表
	listMedia = "https://api.weixin.qq.com/wxa/sec/vod/listmedia?access_token="

	// getMedia 获取媒资详细信息，该接口用于获取已上传到平台的指定媒资信息，用于开发者后台管理使用。用于给用户客户端播放的链接应该使用 getmedialink 接口获取。
	getMedia = "https://api.weixin.qq.com/wxa/sec/vod/getmedia?access_token="

	// getMediaLink 获取媒资播放链接，该接口用于获取视频临时播放链接，用于给用户的播放使用。只有审核通过的视频才能通过该接口获取播放链接。
	getMediaLink = "https://api.weixin.qq.com/wxa/sec/vod/getmedialink?access_token="

	// deleteMedia 删除媒体，该接口用于删除指定媒资。
	deleteMedia = "https://api.weixin.qq.com/wxa/sec/vod/deletemedia?access_token="

	// auditDrama 审核剧本
	auditDrama = "https://api.weixin.qq.com/wxa/sec/vod/auditdrama?access_token="

	// listDramas 获取剧目列表
	listDramas = "https://api.weixin.qq.com/wxa/sec/vod/listdramas?access_token="

	// getDrama 获取剧目信息，该接口用于查询已提交的剧目。
	getDrama = "https://api.weixin.qq.com/wxa/sec/vod/getdrama?access_token="

	// getCdnUsageData 查询 CDN 用量数据，该接口用于查询点播 CDN 的流量数据。
	getCdnUsageData = "https://api.weixin.qq.com/wxa/sec/vod/getcdnusagedata?access_token="

	// getCdnLogs 查询 CDN 日志，该接口用于查询点播 CDN 的日志。
	getCdnLogs = "https://api.weixin.qq.com/wxa/sec/vod/getcdnlogs?access_token="
)
