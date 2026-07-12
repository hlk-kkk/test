// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/e2e/mysql_test.go
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

package e2e

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestStartMySQLCreatesReachableDatabaseUnderTmp(t *testing.T) {
	server := StartMySQL(t, MySQLOptions{Database: "media_crawler_pro"})

	require.True(t, strings.HasPrefix(server.WorkDir, "/tmp/"), server.WorkDir)
	require.Equal(t, []string{
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=" + server.Port,
		"RELATION_DB_USER=mcpro",
		"RELATION_DB_PWD=mcpro_test",
		"RELATION_DB_NAME=media_crawler_pro",
	}, server.Env())

	db, err := sql.Open("mysql", server.DSN())
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Ping())

	_, err = db.Exec("CREATE TABLE mysql_probe (id INT PRIMARY KEY, name VARCHAR(20))")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO mysql_probe (id, name) VALUES (1, 'ok')")
	require.NoError(t, err)

	var name string
	require.NoError(t, db.QueryRow("SELECT name FROM mysql_probe WHERE id = 1").Scan(&name))
	require.Equal(t, "ok", name)
}
