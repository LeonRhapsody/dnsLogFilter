package main

import (
	"fmt"
	"testing"
)

func BenchmarkDomainListToTree(b *testing.B) {
	a := DomainListToTree([]string{"apt.list"})
	a.Search("www.baidu.com")

}

func TestDomainListToTree(t *testing.T) {

	a := DomainListToTree([]string{"apt.list"})
	if a.Search("www.baidu.com") != false {
		t.Errorf("test failed")
	}
}

func main() {
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
