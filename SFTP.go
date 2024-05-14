package main

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"path/filepath"
)
import "github.com/jlaffaye/ftp"

func uploadFileToFTP(ftpServer, ftpUser, ftpPassword, localFilePath, remoteFilePath string) error {
	// 建立FTP连接
	client, err := ftp.Dial(ftpServer)
	if err != nil {
		return err
	}
	defer client.Quit()

	// 登录FTP服务器
	err = client.Login(ftpUser, ftpPassword)
	if err != nil {
		return err
	}

	// 打开本地文件
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// 使用 client.Stor 上传文件到远程FTP服务器
	err = client.Stor(remoteFilePath, localFile)
	if err != nil {
		return err
	}

	return nil
}

func SFTP() {
	// SFTP 服务器设置
	var (
		sftpHost   = "your_sftp_host"
		sftpPort   = "22"
		sftpUser   = "your_username"
		sftpPass   = "your_password"
		remoteDir  = "/remote/directory" // 远程目录
		localFile  = "local_file.gz"     // 本地文件
		remoteFile = "remote_file.tmp"   // 远程临时文件
	)

	// 设置 SSH 客户端配置
	config := &ssh.ClientConfig{
		User: sftpUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sftpPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接到 SSH
	conn, err := ssh.Dial("tcp", net.JoinHostPort(sftpHost, sftpPort), config)
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// 创建 SFTP 客户端
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatalf("unable to start SFTP subsystem: %v", err)
	}
	defer client.Close()

	// 打开本地文件
	srcFile, err := os.Open(localFile)
	if err != nil {
		log.Fatalf("unable to open local file: %v", err)
	}
	defer srcFile.Close()

	// 创建远程文件
	dstFile, err := client.Create(filepath.Join(remoteDir, remoteFile))
	if err != nil {
		log.Fatalf("unable to create remote file: %v", err)
	}
	defer dstFile.Close()

	// 将本地文件复制到远程文件
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		log.Fatalf("unable to copy to remote file: %v", err)
	}

	// 重命名远程文件（从 .tmp 到 .gz）
	if err := client.Rename(filepath.Join(remoteDir, remoteFile), filepath.Join(remoteDir, filepath.Base(localFile))); err != nil {
		log.Fatalf("unable to rename remote file: %v", err)
	}

	fmt.Println("File transferred and renamed successfully")
}
