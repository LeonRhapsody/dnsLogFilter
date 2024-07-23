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
	"time"
)

func (t *TaskInfo) outFormat(srcLogArr []string) string {

	//特殊处理，不并入主干
	if t.OutputFormatString == "jituan" {

		var result strings.Builder

		var record_A, record_4A string

		if strings.Contains(srcLogArr[11], ":") {
			record_A = ""
			record_4A = srcLogArr[11]
		} else {
			record_4A = ""
			record_A = srcLogArr[11]
		}

		result.WriteString(
			srcLogArr[4] + "|" +
				srcLogArr[7] + "|" +
				srcLogArr[1] + "|" +
				record_A + "|" +
				srcLogArr[9] + "|" +
				srcLogArr[8] + "|" +
				srcLogArr[10] + "|" +
				record_4A + "|" +
				srcLogArr[2] + "|" +
				"0.00" + "|" +
				srcLogArr[5] + "|" +
				"320000")

		return result.String() + "\n"
	}

	var result strings.Builder
	var record_A, record_4A string
	if strings.Contains(srcLogArr[11], ":") {
		record_A = ""
		record_4A = srcLogArr[11]
	} else {
		record_4A = ""
		record_A = srcLogArr[11]
	}
	for _, i := range t.OutputFormat {
		//由于DRMS日志中没有区分A、4A日志，所以使用DRMS做数据源又像单独输出A、4A日志的话，需要做特殊处理
		switch i {
		case 10017:
			result.WriteString(record_A + "|")
		case 10018:
			result.WriteString(record_4A + "|")
		default:
			result.WriteString(srcLogArr[i] + "|")

		}

	}
	return result.String() + "\n"

}

func (t *TaskInfo) WildcardExactMatch(domain string) bool {
	_, OK := t.ExactDomainFilterRuler.Load(domain)
	return OK
}

func (t *TaskInfo) ExactDomainMatch(domain string) bool {
	return t.DomainFilterRuler.Search(domain)
}

func (t *TaskInfo) ExactIPMatch(ip string) bool {
	_, OK := t.IpFilterRuler.Load(ip)
	return OK
}

func (t *TaskInfo) Match(target string) bool {
	//匹配规则：
	//0 仅IP
	//1 仅精确域名
	//2 仅泛域名
	//3 精确域名+IP
	//4 泛域名+IP

	switch t.FilterTag {
	case 0:
		return t.ExactIPMatch(target)
	case 1:
		return t.ExactDomainMatch(target)
	case 2:
		return t.WildcardExactMatch(target)
	case 3:
		return t.ExactDomainMatch(target) && t.ExactIPMatch(target)
	case 4:
		return t.WildcardExactMatch(target) && t.ExactIPMatch(target)
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

func (T *Tasks) Filter(srcFileName string, taskId int, fileId int, domainNums map[string]int, lastExecutedDay *time.Time) {
	defer T.backupFile(srcFileName)
	nums := 0
	filterMatchCounter := T.filterCounterInitialize()

	var TargetContent []byte

	//判断是gz文件还是普通文本文件
	if strings.HasSuffix(srcFileName, ".gz") {
		buffer := bytes.NewBuffer(nil)
		err := UngzipToBuffer(srcFileName, buffer)
		if err != nil {
			log.Println(err)
		}
		TargetContent = buffer.Bytes()
		buffer.Reset()
		//buffer = *bytes.NewBuffer(nil)

	} else {
		TargetLogData, err := os.ReadFile(srcFileName)
		if err != nil {
			log.Fatal(err)
		}
		TargetContent = TargetLogData

	}

	scanner := bufio.NewScanner(strings.NewReader(string(TargetContent)))
	TargetContent = nil

	start := time.Now()

	//遍历日志文件，匹配域名
	for scanner.Scan() {
		nums++
		srcLogArr := strings.Split(scanner.Text(), "|")

		//剔除异常日志
		if len(srcLogArr) != len(T.InputFormat) {
			continue
		}

		for TaskName, task := range T.TaskInfos {

			if task.Match(srcLogArr[T.DomainTag]) {
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

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

	for TaskName, task := range T.TaskInfos {

		if _, err := os.Stat(task.OutputDir); os.IsNotExist(err) {
			// 目录不存在，创建目录
			err := os.MkdirAll(task.OutputDir, 0755)
			if err != nil {
				fmt.Println("创建目录失败:", err)
				return
			}
			fmt.Println(task.OutputDir, " output dir create successfully")
		}

		if task.outPreFileName[taskId] == nil {
			fileInfo := fileInfo{
				fileName:   path.Join(task.OutputDir, fmt.Sprintf("%s_%d.gz.tmp", T.genFileName(task.OutputFileName), taskId)),
				CreateTime: time.Now(),
			}
			task.outPreFileName[taskId] = &fileInfo

		}

		currentFileInfo, err := os.Stat(task.outPreFileName[taskId].fileName)

		if err == nil {

			if currentFileInfo.Size() >= int64(task.FileMaxSize) || time.Since(task.outPreFileName[taskId].CreateTime) >= task.FileMaxTime {
				gzFile := strings.TrimSuffix(task.outPreFileName[taskId].fileName, ".tmp")
				fmt.Printf("[Rename File] %s to %s\n", task.outPreFileName[taskId].fileName, gzFile)
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

	fmt.Printf("[Analyze] [taskId threads: %d ,nums: %d] [filename: %s]-[record: %d], [end %s,During: %s, Qps: %d], [%s] [%d]\n", taskId, fileId, srcFileName, nums, time.Now().Format("2006-01-02 15:04:05"), times.String(), qps, matchInfo, len(domainNums))

	//if time.Now().Hour() == T.ExecutedHour && start.Day() != (*lastExecutedDay).Day() {
	//	saveMap(domainNums, fmt.Sprintf("domain_%d.txt.%d", time.Now().Day(), taskId))
	//	domainNums = make(map[string]int)
	//	*lastExecutedDay = time.Now()
	//}
}

func (T *Tasks) execTransfer() {
	//var wg sync.WaitGroup

	for i := 0; i < T.AnalyzeThreads; i++ {

		for taskName, _ := range T.TaskInfos {
			T.TempResultMap[taskName+strconv.Itoa(i)] = new(bytes.Buffer) // 或者 &bytes.Buffer{}
		}

		//wg.Add(1)
		go func(id int) {
			//defer wg.Done()

			//给每个线程分配一个计数器和map
			domainNums := make(map[string]int)
			var lastExecutedDay time.Time

			fmt.Printf("初始化分析【%d】线程\n", id)
			var fileID int

			// 设置定时器，每0.2秒检查一次文件修改时间
			timer := time.NewTicker(200 * time.Millisecond)
			defer timer.Stop()
			for {

				select {
				case fileName := <-T.FoundFilePath:
					fileID++
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
								fmt.Println("[New] File write completed,start transfer: ", fileName)

								T.Filter(fileName, id, fileID, domainNums, &lastExecutedDay)

								break outer
							}
						}

					}

				}
			}
		}(i)
	}
	//wg.Wait()
	//fmt.Println("All files have been processed.")
}
