package main

import (
	"bufio"
	"fmt"
	"math/big"
	"net"
	"os"
)

// ipv6TrieNode节点定义
type ipv6TrieNode struct {
	children [2]*ipv6TrieNode
	isEnd    bool // 标记是否是地址段末尾
}

// Trie定义
type Trie struct {
	root *ipv6TrieNode
}

// 创建新的Trie
func NewTrie() *Trie {
	return &Trie{root: &ipv6TrieNode{}}
}

// 将IPv6地址字符串转换为*big.Int
func ipv6ToBigInt(ip string) *big.Int {
	parsedIP := net.ParseIP(ip).To16()
	return big.NewInt(0).SetBytes(parsedIP)
}

// 找出两个IPv6地址的公共前缀长度（通过异或找不同）
func findCommonPrefixLength(start, end *big.Int) int {
	xor := big.NewInt(0).Xor(start, end) // 对起始和结束IP进行异或操作
	bits := xor.BitLen()                 // 获取异或结果中最左边1的位置
	return 128 - bits                    // 公共前缀的长度
}

// 将前缀长度转换为二进制表示
func getPrefixBinary(ip *big.Int, prefixLength int) string {
	// 将IP转换为二进制表示的字符串，并截取前缀部分
	binaryStr := fmt.Sprintf("%0128b", ip)
	return binaryStr[:prefixLength]
}

// 插入IPv6地址的二进制表示
func (t *Trie) insertBinary(binaryIP string) {
	node := t.root
	for i := 0; i < len(binaryIP); i++ {
		bit := int(binaryIP[i] - '0') // 获取当前位
		if node.children[bit] == nil {
			node.children[bit] = &ipv6TrieNode{}
		}
		node = node.children[bit]
	}
	node.isEnd = true // 标记地址段的末尾
}

// 插入IPv6范围
func (t *Trie) InsertRange(startIP, endIP string) error {
	start := ipv6ToBigInt(startIP)
	end := ipv6ToBigInt(endIP)
	prefixLength := findCommonPrefixLength(start, end)
	prefixBinary := getPrefixBinary(start, prefixLength)

	// 插入公共前缀的二进制表示
	t.insertBinary(prefixBinary)
	return nil
}

// Search 查找IPv6地址是否在Trie中
func (t *Trie) Search(ip string) bool {
	binaryIP := fmt.Sprintf("%0128b", ipv6ToBigInt(ip))
	node := t.root
	for i := 0; i < len(binaryIP); i++ {
		bit := int(binaryIP[i] - '0')
		if node.children[bit] == nil {
			return false
		}
		node = node.children[bit]
		if node.isEnd {
			return true // 匹配成功
		}
	}
	return false
}

func ipV6ListToTree(filename []string) *TrieNode {
	counter := 0
	var files string

	trie := NewTrieNode()

	// Insert domains into the v6Trie

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
			trie.Insert(scanner.Text())
			counter++
		}

		File.Close()
	}
	fmt.Printf("%s read %d domain rules\n", files, counter)

	return trie
}

func test() {
	trie := NewTrie()
	// 插入IP范围
	trie.InsertRange("2409:8720:2000:0000:0000:0000:0000:0000", "2409:8720:2000:0000:FFFF:FFFF:FFFF:FFFF")
	trie.InsertRange("2409:8720:2001:0000:0000:0000:0000:0000", "2409:8720:2001:0000:FFFF:FFFF:FFFF:FFFF")
	trie.InsertRange("2409:8720:2002:0000:0000:0000:0000:0000", "2409:8720:2002:0000:FFFF:FFFF:FFFF:FFFF")

	fmt.Println(trie.Search("2409:8720:2000:0000:0000:0000:0000:0001")) // 输出 true
	fmt.Println(trie.Search("2409:8720:2001:0000:FFFF:FFFF:FFFF:FFFF")) // 输出 true
	fmt.Println(trie.Search("2409:8720:2003:0000:0000:0000:0000:0000")) // 输出 false
}
