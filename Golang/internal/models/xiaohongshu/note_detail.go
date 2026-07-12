// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/xiaohongshu/note_detail.go
// GitHub: https://github.com/NanmiCoder
// Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
//
// 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
// 1. 不得用于任何商业用途。
// 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
// 3. 不得进行大规模爬取或对平台造成运营干扰。
// 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
// 5. 不得用于任何非法或不当的用途。
//
// 详细许可条款请参阅项目根目录下的LICENSE文件。
// 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

package xiaohongshu

// Note 主结构体
type NoteDetail struct {
	XsecToken      string       `json:"xsec_token"`
	XsecSource     string       `json:"xsec_source"`
	TagList        []TagInfo    `json:"tag_list"`
	AtUserList     []User       `json:"at_user_list"`
	IpLocation     string       `json:"ip_location"`
	ShareInfo      ShareInfo    `json:"share_info"`
	Title          string       `json:"title"`
	ImageList      []Image      `json:"image_list"`
	LastUpdateTime int64        `json:"last_update_time"`
	Desc           string       `json:"desc"`
	Time           int64        `json:"time"`
	NoteId         string       `json:"note_id"`
	Video          Video        `json:"video"`
	InteractInfo   InteractInfo `json:"interact_info"`
	Type           string       `json:"type"`
	User           User         `json:"user"`
}

// ShareInfo 分享信息
type ShareInfo struct {
	UnShare bool `json:"un_share"`
}

// Image 图片信息
type Image struct {
	TraceId    string      `json:"trace_id"`
	Width      int         `json:"width"`
	URL        string      `json:"url"`
	InfoList   []ImageInfo `json:"info_list"`
	URLPre     string      `json:"url_pre"`
	URLDefault string      `json:"url_default"`
	Stream     struct{}    `json:"stream"`
	LivePhoto  bool        `json:"live_photo"`
	FileId     string      `json:"file_id"`
	Height     int         `json:"height"`
}

// Video 视频信息
type Video struct {
	Media    MediaInfo     `json:"media"`
	Image    VideoImage    `json:"image"`
	Capa     VideoCapa     `json:"capa"`
	Consumer VideoConsumer `json:"consumer"`
}

// MediaInfo 媒体信息
type MediaInfo struct {
	VideoId int64       `json:"video_id"`
	Video   VideoDetail `json:"video"`
	Stream  StreamInfo  `json:"stream"`
}

// VideoDetail 视频详细信息
type VideoDetail struct {
	BizId       string `json:"biz_id"`
	Duration    int    `json:"duration"`
	Md5         string `json:"md5"`
	HdrType     int    `json:"hdr_type"`
	DrmType     int    `json:"drm_type"`
	StreamTypes []int  `json:"stream_types"`
	BizName     int    `json:"biz_name"`
}

// StreamInfo 流信息
type StreamInfo struct {
	H265 []StreamDetail `json:"h265"`
	H266 []StreamDetail `json:"h266"`
	AV1  []StreamDetail `json:"av1"`
	H264 []StreamDetail `json:"h264"`
}

// StreamDetail 流详细信息
type StreamDetail struct {
	Duration      int64    `json:"duration"`
	VideoBitrate  int64    `json:"video_bitrate"`
	AudioBitrate  int64    `json:"audio_bitrate"`
	Vmaf          int      `json:"vmaf"`
	Volume        int      `json:"volume"`
	AvgBitrate    int64    `json:"avg_bitrate"`
	AudioCodec    string   `json:"audio_codec"`
	Ssim          int      `json:"ssim"`
	QualityType   string   `json:"quality_type"`
	DefaultStream int      `json:"default_stream"`
	Format        string   `json:"format"`
	Size          int64    `json:"size"`
	AudioDuration int64    `json:"audio_duration"`
	MasterUrl     string   `json:"master_url"`
	AudioChannels int      `json:"audio_channels"`
	StreamType    int      `json:"stream_type"`
	Fps           int      `json:"fps"`
	VideoCodec    string   `json:"video_codec"`
	Rotate        int      `json:"rotate"`
	Psnr          float64  `json:"psnr"`
	StreamDesc    string   `json:"stream_desc"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	VideoDuration int64    `json:"video_duration"`
	BackupUrls    []string `json:"backup_urls"`
	HdrType       int      `json:"hdr_type"`
	Weight        int      `json:"weight"`
}

// VideoImage 视频图片信息
type VideoImage struct {
	ThumbnailFileid string `json:"thumbnail_fileid"`
}

// VideoCapa 视频能力信息
type VideoCapa struct {
	Duration int `json:"duration"`
}

// VideoConsumer 视频消费者信息
type VideoConsumer struct {
	OriginVideoKey string `json:"origin_video_key"`
}

// NoteDetailResponseFromHtml 从html中获取的笔记详情
type NoteDetailResponseFromHtml struct {
	Note struct {
		NoteDetailMap struct {
			Data map[string]struct {
				Note NoteDetail `json:"note"`
			} `json:",omitempty"`
		} `json:"note_detail_map"`
	} `json:"note"`
}

// NoteDetailApiFeedRequest 从api中获取的笔记详情请求
type NoteDetailApiFeedRequest struct {
	SourceNoteID string   `json:"source_note_id"`
	ImageFormats []string `json:"image_formats"`
	Extra        struct {
		NeedBodyTopic int `json:"need_body_topic"`
	} `json:"extra"`
	XsecToken  string `json:"xsec_token"`
	XsecSource string `json:"xsec_source"`
}

// NoteCardFromApi 从api中获取的笔记卡片
type NoteCardFromApi struct {
	ID        string     `json:"id"`
	ModelType string     `json:"model_type"`
	NoteCard  NoteDetail `json:"note_card"`
}

// NoteDetailResponseFromApi 从api中获取的笔记详情
type NoteDetailResponseFromApi struct {
	CurrentTime int64             `json:"current_time"`
	CursorScore string            `json:"cursor_score"`
	Items       []NoteCardFromApi `json:"items"`
}
