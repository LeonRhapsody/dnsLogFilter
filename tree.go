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

// TrieNode represents a node in the trie
type TrieNode struct {
	children  map[string]*TrieNode
	isEnd     bool // isEnd marks the end of a domain
	matchType byte
}

// NewTrieNode creates a new Trie node
func NewTrieNode() *TrieNode {
	return &TrieNode{children: make(map[string]*TrieNode)}
}

// Insert inserts a domain into the trie
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

// Search searches for a domain in the trie
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

// V6Search searches for a domain in the trie
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

func V6ListToTree(filename []string) *TrieNode {
	counter := 0
	var files string

	trie := NewTrieNode()

	// Insert domains into the trie

	for _, file := range filename {
		files = files + "/" + file

		File, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
		}

		scanner := bufio.NewScanner(File)
		for scanner.Scan() {
			if scanner.Text() == "" {
				continue
			}
			trie.v6Insert(scanner.Text())
			counter++
		}

		File.Close()
	}
	fmt.Printf("%s read %d v6 rules\n", files, counter)

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
