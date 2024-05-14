package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// TrieNode represents a node in the trie
type TrieNode struct {
	children map[string]*TrieNode
	isEnd    bool // isEnd marks the end of a domain
}

// NewTrieNode creates a new Trie node
func NewTrieNode() *TrieNode {
	return &TrieNode{children: make(map[string]*TrieNode)}
}

// Insert inserts a domain into the trie
func (t *TrieNode) Insert(domain string) {
	parts := splitDomain(domain)
	node := t
	for _, part := range parts {
		if _, ok := node.children[part]; !ok {
			node.children[part] = NewTrieNode()
		}
		node = node.children[part]
	}
	node.isEnd = true
}

// Search searches for a domain in the trie
func (t *TrieNode) Search(domain string) bool {
	parts := splitDomain(domain)
	node := t
	for _, part := range parts {
		if node.isEnd {
			return true // Match found due to wildcard
		}
		if _, ok := node.children[part]; !ok {
			return false
		}
		node = node.children[part]
	}
	return node.isEnd
}

// Traverse 方法遍历 Trie 并打印所有域名
// Traverse 方法遍历 Trie 并打印所有域名
func (t *TrieNode) Traverse(parts []string) {

	var result strings.Builder
	if t.isEnd {
		// 逆序输出 parts 切片以还原域名的正常顺序
		for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
			parts[i], parts[j] = parts[j], parts[i]
		}
		result.WriteString(strings.Join(parts, ".") + "\n")

	}
	for part, child := range t.children {
		child.Traverse(append([]string{part}, parts...))
	}

	//write(result.String())

}

func reverseSlice(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i] // Swap elements
	}
	return s
}

// splitDomain splits the domain into parts
func splitDomain(domain string) []string {
	// For simplicity, we assume the domain parts are separated by dots.
	// In a real-world scenario, you should consider IDN (internationalized domain names) and punycode.

	parts := strings.Split(domain, ".")

	return reverseSlice(parts)
}

func DomainListToTree(filename []string) *TrieNode {
	trie := NewTrieNode()

	// Insert domains into the trie

	for _, file := range filename {
		File, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
		}

		scanner := bufio.NewScanner(File)
		for scanner.Scan() {
			if scanner.Text() == "" {
				continue
			}
			trie.Insert(scanner.Text())
		}

		File.Close()
	}

	return trie
}

func treeTest() {
	trie := NewTrieNode()

	// Insert domains into the trie

	File, err := os.Open("1w.txt")
	if err != nil {
		fmt.Println(err)
	}

	scanner := bufio.NewScanner(File)
	for scanner.Scan() {
		trie.Insert(scanner.Text())
	}

	File.Close()

	File2, err := os.Open("target.list")
	if err != nil {
		fmt.Println(err)
	}

	match := 0
	nums := 0
	start := time.Now()
	scanner2 := bufio.NewScanner(File2)
	for scanner2.Scan() {
		nums++

		if trie.Search(scanner2.Text()) {
			match++
			fmt.Println(scanner2.Text())
		}
	}

	File2.Close()

	times := time.Since(start)
	qps := int(float64(nums) / times.Seconds())
	fmt.Printf("times: %s,qps: %d,match %d", times, qps, match)
}
