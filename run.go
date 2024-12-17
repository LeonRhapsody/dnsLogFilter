package main

import (
	"fmt"
	"github.com/LeonRhapsody/DNSLogFilter/manager"
)

func Run() {
	tasks := newTasks()

	fmt.Printf(
		"================运行信息============================\n"+
			"网卡名称：%s\n"+
			"分析线程数：%d\n"+
			"文件输入目录：%s\n"+
			"文件输入格式：%s\n"+
			"备份目录：%s\n"+
			"分析模式：%t\n",
		tasks.EthName,
		tasks.AnalyzeThreads,
		tasks.InputDir,
		tasks.InputFormat,
		tasks.BackupDir,
		tasks.OnlineMode)

	for taskName, task := range tasks.TaskInfos {
		fmt.Printf("%s\n"+
			"  - 输出格式：%s\n"+
			"  - 规则文件：%s %s\n"+
			"  - 输出目录：%s \n"+
			"  - 输出文件最大size：%d (%dM)\n"+
			"  - 输出文件最大时间间隔：%s \n"+
			"  - 过滤模式：%d \n",
			taskName,
			task.OutputFormatString,
			task.FilterIpRuler,
			task.FilterDomainRuler,
			task.OutputDir,
			task.FileMaxSize,
			task.FileMaxSize/1024/1024,
			task.FileMaxTime,
			task.FilterTag)
	}

	fmt.Println("=====================================================")

	if tasks.adminMode {
		ginConfig := manager.NewRouter("127.0.0.1", "35695")
		ginConfig.Router.GET("/status", tasks.getStatus)
		//ginConfig.Router.PUT("/reload", tasks.reload)
		ginConfig.AddRouter()
		go ginConfig.Run()
	}

	go tasks.execTransfer()

	if tasks.OnlineMode {
		tasks.watchSingleDir()
	} else {
		tasks.offlineWatch()
	}
}
