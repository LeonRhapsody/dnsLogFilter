package main

import (
	"bufio"
	"bytes"
	"fmt"
	_ "net/http/pprof" // pprof包的init方法会注册5个uri pattern方法到runtime包中
	"os"
	"strings"
	"sync"
	"time"
)

type fileInfo struct {
	fileName   string
	CreateTime time.Time
}

type Tasks struct {
	EthName string `yaml:"eth_name"`
	hostIP  string

	AnalyzeThreads int `yaml:"analyze_threads"`
	FoundFilePath  chan string
	InputDir       string `yaml:"input_dir"`

	InputFormat string              `yaml:"input_format"`
	BackupDir   string              `yaml:"backup_dir"`
	TaskInfos   map[string]TaskInfo `yaml:"task_infos"`

	OnlineMode bool `yaml:"online_mode"`

	//analyze模块中，每个任务单独一个buffer，用于存储单个文件的分析结果
	TempResultMap map[string]*bytes.Buffer

	DomainNums map[int]map[string]int

	//执行输出域名清单的时间
	ExecutedHour int `yaml:"executed_hour"`

	IpTag     int
	DomainTag int
}

type TaskInfo struct {
	Enable bool `yaml:"enable"`

	TaskType string `yaml:"task_type"`

	FilterIpRuler     []string `yaml:"filter_ip_ruler"`
	FilterDomainRuler []string `yaml:"filter_domain_ruler"`

	IpFilterRuler     sync.Map
	DomainFilterRuler *TrieNode
	V6FilterRuler     *TrieNode

	outPreFileName map[int]*fileInfo

	//1:domain only 2:ip only 3:all
	FilterTag int

	OutputDir string `yaml:"output_dir"`

	OutputFormatString string `yaml:"output_format"`
	OutputFormat       []int

	IsGzip bool `yaml:"is_gzip"`

	FileMaxSizeString string `yaml:"file_max_size"`
	FileMaxSize       int

	FileMaxTime time.Duration `yaml:"file_max_time"`

	extend
}

type extend struct {
	CityCIDRs             map[string]string
	SuccessRateStatistics map[string]int
	TotalRateStatistics   map[string]int
	//文件生成间隔
	Interval time.Duration `yaml:"interval"`

	IsUpload   bool `yaml:"is_upload"`
	UploadFile chan string

	uploadMod string // sftp;ftp
	Host      string
	Port      string
	User      string
	Pass      string
	Path      string
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

	var ListMap sync.Map
	for _, file := range FilterListFile {
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
				}
			} else {

				ListMap.Store(scanner.Text(), 1)

			}

		}

		File.Close()
	}

	return ListMap

}

func main() {

	Run()
	//TestIP()
}
