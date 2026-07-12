// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/config/config.go
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

package config

import (
	"fmt"
	"log"
	"strconv"

	"github.com/spf13/viper"
)

var (
	Obj *Config = nil
)

func InitConfig() {
	viper.SetConfigName("config")     // 配置文件名称(无扩展名)
	viper.SetConfigType("yml")        // 如果配置文件的名称中没有扩展名，则需要配置此项
	viper.AddConfigPath("config/")    // 查找配置文件所在的路径
	viper.AddConfigPath("../config/") // 测试目录下查找上层config目录
	viper.AddConfigPath(".")          // 当前目录
	err := viper.ReadInConfig()       // 查找并读取配置文件
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	err = viper.Unmarshal(&Obj)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
	Obj.ApplyDefaults()

	// 将环境变量中的敏感信息覆盖到配置中
	if Obj.DbConfig == nil {
		Obj.DbConfig = &DbConfig{}
	}

	// 设置关系型数据库配置（从环境变量）
	Obj.DbConfig.RelationDbPwd = RelationDbPwd
	Obj.DbConfig.RelationDbUser = RelationDbUser
	Obj.DbConfig.RelationDbHost = RelationDbHost
	Obj.DbConfig.RelationDbName = RelationDbName

	// 将端口字符串转换为整数
	if port, err := strconv.Atoi(RelationDbPort); err == nil {
		Obj.DbConfig.RelationDbPort = port
	} else {
		log.Printf("Error converting port to integer: %v", err)
		// 可以设置一个默认端口
		Obj.DbConfig.RelationDbPort = 3306
	}

	// 设置Redis配置（从环境变量）
	Obj.DbConfig.RedisDbHost = RedisDbHost
	Obj.DbConfig.RedisDbPwd = RedisDbPwd

	// 转换Redis端口
	if redisPort, err := strconv.Atoi(RedisDbPort); err == nil {
		Obj.DbConfig.RedisDbPort = redisPort
	} else {
		log.Printf("Error converting Redis port to integer: %v", err)
		Obj.DbConfig.RedisDbPort = 6379
	}

	// 转换Redis数据库编号
	if redisDbNum, err := strconv.ParseUint(RedisDbNum, 10, 32); err == nil {
		Obj.DbConfig.RedisDbNum = uint(redisDbNum)
	} else {
		log.Printf("Error converting Redis DB number to uint: %v", err)
		Obj.DbConfig.RedisDbNum = 0
	}

	// 打印读取的配置信息
	// fmt.Printf("Config: %+v\n", Obj)
	//fmt.Printf("Database: %+v\n", Obj.DbConfig)
	//fmt.Printf("Proxy: %+v\n", Obj.ProxyConfig)
	//fmt.Printf("SignSrv: %+v\n", Obj.SignSrvConfig)
}

func GetConfig() *Config {
	if Obj == nil {
		InitConfig()
	}
	return Obj
}

// Config 配置接口体
type Config struct {
	Platform                         string                   `yaml:"platform"`                         // 爬取的平台名
	Keywords                         []string                 `yaml:"keywords"`                         // 爬取的关键词列表
	CrawlerType                      string                   `yaml:"crawlerType"`                      // 爬取的类型
	SaveDataOption                   string                   `yaml:"saveDataOption"`                   // 爬取数据爬虫的类型
	AccountPoolSaveType              string                   `yaml:"accountPoolSaveType"`              // 账号池保存类型
	CookieBridgeURL                  string                   `yaml:"cookieBridgeURL"`                  // CookieBridge服务地址
	StartPage                        int                      `yaml:"startPage"`                        // 开始的页码
	CrawlerMaxNotesCount             int                      `yaml:"crawlerMaxNotesCount"`             // 爬取的最大帖子数量
	MaxConcurrencyNum                int                      `yaml:"maxConcurrencyNum"`                // 并发数量
	RequestTimeOut                   int                      `yaml:"requestTimeOut"`                   // 请求超时时间,单位秒
	EnableGetComments                bool                     `yaml:"enableGetComments"`                // 是否开启获取评论模式
	EnableGetSubComments             bool                     `yaml:"enableGetSubComments"`             // 是否开启获取子评论模式
	EnableCheckpoint                 bool                     `yaml:"enableCheckpoint"`                 // 是否开启断点续爬
	SpecifiedCheckpointID            string                   `yaml:"specifiedCheckpointId"`            // 指定断点续爬检查点ID
	HotSearchMaxItemsCount           int                      `yaml:"hotSearchMaxItemsCount"`           // hot_search模式最多处理多少榜单项
	HotSearchPerItemContentCount     int                      `yaml:"hotSearchPerItemContentCount"`     // hot_search模式每个榜单项最多抓取多少内容
	DataDir                          string                   `yaml:"dataDir"`                          // 文件存储根目录
	SqliteDBPath                     string                   `yaml:"sqliteDbPath"`                     // SQLite数据库文件路径
	DebugRequest                     bool                     `yaml:"debugRequest"`                     // 是否开启调试请求
	CrawlerTimeSleep                 int                      `yaml:"crawlerTimeSleep"`                 // 爬取请求间隔时间,单位秒
	PerNoteMaxComments               int                      `yaml:"perNoteMaxComments"`               // 每个帖子最大评论数量
	SpecifiedNoteIDs                 []string                 `yaml:"specifiedNoteIds"`                 // 指定爬取的帖子ID列表(已废弃,请使用specifiedNoteURLs)
	SpecifiedNoteURLs                []string                 `yaml:"specifiedNoteURLs"`                // 指定爬取的帖子URL列表(携带xsec_token)
	SpecifiedCreatorIDs              []string                 `yaml:"specifiedCreatorIds"`              // 指定爬取的创作者ID列表(已废弃,请使用specifiedCreatorURLs)
	SpecifiedCreatorURLs             []string                 `yaml:"specifiedCreatorURLs"`             // 指定爬取的创作者主页URL列表(携带xsec_token)
	HotSearchCheckpointMaxAgeSeconds int                      `yaml:"hotSearchCheckpointMaxAgeSeconds"` // hot_search断点快照最大可恢复秒数
	PlatformEndpoints                *PlatformEndpointsConfig `yaml:"platformEndpoints"`                // 平台API地址，可被测试mock覆盖
	DbConfig                         *DbConfig                `yaml:"dbConfig"`                         // DB相关的配置信息
	ProxyConfig                      *ProxyConfig             `yaml:"proxyConfig"`                      // IP代理相关的配置信息
	SignSrvConfig                    *SignSrvConfig           `yaml:"signSrvConfig"`                    // 请求签名相关的配置信息
}

type PlatformEndpointsConfig struct {
	XhsApiURL         string `yaml:"xhsApiUrl"`
	XhsIndexURL       string `yaml:"xhsIndexUrl"`
	BilibiliApiURL    string `yaml:"bilibiliApiUrl"`
	BilibiliIndexURL  string `yaml:"bilibiliIndexUrl"`
	BilibiliSpaceURL  string `yaml:"bilibiliSpaceUrl"`
	DouyinApiURL      string `yaml:"douyinApiUrl"`
	DouyinIndexURL    string `yaml:"douyinIndexUrl"`
	KuaishouApiURL    string `yaml:"kuaishouApiUrl"`
	KuaishouRestURL   string `yaml:"kuaishouRestApiUrl"`
	WeiboApiURL       string `yaml:"weiboApiUrl"`
	WeiboHotSearchURL string `yaml:"weiboHotSearchUrl"`
	WeiboHotBandURL   string `yaml:"weiboHotBandUrl"`
	TiebaURL          string `yaml:"tiebaUrl"`
	ZhihuURL          string `yaml:"zhihuUrl"`
	ZhihuZhuanlanURL  string `yaml:"zhihuZhuanlanUrl"`
}

func (c *Config) ApplyDefaults() {
	if c.PlatformEndpoints == nil {
		c.PlatformEndpoints = &PlatformEndpointsConfig{}
	}
	c.PlatformEndpoints.XhsApiURL = envOrConfigured("MCPRO_XHS_API_URL", c.PlatformEndpoints.XhsApiURL, "https://edith.xiaohongshu.com")
	c.PlatformEndpoints.XhsIndexURL = envOrConfigured("MCPRO_XHS_INDEX_URL", c.PlatformEndpoints.XhsIndexURL, "https://www.xiaohongshu.com")
	c.PlatformEndpoints.BilibiliApiURL = envOrConfigured("MCPRO_BILIBILI_API_URL", c.PlatformEndpoints.BilibiliApiURL, "https://api.bilibili.com")
	c.PlatformEndpoints.BilibiliIndexURL = envOrConfigured("MCPRO_BILIBILI_INDEX_URL", c.PlatformEndpoints.BilibiliIndexURL, "https://www.bilibili.com")
	c.PlatformEndpoints.BilibiliSpaceURL = envOrConfigured("MCPRO_BILIBILI_SPACE_URL", c.PlatformEndpoints.BilibiliSpaceURL, "https://space.bilibili.com")
	c.PlatformEndpoints.DouyinApiURL = envOrConfigured("MCPRO_DOUYIN_API_URL", c.PlatformEndpoints.DouyinApiURL, "https://www.douyin.com")
	c.PlatformEndpoints.DouyinIndexURL = envOrConfigured("MCPRO_DOUYIN_INDEX_URL", c.PlatformEndpoints.DouyinIndexURL, "https://www.douyin.com")
	c.PlatformEndpoints.KuaishouApiURL = envOrConfigured("MCPRO_KUAISHOU_API_URL", c.PlatformEndpoints.KuaishouApiURL, "https://www.kuaishou.com/graphql")
	c.PlatformEndpoints.KuaishouRestURL = envOrConfigured("MCPRO_KUAISHOU_REST_API_URL", c.PlatformEndpoints.KuaishouRestURL, "https://www.kuaishou.com")
	c.PlatformEndpoints.WeiboApiURL = envOrConfigured("MCPRO_WEIBO_API_URL", c.PlatformEndpoints.WeiboApiURL, "https://m.weibo.cn")
	c.PlatformEndpoints.WeiboHotSearchURL = envOrConfigured("MCPRO_WEIBO_HOT_SEARCH_URL", c.PlatformEndpoints.WeiboHotSearchURL, "https://weibo.com/ajax/side/hotSearch")
	c.PlatformEndpoints.WeiboHotBandURL = envOrConfigured("MCPRO_WEIBO_HOT_BAND_URL", c.PlatformEndpoints.WeiboHotBandURL, "https://weibo.com/ajax/statuses/hot_band")
	c.PlatformEndpoints.TiebaURL = envOrConfigured("MCPRO_TIEBA_URL", c.PlatformEndpoints.TiebaURL, "https://tieba.baidu.com")
	c.PlatformEndpoints.ZhihuURL = envOrConfigured("MCPRO_ZHIHU_URL", c.PlatformEndpoints.ZhihuURL, "https://www.zhihu.com")
	c.PlatformEndpoints.ZhihuZhuanlanURL = envOrConfigured("MCPRO_ZHIHU_ZHUANLAN_URL", c.PlatformEndpoints.ZhihuZhuanlanURL, "https://zhuanlan.zhihu.com")

	if c.HotSearchMaxItemsCount == 0 {
		c.HotSearchMaxItemsCount = 50
	}
	if c.HotSearchPerItemContentCount == 0 {
		c.HotSearchPerItemContentCount = 4
	}
	if value := GetEnvWrapper("HOT_SEARCH_CHECKPOINT_MAX_AGE_SECONDS", ""); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			c.HotSearchCheckpointMaxAgeSeconds = parsed
		}
	} else if c.HotSearchCheckpointMaxAgeSeconds == 0 {
		c.HotSearchCheckpointMaxAgeSeconds = 6 * 60 * 60
	}
	c.DataDir = envOrConfigured("MCPRO_DATA_DIR", c.DataDir, "data")
	c.CookieBridgeURL = envOrConfigured("COOKIE_BRIDGE_URL", c.CookieBridgeURL, "http://localhost:8274")
	if c.SqliteDBPath == "" {
		c.SqliteDBPath = "media_crawler.db"
	}
	if value := GetEnvWrapper("SQLITE_DB_PATH", ""); value != "" {
		c.SqliteDBPath = value
	}
	if value := GetEnvWrapper("MCPRO_SQLITE_DB_PATH", ""); value != "" {
		c.SqliteDBPath = value
	}
}

func envOrConfigured(envKey, configured, defaultValue string) string {
	if value := GetEnvWrapper(envKey, ""); value != "" {
		return value
	}
	if configured != "" {
		return configured
	}
	return defaultValue
}

// DbConfig db配置结构体
type DbConfig struct {
	RelationDbHost string `yaml:"relationDbHost"` // 关系型数据库Host
	RelationDbName string `yaml:"relationDbName"` // 关系型数据库Db name
	RelationDbUser string `yaml:"relationDbUser"` // 关系型数据库DB user
	RelationDbPwd  string `yaml:"relationDbPwd"`  // 关系性数据库DB password
	RelationDbPort int    `yaml:"relationDbPort"` // 关系型数据库Db port
	RedisDbHost    string `yaml:"redisDbHost"`    // redis数据库Host
	RedisDbPwd     string `yaml:"redisDbPwd"`     // redis数据库密码
	RedisDbPort    int    `yaml:"redisDbPort"`    // redis数据库port
	RedisDbNum     uint   `yaml:"redisDbNum"`     // redis数据库Db num
}

// ProxyConfig 代理配置结构体
type ProxyConfig struct {
	EnableIpProxy         bool   `yaml:"enableIpProxy"`         // 是否开启IP代理
	EnableValidateIpProxy bool   `yaml:"enableValidateIpProxy"` // 是否开启IP代理验证
	IpProxyPoolCount      int    `yaml:"ipProxyPoolCount"`      // 代理池数量
	IpProxyProvideName    string `yaml:"ipProxyProvideName"`    // IP代理提供商的名称
	KdlSecretId           string `yaml:"kdlSecretId"`           //快代理的secret id
	KdlSignature          string `yaml:"kdlSignature"`          // 快代理的签名
	KdlUserName           string `yaml:"kdlUserName"`           // 快代理的订单用户名
	KdlUserPwd            string `yaml:"kdlUserPwd"`            // 快代理订单的用户名密码
}

// SignSrvConfig 请求签名配置结构体
type SignSrvConfig struct {
	SignSrvHost string `yaml:"signSrvHost"` // 签名服务Host
	SignSrvPort int    `yaml:"signSrvPort"` // 签名服务端口
}
