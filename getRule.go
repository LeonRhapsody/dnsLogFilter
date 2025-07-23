package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MatchRule 封装 IP 清单的结构体
type MatchRule struct {
	sync.RWMutex               // 读写锁
	v4ListMap        *sync.Map // IPv4 地址存储
	v6Trie           *TrieNode // IPv6 地址存储
	domainTrie       *TrieNode //domain 存储
	ipRulerFiles     []string  // IP 清单文件列表
	domainRulerFiles []string  // IP 清单文件列表
	ipFilterMode     int       // 模式标志
	fileModTimeMap   map[string]time.Time
}

// 查询并写入 domain 的函数
func (t *TaskInfo) queryAndWriteDomains(filename string) {

	start := time.Now()
	// 构造 DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True", t.UmpMysqlUser, decString(t.UmpMysqlPass), t.UmpMysqlHost, t.UmpMysqlPort, t.DbName)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("数据库连接失败: %v\n", err)
		return
	}
	defer db.Close()

	// 执行查询
	rows, err := db.Query(t.ForceDomainSql)
	if err != nil {
		log.Printf("查询失败: %v\n", err)
		return
	}
	defer rows.Close()

	// 创建或覆盖文件
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("创建文件失败: %v\n", err)
		return
	}
	defer file.Close()

	// 创建带缓冲的 Writer（默认缓冲区 4KB，可以自定义如 bufio.NewWriterSize(file, 64*1024)）
	writer := bufio.NewWriterSize(file, 64*1024)
	defer writer.Flush()

	// 遍历结果写入文件
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			log.Printf("读取字段失败: %v\n", err)
			continue
		}
		writer.WriteString(domain + "\n")
	}
	log.Printf("[query database] 读取数据库完成(%v)，domain 列表已写入 %s\n", time.Since(start), filename)
}

func (r *MatchRule) checkFileModTime() bool {

	if len(r.ipRulerFiles) > 0 {
		for _, file := range r.ipRulerFiles {
			info, err := os.Stat(file)
			if err != nil {
				fmt.Printf("获取文件信息失败: %v\n", err)
				continue
			}
			if lastModTime, ok := r.fileModTimeMap[file]; ok {
				if info.ModTime() != lastModTime {
					r.fileModTimeMap[file] = info.ModTime()
					return true
				}

			}
		}

	}
	return false
}

// NewMatchRule 初始化 IPListCache
func (t *TaskInfo) NewMatchRule(ipListFiles []string, domainListFiles []string) {
	t.taskMatchRule = &MatchRule{
		v4ListMap:        &sync.Map{},
		v6Trie:           NewTrieNode(),
		domainTrie:       NewTrieNode(),
		ipRulerFiles:     ipListFiles,
		domainRulerFiles: domainListFiles,
	}
	t.RefreshIPList() // 初次加载
}

// RefreshIPList 刷新 IP 清单
func (t *TaskInfo) RefreshIPList() {

	//申明写意图
	atomic.StoreInt32(&t.writeFlag, 1)
	defer atomic.StoreInt32(&t.writeFlag, 0)
	//加锁
	t.writeLock.Lock()
	defer t.writeLock.Unlock()

	var (
		v4Counter     int
		v6Counter     int
		domainCounter int
	)

	// 创建新的 sync.Map 和 TrieNode，避免直接修改现有数据
	newV4ListMap := &sync.Map{}
	newV6Trie := NewTrieNode()
	newDomainTrie := NewTrieNode()

	if len(t.taskMatchRule.ipRulerFiles) != 0 {
		// 遍历文件并加载 IP
		for _, file := range t.taskMatchRule.ipRulerFiles {
			fileHandle, err := os.Open(file)
			if err != nil {
				fmt.Printf("Error opening file %s: %v\n", file, err)
				continue
			}
			defer fileHandle.Close()

			scanner := bufio.NewScanner(fileHandle)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}

				if strings.Contains(line, ":") {
					newV6Trie.v6Insert(line)
					v6Counter++
				} else {
					ips, err := parseIPFormat(line)
					if err != nil {
						fmt.Printf("Error parsing IP format in %s: %v\n", line, err)
						continue
					}
					for _, ip := range ips {
						newV4ListMap.Store(ip, struct{}{})
						v4Counter++
					}
				}
			}

			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading file %s: %v\n", file, err)
			}
		}
	}

	if len(t.taskMatchRule.domainRulerFiles) != 0 {

		for _, file := range t.taskMatchRule.domainRulerFiles {
			fileHandle, err := os.Open(file)
			if err != nil {
				fmt.Printf("Error opening file %s: %v\n", file, err)
			}
			defer fileHandle.Close()

			scanner := bufio.NewScanner(fileHandle)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				newDomainTrie.Insert(line)
				domainCounter++
			}
			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading file %s: %v\n", file, err)
			}
		}
	}

	// 更新缓存
	t.taskMatchRule.v4ListMap = newV4ListMap
	t.taskMatchRule.v6Trie = newV6Trie
	t.taskMatchRule.domainTrie = newDomainTrie
	fmt.Printf("Refreshed %d v4IP and %d v6IP rules from files: %s\n", v4Counter, v6Counter, strings.Join(t.taskMatchRule.ipRulerFiles, ", "))
	fmt.Printf("Refreshed %d domain rules from files: %s\n", domainCounter, strings.Join(t.taskMatchRule.domainRulerFiles, ", "))

	// 设置 ipFilterMode
	if v4Counter > 0 && v6Counter > 0 {
		t.taskMatchRule.ipFilterMode = 3
	} else if v4Counter > 0 {
		t.taskMatchRule.ipFilterMode = 2
	} else if v6Counter > 0 {
		t.taskMatchRule.ipFilterMode = 1
	} else {
		t.taskMatchRule.ipFilterMode = 0
	}

}

// GetListMap 获取当前的 sync.Map（只读）
func (r *MatchRule) GetListMap() *sync.Map {
	r.RLock()
	defer r.RUnlock()
	return r.v4ListMap
}

// GetTrie 获取当前的 TrieNode（只读）
func (r *MatchRule) GetTrie() *TrieNode {
	r.RLock()
	defer r.RUnlock()
	return r.v6Trie
}

// GetFilterMode 获取当前的 ipFilterMode
func (r *MatchRule) GetFilterMode() int {
	r.RLock()
	defer r.RUnlock()
	return r.ipFilterMode
}
