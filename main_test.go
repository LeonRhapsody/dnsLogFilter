package main

import "testing"

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
