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
	"sync/atomic"
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

func (r *MatchRule) domainMatch(domain string) bool {
	return r.domainTrie.Search(domain)
}

func (r *MatchRule) requestIPMatch(ip string) bool {

	switch r.ipFilterMode {
	case 0:
		return false
	case 1:
		_, OK := r.v4ListMap.Load(ip)
		return OK
	case 2:
		return r.v6Trie.V6Search(ip)
	case 3:
		if !strings.Contains(ip, ":") {
			_, OK := r.v4ListMap.Load(ip)
			return OK
		} else {
			return r.v6Trie.V6Search(ip)

		}
	default:
		return false
	}

}

func (r *MatchRule) resolveIPMatch(ips string) bool {
	switch r.ipFilterMode {
	case 0:
		return false
	case 1:
		for _, ip := range strings.Split(ips, ";") {
			_, OK := r.v4ListMap.Load(ip)
			if OK {
				return true
			}
		}
	case 2:
		for _, ip := range strings.Split(ips, ";") {

			if r.v6Trie.V6Search(ip) {
				return true
			}
		}
	case 3:
		if !strings.Contains(ips, ":") {
			for _, ip := range strings.Split(ips, ";") {
				_, OK := r.v4ListMap.Load(ip)
				if OK {
					return OK
				}
			}
		} else {
			for _, ip := range strings.Split(ips, ";") {

				if r.v6Trie.V6Search(ip) {
					return true
				}
			}

		}
	default:
		return false
	}

	return false

}

func (r *MatchRule) Match(IP string, domain string, result string, mode int) bool {
	//匹配规则：
	//01 仅域名
	//10 仅请求IP
	//20 仅解析IP

	//11 请求IP+域名
	//21 解析IP+域名

	switch mode {
	case 01:
		return r.domainMatch(domain)
	case 10:
		return r.requestIPMatch(IP)
	case 20:
		return r.resolveIPMatch(IP)
	case 11:
		return r.domainMatch(domain) && r.requestIPMatch(IP)
	case 21:
		return r.domainMatch(domain) && r.resolveIPMatch(IP)
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

func (T *Tasks) Filter(srcFileName string, taskId int, fileId int) {

	if T.IsDelete {
		defer T.deleteFile(srcFileName)
	} else {
		defer T.backupFile(srcFileName)
	}

	for _, task := range T.TaskInfos {
		//确认写意图，确保写的线程优先级更高，能够及时拿到锁
		for {
			if atomic.LoadInt32(&task.writeFlag) == 1 {
				time.Sleep(10 * time.Millisecond)
			} else {
				break
			}
		}
		task.writeLock.RLock()
		defer task.writeLock.RUnlock()
	}

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
		if len(srcLogArr) < 7 || srcLogArr[0] == "q" {
			continue
		}

		//统计每日主域名和访问数量
		if T.CountDomainMode {
			if srcLogArr[T.RequestTypeIndex] == "65" {
				T.DomainCounter.domainIncrement(srcLogArr[T.DomainIndex])
			}
		}

		for TaskName, task := range T.TaskInfos {

			if task.taskMatchRule.Match(srcLogArr[T.RequestIPIndex], srcLogArr[T.DomainIndex], srcLogArr[T.ResultIndex], task.FilterTag) {
				filterMatchCounter[TaskName]++

				//如果输出标记为full，不处理日志格式直接输出
				switch task.OutputFormatString {
				case "full":
					T.TempResultMap[TaskName+strconv.Itoa(taskId)].WriteString(scanner.Text() + "\n")

				default:
					T.TempResultMap[TaskName+strconv.Itoa(taskId)].WriteString(task.outFormat(srcLogArr))

				}

			}

		}

	}

	if err = scanner.Err(); err != nil {
		log.Println(err)
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
				if task.Upload.IsUpload {
					err = task.uploadFile(gzFile)
					if err != nil {
						log.Println("[Error Upload] failed: %v", err)
					} else {
						log.Printf("[Upload] %s to sftp %s successfully\n", gzFile, task.Upload.SFTPHost)
						T.deleteFile(gzFile)

					}
				}
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

}

func (T *Tasks) execTransfer() {
	// 为每个任务和线程初始化缓冲区
	for i := 0; i < T.AnalyzeThreads; i++ {
		for taskName := range T.TaskInfos {
			T.TempResultMap[taskName+strconv.Itoa(i)] = new(bytes.Buffer)
		}

		T.wg.Add(1)
		go func(id int) {
			defer T.wg.Done()

			log.Printf("初始化分析【%d】线程\n", id)
			var fileID int

			// 设置定时器，每0.2秒检查一次文件修改时间
			timer := time.NewTicker(100 * time.Millisecond)
			defer timer.Stop()

			for {

				select {
				case fileName := <-T.NewFilePath:
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
							if time.Since(modTime) > 200*time.Millisecond {
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

//
//func (T *Tasks) execTransfer() {
//
//	for i := 0; i < T.AnalyzeThreads; i++ {
//
//		for taskName, _ := range T.TaskInfos {
//			T.TempResultMap[taskName+strconv.Itoa(i)] = new(bytes.Buffer) // 或者 &bytes.Buffer{}
//		}
//
//		T.wg.Add(1)
//		go func(id int) {
//			defer T.wg.Done()
//
//			log.Printf("初始化分析【%d】线程\n", id)
//			var fileID int
//
//			// 设置定时器，每0.2秒检查一次文件修改时间
//			timer := time.NewTicker(100 * time.Millisecond)
//			defer timer.Stop()
//
//			for {
//
//				select {
//				case fileName := <-T.NewFilePath:
//					fileID++
//					if fileName == "done" {
//						log.Printf("thread %d recive done,quit \n", id)
//						return
//					}
//				outer:
//					for {
//						select {
//						case <-timer.C:
//							// 获取文件信息
//							fileInfo, err := os.Stat(fileName)
//							if err != nil {
//								fmt.Println("Error:", err)
//								return
//							}
//							modTime := fileInfo.ModTime()
//
//							// 检查文件修改时间是否变化
//							if time.Since(modTime) > 1*time.Second {
//								log.Println("[New] File write completed,start transfer: ", fileName)
//
//								T.Filter(fileName, id, fileID)
//
//								break outer
//							}
//						}
//
//					}
//
//				}
//			}
//		}(i)
//	}
//
//}
