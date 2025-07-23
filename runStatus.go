package main

import (
	"fmt"
	"runtime"
)

func printDetailedStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	//fmt.Println("=== 系统运行时统计 ===")
	// 内存相关
	fmt.Printf("Alloc: %v MiB (当前分配的内存)\n", m.Alloc/1024/1024)
	//fmt.Printf("HeapAlloc: %v MiB (堆上分配的内存)\n", m.HeapAlloc/1024/1024)
	//fmt.Printf("HeapInuse: %v MiB (使用中的堆内存)\n", m.HeapInuse/1024/1024)
	//fmt.Printf("HeapIdle: %v MiB (空闲的堆内存)\n", m.HeapIdle/1024/1024)
	//fmt.Printf("Sys: %v MiB (从操作系统获取的总内存)\n", m.Sys/1024/1024)
	//fmt.Printf("NumGC: %v (垃圾回收次数)\n", m.NumGC)

	// CPU 和 goroutine 相关
	//fmt.Printf("NumGoroutine: %v (当前 goroutine 数量)\n", runtime.NumGoroutine())
	////fmt.Printf("NumCPU: %v (可用 CPU 核心数)\n", runtime.NumCPU())
	//fmt.Println("====================")
}
