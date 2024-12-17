package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (t *TaskInfo) outFormat(srcLogArr []string) string {
	var result strings.Builder

	recordA, record4a := "", ""
	// 处理记录类型
	if strings.Contains(srcLogArr[11], ":") {
		record4a = srcLogArr[11]
	} else {
		recordA = srcLogArr[11]
	}

	//特殊处理，不并入主干
	if t.OutputFormatString == "jituan" {
		result.WriteString(
			srcLogArr[4] + "|" +
				srcLogArr[7] + "|" +
				srcLogArr[1] + "|" +
				recordA + "|" +
				srcLogArr[9] + "|" +
				srcLogArr[8] + "|" +
				srcLogArr[10] + "|" +
				record4a + "|" +
				srcLogArr[2] + "|" +
				"0.00" + "|" +
				srcLogArr[5] + "|" +
				"320000\n")

		return result.String()
	}

	for _, i := range t.OutputFormat {
		//由于DRMS日志中没有区分A、4A日志，所以使用DRMS做数据源又像单独输出A、4A日志的话，需要做特殊处理
		switch i {
		case 10017:
			result.WriteString(recordA + "|")
		case 10018:
			result.WriteString(record4a + "|")
		default:
			result.WriteString(srcLogArr[i] + "|")

		}

	}
	return result.String() + "\n"

}

func (t *TaskInfo) domainMatch(domain string) bool {
	return t.DomainFilterRuler.Search(domain)
}

func (t *TaskInfo) requestIPMatch(ip string) bool {

	if strings.Contains(ip, ":") {
		return t.IpFilterV6Ruler.V6Search(ip)
	} else {
		_, OK := t.IpFilterRuler.Load(ip)
		return OK
	}

}

func (t *TaskInfo) resolveIPMatch(ips string) bool {
	for _, ip := range strings.Split(ips, ";") {
		_, OK := t.IpFilterRuler.Load(ip)
		return OK
	}
	return false

}

func (t *TaskInfo) Match(IP string, domain string, result string) bool {
	//匹配规则：
	//01 仅域名
	//10 仅请求IP
	//20 仅解析IP

	//11 请求IP+域名
	//21 解析IP+域名

	switch t.FilterTag {
	case 01:
		return t.domainMatch(domain)
	case 10:
		return t.requestIPMatch(IP)
	case 20:
		return t.resolveIPMatch(IP)
	case 11:
		return t.domainMatch(domain) && t.requestIPMatch(IP)
	case 21:
		return t.domainMatch(domain) && t.resolveIPMatch(IP)
	default:
		return false
	}
}

func (T *Tasks) genFileName(fileName string) string {

	if fileName == "" {
		return fmt.Sprintf("%s_%s_%s", "250", T.hostIP, time.Now().Format("20060102150405"))
	}

	str01 := strings.Split(fileName, "_")

	for i, str := range str01 {
		switch str {
		case "ip":
			str01[i] = T.hostIP
		case "time":
			str01[i] = time.Now().Format("20060102150405")
		}

	}
	return strings.Join(str01, "_")
}

func (T *Tasks) filterCounterInitialize() map[string]int {
	filterMatchCounter := make(map[string]int)
	for taskName, _ := range T.TaskInfos {
		filterMatchCounter[taskName] = 0
	}
	return filterMatchCounter
}

// 获取主域名
func getMainDomain(domain string) string {
	//// 使用net/url解析URL
	//parsedURL, err := url.Parse(domain)
	//if err != nil {
	//	return ""
	//}
	//
	//// 获取主机部分
	//host := parsedURL.Hostname()
	// 分割主机名
	//parts := strings.Split(host, ".")
	parts := strings.Split(domain, ".")

	// 根据主域名规则获取主域名
	if len(parts) < 2 {
		return domain // 只有一个部分
	}

	// 返回最后两个部分作为主域名
	return strings.Join(parts[len(parts)-2:], ".")
}

// 读取gz和普通文件
func (T *Tasks) readFile(fileName string) ([]byte, error) {
	if strings.HasSuffix(fileName, ".gz") {
		return UnGzipFile(fileName)
	}
	return os.ReadFile(fileName)
}

// 更新域名计数的函数
func (T *Tasks) updateDomainCounter(domain string) {
	if T.DomainCounterEnable {
		primaryDomain := getMainDomain(domain)
		//primaryDomain := domain
		val, ok := T.domainCounter.Load(primaryDomain)
		if !ok {
			T.domainCounter.Store(primaryDomain, 1)
		} else {
			T.domainCounter.Store(primaryDomain, val.(int)+1)
		}
	}
}

func (T *Tasks) Filter(srcFileName string, taskId int, fileId int) {
	defer T.backupFile(srcFileName)
	srcLogArr := make([]string, 12)
	nums := 0
	filterMatchCounter := T.filterCounterInitialize()

	// 读取文件内容
	TargetContent, err := T.readFile(srcFileName)
	if err != nil {
		log.Fatal(err)
		return
	}

	scanner := bufio.NewScanner(bytes.NewReader(TargetContent))
	TargetContent = nil

	start := time.Now()

	//遍历日志文件，匹配域名
	for scanner.Scan() {
		nums++
		copy(srcLogArr, strings.Split(scanner.Text(), "|"))

		//剔除异常日志
		if len(srcLogArr) < 7 {
			continue
		}

		//统计每日主域名和访问数量
		T.updateDomainCounter(srcLogArr[T.DomainIndex])

		for TaskName, task := range T.TaskInfos {

			if task.Match(srcLogArr[T.RequestIPIndex], srcLogArr[T.DomainIndex], srcLogArr[T.ResultIndex]) {
				filterMatchCounter[TaskName]++

				//如果输出标记为full，不处理日志格式直接输出
				if task.OutputFormatString == "full" {
					T.TempResultMap[TaskName+strconv.Itoa(taskId)].WriteString(scanner.Text() + "\n")
				} else {
					T.TempResultMap[TaskName+strconv.Itoa(taskId)].WriteString(task.outFormat(srcLogArr))
				}
			}

		}

	}

	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for TaskName, task := range T.TaskInfos {

		if _, err = os.Stat(task.OutputDir); os.IsNotExist(err) {
			// 目录不存在，创建目录
			err = os.MkdirAll(task.OutputDir, 0755)
			if err != nil {
				log.Fatal("创建目录失败:", err)
				return
			}
			log.Println(task.OutputDir, " output dir create successfully")
		}

		//记录当前写入文件的信息
		if task.outPreFileName[taskId] == nil {

			task.outPreFileName[taskId] = &fileInfo{
				fileName:   path.Join(task.OutputDir, fmt.Sprintf("%s_%d.gz.tmp", T.genFileName(task.OutputFileName), taskId)),
				CreateTime: time.Now(),
			}

		}

		currentFileInfo, err := os.Stat(task.outPreFileName[taskId].fileName)

		if err == nil {

			if currentFileInfo.Size() >= int64(task.FileMaxSize) || time.Since(task.outPreFileName[taskId].CreateTime) >= task.FileMaxTime {
				gzFile := strings.TrimSuffix(task.outPreFileName[taskId].fileName, ".tmp")
				log.Printf("[Rename File] %s to %s\n", task.outPreFileName[taskId].fileName, gzFile)
				os.Rename(task.outPreFileName[taskId].fileName, gzFile)
				task.outPreFileName[taskId].fileName = path.Join(task.OutputDir, fmt.Sprintf("%s_%s_%s_%d.gz.tmp", "250", T.hostIP, time.Now().Format("20060102150405"), taskId))
				task.outPreFileName[taskId].CreateTime = time.Now()
			}
		}

		if task.IsGzip {
			T.WriteGzLog(task.outPreFileName[taskId].fileName, T.TempResultMap[TaskName+strconv.Itoa(taskId)])
		} else {
			T.WriteLog(task.outPreFileName[taskId].fileName, T.TempResultMap[TaskName+strconv.Itoa(taskId)])
		}
		T.TempResultMap[TaskName+strconv.Itoa(taskId)].Reset()

	}

	times := time.Since(start)
	qps := int(float64(nums) / times.Seconds())

	var matchInfo string

	for taskName, matchNum := range filterMatchCounter {
		matchInfo = matchInfo + fmt.Sprintf("%s: match %d ,", taskName, matchNum)
	}
	T.statusLock.Lock()
	T.AnalyzedFileNums++
	T.statusLock.Unlock()

	log.Printf("[Analyze] [taskId threads: %d ,nums: %d] [filename: %s]-[record: %d], [end %s,During: %s, Qps: %d], [%s] \n", taskId, fileId, srcFileName, nums, time.Now().Format("2006-01-02 15:04:05"), times.String(), qps, matchInfo)

	//统计每日主域名和访问数量
	if T.DomainCounterEnable && time.Now().Hour() >= T.ExecutedHour && start.Day() != (T.lastExecutedDay).Day() {
		go sortByValueAndSaveMap(T.domainCounter, fmt.Sprintf("domain_%d.txt", time.Now().Day()))
		T.domainCounter = sync.Map{}
		*(T.lastExecutedDay) = time.Now()
	}
}

func (T *Tasks) execTransfer() {

	for i := 0; i < T.AnalyzeThreads; i++ {

		for taskName, _ := range T.TaskInfos {
			T.TempResultMap[taskName+strconv.Itoa(i)] = new(bytes.Buffer) // 或者 &bytes.Buffer{}
		}

		T.wg.Add(1)
		go func(id int) {
			defer T.wg.Done()

			log.Printf("初始化分析【%d】线程\n", id)
			var fileID int

			// 设置定时器，每0.2秒检查一次文件修改时间
			timer := time.NewTicker(200 * time.Millisecond)
			defer timer.Stop()
			for {

				select {
				case fileName := <-T.FoundFilePath:
					fileID++
					if fileName == "done" {
						log.Printf("thread %d recive done,quit \n", id)
						return
					}
				outer:
					for {
						select {
						case <-timer.C:
							// 获取文件信息
							fileInfo, err := os.Stat(fileName)
							if err != nil {
								fmt.Println("Error:", err)
								return
							}
							modTime := fileInfo.ModTime()

							// 检查文件修改时间是否变化
							if time.Since(modTime) > 1*time.Second {
								log.Println("[New] File write completed,start transfer: ", fileName)

								T.Filter(fileName, id, fileID)

								break outer
							}
						}

					}

				}
			}
		}(i)
	}

}
