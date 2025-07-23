package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func IsGzipFile(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer file.Close()

	// 读取文件前两个字节
	buffer := make([]byte, 2)
	_, err = file.Read(buffer)
	if err != nil {
		fmt.Println(err)
		return false
	}

	// 判断文件类型是否为 gzip 格式
	fmt.Printf(http.DetectContentType(buffer))
	return strings.EqualFold(http.DetectContentType(buffer), "application/x-gzip")
}

func UngzipToFile(filename string) (string, error) {
	buffer := bytes.NewBuffer(nil)

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	// 解压缩文件
	outputFilename := strings.TrimSuffix(filename, ".gz")
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		return outputFilename, err
	}
	defer outputFile.Close()

	_, err = io.Copy(buffer, reader)
	if err != nil {
		return outputFilename, err
	}

	return outputFilename, nil
}

// UnGzipFile 解压缩 gzip 文件并返回内容
func UnGzipFile(fileName string) ([]byte, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	// 读取解压缩后的内容
	unzippedData, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, err
	}

	return unzippedData, nil
}

func UngzipToBuffer(filename string, buffer *bytes.Buffer) error {

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(buffer, reader)
	if err != nil {
		return err
	}

	return nil
}

func (T *Tasks) WriteGzLog(outFilePath string, resultBuffer *bytes.Buffer) {
	//L.FileLock.Lock()
	file, err := os.OpenFile(outFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("无法打开文件:", err)
	}
	defer file.Close()
	//defer L.FileLock.Unlock()

	// 创建Gzip写入器，并将其与缓冲区关联
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// 将内容写入缓冲区
	_, err = gzipWriter.Write(resultBuffer.Bytes())
	if err != nil {
		fmt.Println("写入内容失败:", err)
		return
	}
	resultBuffer.Reset()

	// 确保缓冲区中的数据刷新到文件
	err = gzipWriter.Flush()
	if err != nil {
		fmt.Println("刷新缓冲区失败:", err)
		return
	}

}

func (T *Tasks) WriteLog(outFilePath string, resultBuffer *bytes.Buffer) {
	//L.FileLock.Lock()
	file, err := os.OpenFile(outFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("无法打开文件:", err)
	}
	defer file.Close()
	//defer L.FileLock.Unlock()

	_, err = io.Copy(file, resultBuffer)
	if err != nil {
		fmt.Printf("%s 写入失败:%e", outFilePath, err)

	}

}

func WriteLog(outFilePath string, resultBuffer *bytes.Buffer) {
	//L.FileLock.Lock()
	file, err := os.OpenFile(outFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("无法打开文件:", err)
	}
	defer file.Close()
	//defer L.FileLock.Unlock()

	_, err = io.Copy(file, resultBuffer)
	if err != nil {
		fmt.Printf("%s 写入失败:%e", outFilePath, err)

	}

}

func (T *Tasks) backupFile(sourcePath string) error {
	if T.BackupDir == "" {
		return nil
	}

	// 从源文件路径中提取文件名
	filename := filepath.Base(sourcePath)
	// 将目标目录和文件名组合成完整的目标文件路径
	destinationPath := filepath.Join(T.BackupDir, filename)

	// 使用os.Rename函数移动文件
	err := os.Rename(sourcePath, destinationPath)
	if err != nil {
		// 如果发生错误，打印错误并退出
		log.Printf("Error moving file: %v", err.Error())
		//return err
	}
	log.Printf("[Backup] %s to %s\n", sourcePath, destinationPath)

	return nil
}

func (T *Tasks) deleteFile(filePath string) error {
	// 检查文件路径是否为空
	if filePath == "" {
		return nil
	}

	// 使用 os.Remove 函数删除文件
	err := os.Remove(filePath)
	if err != nil {
		// 如果发生错误，记录错误并返回
		log.Printf("Error deleting file: %v", err.Error())
		return err
	}

	// 记录删除操作
	log.Printf("[Delete] %s\n", filePath)
	return nil
}

func saveMap(map1 sync.Map, outFilePath string) {
	var result strings.Builder

	map1.Range(func(key, value any) bool {
		result.WriteString(fmt.Sprintf("%s|%d\n", key, value.(int)))
		return true

	})

	// 打开（或创建）文件，准备追加内容
	file, err := os.OpenFile(outFilePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return // 如果打开文件失败，提前退出函数
	}
	defer file.Close()

	// 将字符串读取器传递给 io.Copy
	_, err = io.Copy(file, strings.NewReader(result.String()))
	if err != nil {
		fmt.Printf("%s 写入失败: %v\n", outFilePath, err)
	}

	fmt.Printf("[save] write domain list to %s\n", outFilePath)
}

func sortByValueAndSaveMap(map1 sync.Map, outFilePath string) {
	var result strings.Builder
	var entries []struct {
		Key   string
		Value int
	}

	// 遍历 map 并收集键值对
	map1.Range(func(key, value any) bool {
		entries = append(entries, struct {
			Key   string
			Value int
		}{
			Key:   key.(string),
			Value: value.(int),
		})
		return true
	})

	// 根据值进行倒排排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value > entries[j].Value // 倒序
	})

	// 将排序后的键值对写入结果
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("%s|%d\n", entry.Key, entry.Value))
	}

	// 打开（或创建）文件，准备追加内容
	file, err := os.OpenFile(outFilePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return // 如果打开文件失败，提前退出函数
	}
	defer file.Close()

	// 将字符串读取器传递给 io.Copy
	_, err = io.Copy(file, strings.NewReader(result.String()))
	if err != nil {
		fmt.Printf("%s 写入失败: %v\n", outFilePath, err)
	}

	fmt.Printf("[save] write domain list to %s\n", outFilePath)
}
