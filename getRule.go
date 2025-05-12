package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
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
func NewMatchRule(ipListFiles []string, domainListFiles []string) *MatchRule {
	cache := &MatchRule{
		v4ListMap:        &sync.Map{},
		v6Trie:           NewTrieNode(),
		domainTrie:       NewTrieNode(),
		ipRulerFiles:     ipListFiles,
		domainRulerFiles: domainListFiles,
	}
	cache.RefreshIPList() // 初次加载
	return cache
}

// RefreshIPList 刷新 IP 清单
func (r *MatchRule) RefreshIPList() {
	r.Lock()         // 写锁，确保刷新时独占
	defer r.Unlock() // 确保解锁

	var (
		v4Counter     int
		v6Counter     int
		domainCounter int
	)

	// 创建新的 sync.Map 和 TrieNode，避免直接修改现有数据
	newV4ListMap := &sync.Map{}
	newV6Trie := NewTrieNode()
	newDomainTrie := NewTrieNode()

	if len(r.ipRulerFiles) != 0 {
		// 遍历文件并加载 IP
		for _, file := range r.ipRulerFiles {
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

	if len(r.domainRulerFiles) != 0 {

		for _, file := range r.domainRulerFiles {
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
	r.v4ListMap = newV4ListMap
	r.v6Trie = newV6Trie
	r.domainTrie = newDomainTrie
	fmt.Printf("Refreshed %d v4IP and %d v6IP rules from files: %s\n", v4Counter, v6Counter, strings.Join(r.ipRulerFiles, ", "))
	fmt.Printf("Refreshed %d domain rules from files: %s\n", domainCounter, strings.Join(r.domainRulerFiles, ", "))

	// 设置 ipFilterMode
	if v4Counter > 0 && v6Counter > 0 {
		r.ipFilterMode = 3
	} else if v4Counter > 0 {
		r.ipFilterMode = 2
	} else if v6Counter > 0 {
		r.ipFilterMode = 1
	} else {
		r.ipFilterMode = 0
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
