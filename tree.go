package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	Exact    byte = 1
	Multiple byte = 2
	notMatch byte = 0
)

// TrieNode represents a node in the v6Trie
type TrieNode struct {
	children  map[string]*TrieNode
	isEnd     bool // isEnd marks the end of a domain
	matchType byte
}

// NewTrieNode creates a new Trie node
func NewTrieNode() *TrieNode {
	return &TrieNode{children: make(map[string]*TrieNode)}
}

// Insert inserts a domain into the v6Trie
// Insert inserts a domain into the v6Trie
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

func (t *TrieNode) v6Insert(domain string) {
	parts := strings.Split(domain, ":")
	node := t
	for _, part := range parts {
		if _, ok := node.children[part]; !ok {
			node.children[part] = NewTrieNode()
		}
		node = node.children[part]
	}
	node.isEnd = true
}

func (t *TrieNode) print() {
	printNode(t)
}

func printNode(node *TrieNode) {
	for nodeName, trieNode := range node.children {
		fmt.Printf("NodeName: %s, Address: %p, isEND: %v,Value: %+v\n", nodeName, trieNode, trieNode.isEnd, trieNode)
		printNode(trieNode)
	}
}

// Search searches for a domain in the v6Trie
func (t *TrieNode) Search(domain string) bool {
	//todo: search时，node.isEND代表的是上一级的结果，当前层级的isEND应该在下一级展示,暂未研究是否可以优化insert

	parts := splitDomain(domain)
	node := t

	for i, part := range parts {

		//fmt.Printf("i:%d, Search for: %s, isEnd: %v, MatchType: %v\n", i, part, node.isEnd, node.matchType)
		//if _, ok := node.children[part]; ok {
		//	fmt.Println(node.children[part].children)
		//}

		if _, ok := node.children[part]; !ok {
			return false
		}

		//如果下一级存在*的匹配，立即返回true
		if _, ok := node.children[part].children["*"]; ok {
			return true
		}

		if node.children[part].isEnd && i == len(parts)-1 {
			return true // Match found due to wildcard
		}

		node = node.children[part]

	}

	return false
}

// V6Search searches for a domain in the v6Trie
func (t *TrieNode) V6Search(domain string) bool {
	//todo: search时，node.isEND代表的是上一级的结果，当前层级的isEND应该在下一级展示,暂未研究是否可以优化insert

	parts := strings.Split(domain, ":")
	node := t

	for i, part := range parts {

		//fmt.Printf("i:%d, Search for: %s, isEnd: %v, MatchType: %v\n", i, part, node.isEnd, node.matchType)
		//if _, ok := node.children[part]; ok {
		//	fmt.Println(node.children[part].children)
		//}

		if node.children == nil {
			return false
		}
		if _, ok := node.children[part]; !ok {
			return false
		}

		//如果下一级存在*的匹配，立即返回true
		if _, ok := node.children[part].children["*"]; ok {
			return true
		}

		if node.children[part].isEnd && i == len(parts)-1 {
			return true // Match found due to wildcard
		}

		node = node.children[part]

	}

	return false
}

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

// splitDomain splits the domain into parts and reverses the order
func splitDomain(domain string) []string {
	// For simplicity, we assume the domain parts are separated by dots.
	parts := strings.Split(domain, ".")

	// 反转切片
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i] // Swap elements
	}
	return parts
}

// V6ListToTree 读取文件中的 IPv6 地址并构建 Trie 树
func V6ListToTree(filenames []string) *TrieNode {
	counter := 0
	trie := NewTrieNode()

	if len(filenames) == 0 {
		return trie
	}

	for _, file := range filenames {
		// 打开文件
		fileHandle, err := os.Open(file)
		if err != nil {
			fmt.Printf("Error opening file %s: %v\n", file, err)
			continue
		}
		defer fileHandle.Close() // 确保文件句柄被正确关闭

		// 扫描文件内容
		scanner := bufio.NewScanner(fileHandle)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue // 跳过空行
			}
			trie.v6Insert(line)
			counter++
		}

		// 检查扫描错误
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
		}
	}

	fmt.Printf("Read %d IPv6 rules from files: %s\n", counter, strings.Join(filenames, ", "))

	return trie
}
func treeTest() {
	tree := NewTrieNode()

	tree.Insert("www.baidu.com")
	tree.Insert("*.www.sina.com")
	//tree.Insert("sina.com")

	tree.print()
	fmt.Println(tree.Search("3.2.1.www.baidu.com"))
	fmt.Println(tree.Search("2.1.www.baidu.com"))
	fmt.Println(tree.Search("1.www.baidu.com"))
	fmt.Println(tree.Search("www.baidu.com"))
	fmt.Println(tree.Search("aaa.baidu.com"))
	fmt.Println(tree.Search("baidu.com"))
	fmt.Println(tree.Search("com"))

	fmt.Println(tree.Search("3.2.1.www.sina.com"))
	fmt.Println(tree.Search("2.1.www.sina.com"))
	fmt.Println(tree.Search("1.www.sina.com"))
	fmt.Println(tree.Search("www.sina.com"))
	fmt.Println(tree.Search("aaa.sina.com"))
	fmt.Println(tree.Search("sina.com"))
	fmt.Println(tree.Search("com"))
}
