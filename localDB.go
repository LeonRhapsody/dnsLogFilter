package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// domainIncrement 将域名发送到通道
func (dc *DomainCounter) domainIncrement(domain string) {
	dc.cond.L.Lock()
	// 如果正在 flush，等待直到 flush 完成
	for dc.flushing {
		dc.cond.Wait()
	}
	// 发送到通道（通道满时会阻塞）
	dc.counts <- domain
	dc.cond.L.Unlock()
}

// collect 收集通道中的域名并计数
func (dc *DomainCounter) collect() {
	for {
		select {
		case domain, ok := <-dc.counts:
			if !ok {
				return // 通道关闭，退出
			}
			// 合并计数
			dc.mu.Lock()
			dc.merged[domain]++
			dc.mu.Unlock()
		case <-dc.flushReq:
			// 收到 flush 请求，暂停并确认
			dc.flushDone <- struct{}{}
			// 等待 flushing 结束
			dc.cond.L.Lock()
			for dc.flushing {
				dc.cond.Wait()
			}
			dc.cond.L.Unlock()
		}
	}
}

// write 将计数持久化到文件并重置 map
func (dc *DomainCounter) write() error {
	start := time.Now()
	dc.cond.L.Lock()
	// 标记正在 flush，暂停新的写操作
	dc.flushing = true
	dc.cond.L.Unlock()

	// 通知收集器暂停
	dc.flushReq <- struct{}{}
	// 等待收集器确认暂停
	<-dc.flushDone

	// 处理通道中剩余消息
	dc.mu.Lock()

	// 获取合并后的计数
	merged := dc.merged
	dc.merged = make(map[string]int) // 重置 merged map
	dc.mu.Unlock()

	// 恢复 flushing 标志并通知等待的写进程和收集器
	dc.cond.L.Lock()
	dc.flushing = false
	dc.cond.Broadcast()
	dc.cond.L.Unlock()

	// 读取现有计数
	existingCounts := make(map[string]int)
	file, err := os.OpenFile(dc.countsFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("警告: 无法打开计数文件 %s: %v", dc.countsFile, err)
		return fmt.Errorf("无法打开计数文件: %v", err)
	}
	if file != nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				log.Printf("格式错误: %q", line)
				continue
			}

			domain := strings.TrimSpace(parts[0])
			countStr := strings.TrimSpace(parts[1])

			count, err := strconv.Atoi(countStr)
			if err != nil {
				log.Printf("无法将 count 转换为整数: %q，错误: %v", line, err)
				continue
			}

			existingCounts[domain] = count
		}
		if err := scanner.Err(); err != nil {
			log.Printf("警告: 读取计数文件失败: %v", err)
			file.Close()
			return fmt.Errorf("读取计数文件失败: %v", err)
		}
		// 移动文件指针到开头，为后续写入准备
		if _, err := file.Seek(0, 0); err != nil {
			file.Close()
			return fmt.Errorf("重置文件指针失败: %v", err)
		}
		// 清空文件内容（仅在写入前）
		if err := file.Truncate(0); err != nil {
			file.Close()
			return fmt.Errorf("清空文件失败: %v", err)
		}
	} else {
		// 文件不存在，创建新文件
		file, err = os.Create(dc.countsFile)
		if err != nil {
			return fmt.Errorf("创建计数文件失败: %v", err)
		}
	}
	defer file.Close()

	// 合并新计数
	for domain, count := range merged {
		existingCounts[domain] += count
	}
	log.Printf("[Debug] 合并计数: %d 个域", len(existingCounts))

	// 写入文件
	writer := bufio.NewWriter(file)
	for domain, count := range existingCounts {
		if _, err := fmt.Fprintf(writer, "%s:%d\n", domain, count); err != nil {
			log.Printf("警告: 写入计数文件失败: %v", err)
			return fmt.Errorf("写入计数文件失败: %v", err)
		}
	}
	if err := writer.Flush(); err != nil {
		log.Printf("警告: 刷新计数文件失败: %v", err)
		return fmt.Errorf("刷新计数文件失败: %v", err)
	}

	log.Printf("[Sync] %d 个域 write 完成，耗时 %v", len(merged), time.Since(start))
	return nil
}
