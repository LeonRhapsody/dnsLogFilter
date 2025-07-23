package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func transferFormat(inputFormatStr string, outputFormatStr string) []int {
	var format []int
	inputFormat := strings.Split(inputFormatStr, ",")
	outputFormat := strings.Split(outputFormatStr, ",")

	for _, outSeg := range outputFormat {

		for index, inputSeg := range inputFormat {
			if inputSeg == outSeg {
				format = append(format, index)
				break
			}

			//由于DRMS日志中没有区分A、4A日志，所以使用DRMS做数据源又像单独输出A、4A日志的话，需要做特殊处理
			if outSeg == "17" {
				format = append(format, 10017)
				break

			}
			if outSeg == "18" {
				format = append(format, 10018)
				break

			}

			if index == len(inputFormat)-1 {
				fmt.Printf(" %s Does not belong to the category of input format\n", outSeg)
				os.Exit(1)

			}

		}

	}
	return format
}

func IPListToCache(filterListFiles []string) (*sync.Map, *TrieNode, int) {
	var (
		v4Counter     int
		v6Counter     int
		ipfileterMode int

		listMap sync.Map
	)
	trie := NewTrieNode()

	if len(filterListFiles) == 0 {
		return &listMap, trie, 0
	}

	for _, file := range filterListFiles {

		// 打开文件
		fileHandle, err := os.Open(file)
		if err != nil {
			fmt.Printf("Error opening file %s: %v\n", file, err)
			continue
		}
		defer fileHandle.Close() // 确保文件句柄被关闭

		// 逐行扫描文件内容
		scanner := bufio.NewScanner(fileHandle)
		for scanner.Scan() {

			line := strings.TrimSpace(scanner.Text())

			if line == "" {
				continue // 跳过空行
			}

			if strings.Contains(line, ":") {
				trie.v6Insert(line)
				v6Counter++
			} else {
				ips, err := parseIPFormat(scanner.Text())
				if err != nil {
					panic(fmt.Sprintf("Error parsing IP format: %v\n", err))
				}

				// 将 IP 存入 sync.Map
				for _, ip := range ips {
					listMap.Store(ip, struct{}{}) // 使用空结构体减少内存占用
					v4Counter++
				}
			}

		}

		// 检查扫描错误
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
		}
	}

	fmt.Printf("Read %d v4IP and %d v6IP rules from files: %s\n", v4Counter, v6Counter, strings.Join(filterListFiles, ", "))

	if v4Counter > 0 {
		if v6Counter > 0 {
			ipfileterMode = 3
		} else {
			ipfileterMode = 2
		}
		ipfileterMode = 1
	}

	return &listMap, trie, ipfileterMode
}

func IPListToTxt(FilterListFile []string) {
	counter := 0

	var buffer bytes.Buffer
	var files string

	for _, file := range FilterListFile {
		files = files + "/" + file

		File, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
		}

		scanner := bufio.NewScanner(File)
		for scanner.Scan() {
			ips, err := parseIPFormat(scanner.Text())
			if err != nil {
				fmt.Println(err)
			}
			for _, ip := range ips {
				buffer.WriteString(ip + "\n")
				counter++
			}

		}

		File.Close()
	}
	fmt.Printf("%s read %d ip rules\n", files, counter)
	file, err := os.OpenFile("ipv4.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("无法打开文件:", err)
	}
	defer file.Close()
	//defer L.FileLock.Unlock()

	_, err = io.Copy(file, &buffer)
	if err != nil {
		fmt.Printf("%s 写入失败:%e", "ipv4.txt", err)

	}

}

// DomainListToTree 读取文件中的域名并构建 Trie 树
func DomainListToTree(filenames []string) *TrieNode {
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
			trie.Insert(line)
			counter++
		}

		// 检查扫描错误
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
		}
	}

	fmt.Printf("Read %d domain rules from files: %s\n", counter, strings.Join(filenames, ", "))

	return trie
}

func fileExists(filename string) bool {
	// 使用 os.Stat 获取文件信息
	_, err := os.Stat(filename)
	// 判断错误是否为文件不存在
	if os.IsNotExist(err) {
		return false
	}
	return err == nil // 如果没有错误，则文件存在
}

func (T *Tasks) configValid() (bool, error) {
	if !fileExists(T.InputDir) {
		return invalid, fmt.Errorf("%s not exsit", T.InputDir)
	}

	for _, taskInfo := range T.TaskInfos {
		if !taskInfo.Enable {
			continue
		}
		if taskInfo.IsMatchResolveIP {
			if len(taskInfo.FilterIpRuler) == 0 {
				return invalid, fmt.Errorf("if option IsMatchResolveIP is True,FilterIpRuler can't be empty ")
			}
		}
		for _, filename := range taskInfo.FilterDomainRuler {
			if !fileExists(filename) {
				return invalid, fmt.Errorf("%s not exsit", filename)
			}
		}
		for _, filename := range taskInfo.FilterIpRuler {
			if !fileExists(filename) {
				return invalid, fmt.Errorf("%s not exsit", filename)
			}
		}
	}

	return valid, nil
}

func (t *TaskInfo) getTaskType() int {
	//匹配规则：
	//01 仅域名
	//10 仅请求IP
	//20 仅解析IP

	//11 请求IP+域名
	//21 解析IP+域名

	const domainOnly = 01
	const requestIP = 10
	const resolveIP = 20

	//仅IP
	if len(t.FilterDomainRuler) == 0 && len(t.FilterIpRuler) != 0 {
		if t.IsMatchResolveIP {
			return resolveIP
		} else {
			return requestIP
		}
	}

	//仅域名
	if len(t.FilterDomainRuler) != 0 && len(t.FilterIpRuler) == 0 {
		return domainOnly
	}

	//IP+域名
	if len(t.FilterDomainRuler) != 0 && len(t.FilterIpRuler) != 0 {
		if t.IsMatchResolveIP {
			return resolveIP + domainOnly
		} else {
			return requestIP + domainOnly
		}
	}

	return 0
}

func newTasks() *Tasks {
	tasks := readConf("./config.yaml")

	if ok, err := tasks.configValid(); !ok {
		fmt.Printf("配置文件校验错误: %s\n", err.Error())
		os.Exit(1)
	}

	tasks.RunStatus.StartTime = time.Now().Format("2006-01-02 15:04:05")

	tasks.NewFilePath = make(chan string, tasks.AnalyzeThreads*200)
	tasks.hostIP = GetIPAddress(tasks.EthName)
	tasks.TempResultMap = make(map[string]*bytes.Buffer)
	tasks.RunStatus.TaskMatchDetails = make(map[string]int)

	for i, v := range strings.Split(tasks.InputFormat, ",") {
		switch v {
		case "1":
			tasks.RequestIPIndex = i
		case "3":
			tasks.DNSServerIndex = i
		case "7":
			tasks.RequestTypeIndex = i
		case "6":
			tasks.DomainIndex = i
		case "14":
			tasks.RCodeIndex = i
		case "15":
			tasks.ResultIndex = i
		default:

		}
	}

	//初始化rocksdb
	if tasks.CountDomainMode {
		//var err error
		//tasks.DomainCounter, err = NewDomainCounter("./stats_db", tasks.AnalyzeThreads) // 使用绝对路径
		//if err != nil {
		//	log.Fatalf("Failed to initialize counter: %v", err)
		//}

		tasks.DomainCounter = &DomainCounter{
			countsFile: "output.txt",
			counts:     make(chan string, 10000),
			cond:       sync.NewCond(&sync.Mutex{}),
			merged:     make(map[string]int),
			flushReq:   make(chan struct{}, 1),
			flushDone:  make(chan struct{}),
			flushing:   false,
		}

		// 创建调度器
		c := cron.New()
		_, err := c.AddFunc(tasks.CountDomainCron, func() {
			log.Println("[Cron] Starting exec CronTask: FlushToFile")
			if err := tasks.DomainCounter.write(); err != nil {
				log.Printf("Failed to flush to file: %v", err)
			}
			log.Println("[Cron] Completed FlushToFile to ", tasks.DomainCounter.countsFile)
		})
		if err != nil {
			log.Fatalf("Failed to schedule job: %v", err)
		}
		c.Start()
		//defer c.Stop()

	}

	for taskName, task := range tasks.TaskInfos {
		task.TaskID++

		if !task.Enable {
			delete(tasks.TaskInfos, taskName)
			continue
		}

		task.FilterTag = task.getTaskType()

		//正对DRMS数据源输出集团日志的特殊处理，未来会删除
		if !(task.OutputFormatString == "jituan") && !(task.OutputFormatString == "full") {
			task.OutputFormat = transferFormat(tasks.InputFormat, task.OutputFormatString)
		}

		task.outPreFileName = make(map[int]*fileInfo)

		if task.ForceDomainMode {
			if !strings.HasPrefix(task.ForceDomainSql, "select") {
				log.Fatalf("only support select operation")
			}
			task.queryAndWriteDomains(task.ForceDomainList)
		}
		task.NewMatchRule(task.FilterIpRuler, task.FilterDomainRuler)

		if task.FileMaxSizeString != "" {
			unit := task.FileMaxSizeString[len(task.FileMaxSizeString)-1:]
			value, _ := strconv.Atoi(task.FileMaxSizeString[:len(task.FileMaxSizeString)-1])
			switch strings.ToLower(unit) {
			case "g":
				task.FileMaxSize = value * 1 << 30
			case "m":
				task.FileMaxSize = value * 1 << 20
			case "k":
				task.FileMaxSize = value * 1 << 10
			}
		} else {

			//默认200M
			task.FileMaxSize = 200 * 1 << 20
		}

		if task.FileMaxTime == 0*time.Second {
			task.FileMaxTime = 99999 * time.Hour
		}

		if runtime.GOOS == "darwin" {
			task.IsGzip = false
		}

		//初始化sftp
		if task.Upload.IsUpload {
			config := UploadConfig{
				Servers: []SFTPConfig{
					{
						Host:     task.Upload.SFTPHost,
						Port:     task.Upload.SFTPPort,
						Username: task.Upload.SFTPUser,
						Password: decString(task.Upload.SFTPPass),
						Path:     task.Upload.SFTPPath,
					},
				},
				MaxRetries:    task.Upload.MaxRetries,
				RetryDelay:    task.Upload.RetryDelay,
				RateLimitKBps: task.Upload.RateLimitKBps, // 1MB/s
				Timeout:       task.Upload.Timeout,
			}

			var err error

			if task.Upload.sftpUploadManager, err = NewUploadManager(config); err != nil {
				log.Fatalf(err.Error())
			}

		}

		tasks.TaskInfos[taskName] = task

	}

	return tasks
}

func readConf(filename string) *Tasks {
	configBytes, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}
	// 创建配置文件的结构体
	var tasks Tasks
	// 调用 Unmarshall 去解码文件内容
	// 注意要穿配置结构体的指针进去
	err = yaml.Unmarshal(configBytes, &tasks)
	if err != nil {
		log.Fatalln(err)
	}
	return &tasks

}
