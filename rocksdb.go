package main

//
//import (
//	"bufio"
//	"fmt"
//	"github.com/linxGnu/grocksdb"
//	"log"
//	"os"
//)
//
//// NewDomainCounter 初始化 RocksDB 和计数器
//func NewDomainCounter(dbPath string, numThreads int) (*DomainCounter, error) {
//	// 配置 RocksDB 选项
//	opts := grocksdb.NewDefaultOptions()
//	opts.SetCreateIfMissing(true)
//	opts.SetMaxBackgroundJobs(4)
//	opts.SetWriteBufferSize(64 * 1024 * 1024) // 64MB 写缓冲
//	opts.SetMaxOpenFiles(1000)
//	opts.SetCompression(grocksdb.SnappyCompression) // 启用压缩
//
//	db, err := grocksdb.OpenDb(opts, dbPath)
//	if err != nil {
//		return nil, fmt.Errorf("failed to open database: %v", err)
//	}
//
//	// 初始化每个线程的计数器
//	threads := make([]map[string]int, numThreads)
//	for i := 0; i < numThreads; i++ {
//		threads[i] = make(map[string]int)
//	}
//
//	return &DomainCounter{
//		threads:    threads,
//		db:         db,
//		wo:         grocksdb.NewDefaultWriteOptions(),
//		ro:         grocksdb.NewDefaultReadOptions(),
//		numThreads: numThreads,
//	}, nil
//}
//
//// domainIncrement 线程安全的域名计数
//func (dc *DomainCounter) domainIncrement(threadID int, domain string) {
//	// 线程内计数，无需锁
//	dc.threads[threadID][domain]++
//}
//
//// FlushToDB 合并计数器并批量写入数据库
//func (dc *DomainCounter) FlushToDB() error {
//	dc.mu.Lock()
//	defer dc.mu.Unlock()
//
//	// 合并所有线程的计数器
//	merged := make(map[string]int)
//	for _, counter := range dc.threads {
//		for domain, count := range counter {
//			merged[domain] += count
//		}
//	}
//
//	// 批量写入 RocksDB
//	batch := grocksdb.NewWriteBatch()
//	defer batch.Destroy()
//
//	for domain, count := range merged {
//		// 获取当前计数
//		current, err := dc.db.Get(dc.ro, []byte(domain))
//		if err != nil {
//			return fmt.Errorf("failed to read domain %s: %v", domain, err)
//		}
//		defer current.Free()
//
//		newCount := count
//		if current.Data() != nil {
//			var existing int
//			fmt.Sscanf(string(current.Data()), "%d", &existing)
//			newCount += existing
//
//		}
//
//		// 写入新计数
//		batch.Put([]byte(domain), []byte(fmt.Sprintf("%d", newCount)))
//	}
//
//	if err := dc.db.Write(dc.wo, batch); err != nil {
//		return fmt.Errorf("failed to write batch: %v", err)
//	}
//
//	// 清空所有线程计数器
//	for i := 0; i < dc.numThreads; i++ {
//		clear(dc.threads[i])
//	}
//
//	return nil
//}
//
//// / ExportToFile 导出数据到文件
//func (dc *DomainCounter) ExportToFile(outputPath string) error {
//	iterator := dc.db.NewIterator(dc.ro)
//	defer iterator.Close()
//
//	file, err := os.Create(outputPath)
//	if err != nil {
//		return fmt.Errorf("failed to create output file: %v", err)
//	}
//	defer file.Close()
//
//	// 使用缓冲写入
//	writer := bufio.NewWriter(file)
//	defer writer.Flush()
//
//	for iterator.SeekToFirst(); iterator.Valid(); iterator.Next() {
//		domain := string(iterator.Key().Data())
//		count := string(iterator.Value().Data())
//		_, err := fmt.Fprintf(writer, "%s: %s\n", domain, count)
//		if err != nil {
//			return fmt.Errorf("failed to write to file: %v", err)
//		}
//		iterator.Key().Free()
//		iterator.Value().Free()
//	}
//
//	if err := iterator.Err(); err != nil {
//		return fmt.Errorf("iterator error: %v", err)
//	}
//	log.Printf("[count domain to File] writr to %s\n", outputPath)
//
//	return nil
//}
//
//// Close 关闭数据库
//func (dc *DomainCounter) Close() {
//	dc.db.Close()
//}
