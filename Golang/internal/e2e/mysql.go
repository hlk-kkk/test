// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/e2e/mysql.go
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
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

type MySQLOptions struct {
	Database string
	User     string
	Password string
}

type MySQLServer struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	WorkDir  string
	DataDir  string

	mysqladminPath string
	cmd            *exec.Cmd
	done           chan error
}

func StartMySQL(t testing.TB, opts MySQLOptions) *MySQLServer {
	t.Helper()

	mysqldPath := executableOrSkip(t, "mysqld")
	mysqlPath := executableOrSkip(t, "mysql")
	mysqladminPath := executableOrSkip(t, "mysqladmin")
	if opts.Database == "" {
		opts.Database = "media_crawler_pro"
	}
	if opts.User == "" {
		opts.User = "mcpro"
	}
	if opts.Password == "" {
		opts.Password = "mcpro_test"
	}
	validateMySQLIdentifier(t, opts.Database, "database")
	validateMySQLIdentifier(t, opts.User, "user")
	validateMySQLIdentifier(t, opts.Password, "password")

	workDir, err := os.MkdirTemp("/tmp", "mediacrawlerpro-go-mysql-*")
	if err != nil {
		t.Fatalf("create mysql work dir: %v", err)
	}
	dataDir := filepath.Join(workDir, "data")
	initLogPath := filepath.Join(workDir, "init.err")
	serverLogPath := filepath.Join(workDir, "mysql.err")

	initCmd := exec.Command(
		mysqldPath,
		"--initialize-insecure",
		"--datadir="+dataDir,
		"--log-error="+initLogPath,
	)
	if output, err := initCmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(workDir)
		t.Fatalf("initialize mysql datadir: %v\n%s\n%s", err, output, readFileBestEffort(initLogPath))
	}

	server := &MySQLServer{
		Host:           "127.0.0.1",
		Port:           freeTCPPort(t),
		User:           opts.User,
		Password:       opts.Password,
		Database:       opts.Database,
		WorkDir:        workDir,
		DataDir:        dataDir,
		mysqladminPath: mysqladminPath,
	}
	server.cmd = exec.Command(
		mysqldPath,
		"--datadir="+dataDir,
		"--socket="+filepath.Join(workDir, "mysql.sock"),
		"--port="+server.Port,
		"--bind-address="+server.Host,
		"--skip-networking=0",
		"--mysqlx=0",
		"--pid-file="+filepath.Join(workDir, "mysql.pid"),
		"--log-error="+serverLogPath,
	)
	if err := server.cmd.Start(); err != nil {
		_ = os.RemoveAll(workDir)
		t.Fatalf("start mysql: %v\n%s", err, readFileBestEffort(serverLogPath))
	}
	server.done = make(chan error, 1)
	go func() {
		server.done <- server.cmd.Wait()
	}()
	t.Cleanup(server.Close)

	server.waitUntilReady(t, serverLogPath)
	server.bootstrap(t, mysqlPath, serverLogPath)
	return server
}

func (s *MySQLServer) Env() []string {
	return []string{
		"RELATION_DB_HOST=" + s.Host,
		"RELATION_DB_PORT=" + s.Port,
		"RELATION_DB_USER=" + s.User,
		"RELATION_DB_PWD=" + s.Password,
		"RELATION_DB_NAME=" + s.Database,
	}
}

func (s *MySQLServer) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		s.User,
		s.Password,
		s.Host,
		s.Port,
		s.Database,
	)
}

func (s *MySQLServer) Close() {
	if s == nil {
		return
	}
	if s.cmd != nil && s.cmd.Process != nil {
		shutdown := exec.Command(
			s.mysqladminPath,
			"-uroot",
			"-h"+s.Host,
			"-P"+s.Port,
			"--protocol=tcp",
			"shutdown",
		)
		_ = shutdown.Run()

		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			_ = s.cmd.Process.Kill()
			<-s.done
		}
		s.cmd = nil
	}
	if s.WorkDir != "" {
		_ = os.RemoveAll(s.WorkDir)
		s.WorkDir = ""
	}
}

func (s *MySQLServer) waitUntilReady(t testing.TB, logPath string) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-s.done:
			t.Fatalf("mysql exited before ready: %v\n%s", err, readFileBestEffort(logPath))
		default:
		}

		pingCmd := exec.Command(
			s.mysqladminPath,
			"ping",
			"-uroot",
			"-h"+s.Host,
			"-P"+s.Port,
			"--protocol=tcp",
			"--silent",
		)
		if err := pingCmd.Run(); err == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("mysql did not become ready\n%s", readFileBestEffort(logPath))
}

func (s *MySQLServer) bootstrap(t testing.TB, mysqlPath, logPath string) {
	t.Helper()

	statement := fmt.Sprintf(
		"CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER '%s'@'%%' IDENTIFIED BY '%s'; GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%'; FLUSH PRIVILEGES;",
		quoteMySQLIdentifier(s.Database),
		s.User,
		s.Password,
		quoteMySQLIdentifier(s.Database),
		s.User,
	)
	bootstrapCmd := exec.Command(
		mysqlPath,
		"-uroot",
		"-h"+s.Host,
		"-P"+s.Port,
		"--protocol=tcp",
		"-e",
		statement,
	)
	if output, err := bootstrapCmd.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap mysql database: %v\n%s\n%s", err, output, readFileBestEffort(logPath))
	}
}

func executableOrSkip(t testing.TB, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s is not available", name)
	}
	return path
}

func freeTCPPort(t testing.TB) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate mysql tcp port: %v", err)
	}
	defer listener.Close()
	return fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)
}

var mysqlSafeNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func validateMySQLIdentifier(t testing.TB, value, label string) {
	t.Helper()
	if !mysqlSafeNamePattern.MatchString(value) {
		t.Fatalf("unsafe mysql %s: %q", label, value)
	}
}

func quoteMySQLIdentifier(identifier string) string {
	return "`" + identifier + "`"
}

func readFileBestEffort(path string) string {
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(body)
}
