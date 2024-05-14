package main

import "fmt"

func Run() {
	tasks := readConf("./config.yaml")

	for taskName, task := range tasks.TaskInfos {
		fmt.Println(taskName)
		fmt.Printf("  - 输出格式：%s \n", task.OutputFormatString)
		fmt.Printf("  - 规则文件：%s %s \n", task.FilterIpRuler, task.FilterDomainRuler)
		fmt.Printf("  - 输出目录：%s \n", task.OutputDir)
		fmt.Printf("  - 输出文件最大size：%d (%dM) \n", task.FileMaxSize, task.FileMaxSize/1024/1024)
		fmt.Printf("  - 输出文件最大时间间隔：%s \n", task.FileMaxTime)

	}

	//task := tasks.TaskInfos["aliyun"]

	//start1 := time.Now()
	//for i := 0; i < 5000000; i++ {
	//	task.IpFilterRuler.Load(i)
	//}
	//fmt.Println(time.Since(start1))
	//
	//start2 := time.Now()
	//for i := 0; i < 5000000; i++ {
	//	task.v6WithInRange("1.1.1.1")
	//}
	//fmt.Println(time.Since(start2))
	//
	//start3 := time.Now()
	//
	//for i := 0; i < 5000000; i++ {
	//	task.DomainFilterRuler.Search("aaa")
	//}
	//fmt.Println(time.Since(start3))

	go tasks.execTransfer()

	if tasks.OnlineMode {
		tasks.watchSingleDir()
	} else {
		tasks.offlineWatch()
	}
}
