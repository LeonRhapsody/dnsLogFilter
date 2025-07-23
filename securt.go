package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
)

var (
	inputString string
	keyFlag     = []byte("0123456789abcdefghijklmnopqrstuv")
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "enc",
		Short: "加密字符串",
		Long:  "加密字符串",
		Run: func(cmd *cobra.Command, args []string) {
			result := encString(inputString)
			fmt.Printf("加密结果: %s\n", result)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "dec",
		Short: "解密字符串",
		Long:  "解密字符串",
		Run: func(cmd *cobra.Command, args []string) {
			result := decString(inputString)
			fmt.Printf("解密结果: %s\n", result)
		},
	})
	rootCmd.PersistentFlags().StringVarP(&inputString, "input", "s", "", "input")

}

// encString 加密输入字符串
func encString(input string) string {
	// 确保密钥长度为 32 字节
	key := []byte(keyFlag)
	if len(key) != 32 {
		log.Fatal("密钥必须为 32 字节")
	}

	// 创建 AES 加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("创建加密器失败: %v", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("创建 GCM 失败: %v", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Fatalf("生成 nonce 失败: %v", err)
	}

	// 加密并附加 nonce
	cipherText := gcm.Seal(nonce, nonce, []byte(input), nil)
	// 转换为 base64
	return base64.StdEncoding.EncodeToString(cipherText)
}

// decString 解密输入字符串
func decString(input string) string {
	// 确保密钥长度为 32 字节
	key := []byte(keyFlag)
	if len(key) != 32 {
		log.Fatal("密钥必须为 32 字节")
	}

	// 解码 base64
	data, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		log.Fatalf("解码 base64 失败: %v", err)
	}

	// 创建 AES 加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("创建加密器失败: %v", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("创建 GCM 失败: %v", err)
	}

	// 提取 nonce 和密文
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		log.Fatal("无效的密文")
	}
	nonce, cipherText := data[:nonceSize], data[nonceSize:]

	// 解密
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		log.Fatalf("解密失败: %v", err)
	}

	return string(plainText)
}
