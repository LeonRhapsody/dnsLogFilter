package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ValueTrieNode represents a node in the trie
type ValueTrieNode struct {
	children  map[string]*ValueTrieNode
	isEnd     bool // isEnd marks the end of a domain
	matchType byte
	value     []int
}

// NewValueTrieNode New ValueTrieNode creates a new Trie node
func NewValueTrieNode() *ValueTrieNode {
	return &ValueTrieNode{children: make(map[string]*ValueTrieNode)}
}

// Insert inserts a domain into the trie
func (t *ValueTrieNode) Insert(key string, value int) {

	var parts []string
	if strings.Contains(key, ":") {
		parts = strings.Split(key, ":")

	} else {
		parts = splitDomain(strings.TrimSuffix(key, "."))

	}
	node := t
	for _, part := range parts {
		if _, ok := node.children[part]; !ok {
			node.children[part] = NewValueTrieNode()
		}
		node = node.children[part]
	}

	node.isEnd = true
	node.value = append(node.value, value)
}

func (t *ValueTrieNode) print() {
	printNode2(t)
}

func printNode2(node *ValueTrieNode) {
	for nodeName, trieNode := range node.children {
		fmt.Printf("NodeName: %s, Address: %p, isEND: %v,Value: %+v\n", nodeName, trieNode, trieNode.isEnd, trieNode)
		printNode2(trieNode)
	}
}

// Search SearchDomain Search searches for a domain in the trie
func (t *ValueTrieNode) Search(key string) (bool, []int) {

	var parts []string
	if strings.Contains(key, ":") {
		parts = strings.Split(key, ":")

	} else {
		parts = splitDomain(strings.TrimSuffix(key, "."))

	}
	node := t

	for i, part := range parts {

		//fmt.Printf("i:%d, Search for: %s, isEnd: %v, MatchType: %v\n", i, part, node.isEnd, node.matchType)
		//if _, ok := node.children[part]; ok {
		//	fmt.Println(node.children[part].children)
		//}

		if _, ok := node.children[part]; !ok {
			return false, nil
		}

		//如果下一级存在*的匹配，立即返回true
		if _, ok := node.children[part].children["*"]; ok {
			return true, node.children[part].children["*"].value
		}

		if node.children[part].isEnd && i == len(parts)-1 {
			return true, node.children[part].value // Match found due to wildcard
		}

		node = node.children[part]

	}

	return false, nil
}

// Traverse 方法遍历 Trie 并打印所有域名
func (t *ValueTrieNode) Traverse(parts []string) {

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

func expandGenerate(generate string) []string {
	// Split the $GENERATE directive
	parts := strings.Fields(generate)
	if len(parts) < 3 {
		fmt.Println("Invalid format:", generate)
		return nil
	}

	// Parse the range
	rangeParts := strings.Split(parts[1], "-")
	if len(rangeParts) != 2 {
		fmt.Println("Invalid range:", parts[1])
		return nil
	}

	start, err1 := strconv.Atoi(rangeParts[0])
	end, err2 := strconv.Atoi(rangeParts[1])
	if err1 != nil || err2 != nil {
		fmt.Println("Invalid range numbers:", rangeParts)
		return nil
	}

	// Generate the domain names
	template := parts[2]
	var result []string
	for i := start; i <= end; i++ {
		result = append(result, strings.Replace(template, "$", strconv.Itoa(i), 1))
	}
	return result
}
