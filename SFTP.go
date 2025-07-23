package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/time/rate"
)

// SFTPConfig 定义SFTP服务器配置
type SFTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Path     string
}

// UploadConfig 上传配置
type UploadConfig struct {
	Servers       []SFTPConfig
	MaxRetries    int
	RetryDelay    time.Duration
	RateLimitKBps int
	Timeout       time.Duration
}

// SFTPClient 封装SFTP客户端
type SFTPClient struct {
	client *sftp.Client
	config SFTPConfig
}

// NewSFTPClient 创建SFTP客户端
func NewSFTPClient(config SFTPConfig) (*SFTPClient, error) {
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create sftp client: %v", err)
	}

	return &SFTPClient{client: client, config: config}, nil
}

// Close 关闭SFTP客户端
func (c *SFTPClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// UploadFileWithRetry 上传文件，支持重试和限速
func (c *SFTPClient) UploadFileWithRetry(filePath, remotePath string, config UploadConfig) error {
	limiter := rate.NewLimiter(rate.Limit(config.RateLimitKBps*1024), config.RateLimitKBps*1024)

	filename := filepath.Base(filePath)
	tmpRemotePath := fmt.Sprintf("%s/%s.tmp", remotePath, filename)
	okRemotePath := fmt.Sprintf("%s/%s", remotePath, filename)

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := c.uploadFile(filePath, tmpRemotePath, limiter, config.Timeout)
		if err == nil {
			// 上传成功，重命名文件
			err = c.client.Rename(tmpRemotePath, okRemotePath)
			if err == nil {
				return nil
			}
			log.Printf("[Upload Error] Failed to rename %s to %s: %v", tmpRemotePath, remotePath, err)
		}
		if attempt < config.MaxRetries {
			log.Printf("[Upload Error] Upload attempt %d failed for %s: %v. Retrying...", attempt+1, filename, err)
			time.Sleep(config.RetryDelay)
		}
	}
	return fmt.Errorf("[Upload Error] failed to upload %s after %d retries", filename, config.MaxRetries)
}

// uploadFile 执行文件上传
func (c *SFTPClient) uploadFile(localPath, remotePath string, limiter *rate.Limiter, timeout time.Duration) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("[Upload Error] failed to open local file: %v", err)
	}
	defer localFile.Close()

	remoteFile, err := c.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("[Upload Error] failed to create remote file: %v", err)
	}
	defer remoteFile.Close()

	// 检查 limiter 是否为 nil
	if limiter == nil {
		return fmt.Errorf("[Upload Error] rate limiter is nil")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 设置超时
	done := make(chan error, 1)
	go func() {
		// 使用限速器进行上传
		reader := &rateLimitedReader{Reader: localFile, Limiter: limiter, Ctx: ctx}
		_, err := io.Copy(remoteFile, reader)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("[Upload Error] failed to upload file: %v", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("[Upload Error] upload timed out after %v", timeout)
	}
}

// rateLimitedReader 实现限速读取
type rateLimitedReader struct {
	Reader  io.Reader
	Limiter *rate.Limiter
	Ctx     context.Context // 新增
}

func (r *rateLimitedReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if n > 0 {
		err = r.Limiter.WaitN(r.Ctx, n) // 使用 r.Ctx 替代 nil
	}
	return
}

// UploadManager 管理多个SFTP服务器
type UploadManager struct {
	clients []*SFTPClient
	config  UploadConfig
}

// NewUploadManager 创建上传管理器
func NewUploadManager(config UploadConfig) (*UploadManager, error) {
	var clients []*SFTPClient
	for _, server := range config.Servers {
		client, err := NewSFTPClient(server)
		if err != nil {
			log.Printf("Failed to create client for %s:%d: %v", server.Host, server.Port, err)
			continue
		}
		clients = append(clients, client)
	}
	if len(clients) == 0 {
		return &UploadManager{clients: clients, config: config}, fmt.Errorf("no valid SFTP clients created")
	}
	return &UploadManager{clients: clients, config: config}, nil
}

// Close 关闭所有客户端
func (m *UploadManager) Close() {
	for _, client := range m.clients {
		client.Close()
	}
}

// UploadToAll 上传文件到所有服务器
func (m *UploadManager) UploadToAll(filePath string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, client := range m.clients {
		wg.Add(1)
		go func(client *SFTPClient, index int) {
			defer wg.Done()
			err := client.UploadFileWithRetry(filePath, client.config.Path, m.config)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("server %d (%s:%d): %v", index, client.config.Host, client.config.Port, err)
				}
				mu.Unlock()
			}
		}(client, i)
	}

	wg.Wait()
	return firstErr
}

func (t *TaskInfo) uploadFile(filePath string) error {

	err := t.Upload.sftpUploadManager.UploadToAll(filePath)
	if err != nil {
		return err
	}
	return nil

}
