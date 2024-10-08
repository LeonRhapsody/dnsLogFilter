package main

import (
	"bufio"
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
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

func taskListRead(FilterListFile string, ListMap map[string]int) {

	File, err := os.Open(FilterListFile)
	if err != nil {
		fmt.Println(err)
	}

	scanner := bufio.NewScanner(File)
	for scanner.Scan() {
		ListMap[scanner.Text()] = 1
	}

	File.Close()

}

func IPListToSyncMap(FilterListFile []string) sync.Map {
	counter := 0
	var files string
	var ListMap sync.Map
	for _, file := range FilterListFile {
		files = files + "/" + file

		File, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
		}

		scanner := bufio.NewScanner(File)
		for scanner.Scan() {
			//ListMap.Store(scanner.Text(), 1)
			if !strings.Contains(scanner.Text(), ":") {
				ips := parseIPFormat(scanner.Text())
				for _, ip := range ips {
					ListMap.Store(ip, 1)
					counter++
				}
			} else {

				ListMap.Store(scanner.Text(), 1)
				counter++

			}

		}

		File.Close()
	}
	fmt.Printf("%s read %d domain rules\n", files, counter)

	return ListMap

}
func DomainListToSyncMap(FilterListFile []string) sync.Map {
	counter := 0
	var files string
	var ListMap sync.Map
	for _, file := range FilterListFile {
		files = files + "/" + file
		File, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
		}

		scanner := bufio.NewScanner(File)
		for scanner.Scan() {
			ListMap.Store(scanner.Text(), 1)
			counter++

		}

		File.Close()
	}

	fmt.Printf("%s read %d domain rules\n", files, counter)
	return ListMap

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
	yaml.Unmarshal(configBytes, &tasks)
	err = yaml.Unmarshal(configBytes, &tasks)
	if err != nil {
		log.Fatalln(err)
	}

	tasks.FoundFilePath = make(chan string, 10)
	tasks.hostIP = GetIPAddress(tasks.EthName)
	tasks.TempResultMap = make(map[string]*bytes.Buffer)
	tasks.DomainNums = make(map[int]map[string]int)

	for i := 0; i < 10; i++ {
		tasks.DomainNums[i] = make(map[string]int)
	}

	for i, v := range strings.Split(tasks.InputFormat, ",") {
		if v == "6" {
			tasks.DomainTag = i
		}
		if v == "1" {
			tasks.IpTag = i
		}
	}

	for taskName, task := range tasks.TaskInfos {

		if !task.Enable {
			delete(tasks.TaskInfos, taskName)
			continue
		}

		//正对DRMS数据源输出集团日志的特殊处理，未来会删除
		if !(task.OutputFormatString == "jituan") && !(task.OutputFormatString == "full") {
			task.OutputFormat = transferFormat(tasks.InputFormat, task.OutputFormatString)
		}

		task.CityCIDRs = make(map[string]string)
		task.TotalRateStatistics = make(map[string]int)
		task.SuccessRateStatistics = make(map[string]int)
		task.outPreFileName = make(map[int]*fileInfo)

		if len(task.FilterIpRuler) != 0 {
			task.IpFilterRuler = IPListToSyncMap(task.FilterIpRuler)
		}
		if task.DomainExactMatch {
			task.ExactDomainFilterRuler = DomainListToSyncMap(task.FilterDomainRuler)
		} else {
			task.DomainFilterRuler = DomainListToTree(task.FilterDomainRuler)

		}

		//匹配规则：
		//0 仅IP
		//1 仅精确域名
		//2 仅泛域名
		//3 精确域名+IP
		//4 泛域名+IP

		if len(task.FilterIpRuler) != 0 {
			task.FilterTag = 0
		}

		if len(task.FilterDomainRuler) != 0 {
			if task.DomainExactMatch {
				task.FilterTag = 1
			} else {
				task.FilterTag = 2
			}
		}

		if len(task.FilterIpRuler) != 0 && len(task.FilterDomainRuler) != 0 {
			if task.DomainExactMatch {
				task.FilterTag = 3
			} else {
				task.FilterTag = 4
			}
		}

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

			task.FileMaxSize = 200 * 1 << 20
		}
		if task.FileMaxTime == 0*time.Second {
			task.FileMaxTime = 99999 * time.Hour
		}

		if runtime.GOOS == "darwin" {
			task.IsGzip = false
		}

		tasks.TaskInfos[taskName] = task

	}

	return &tasks

}
