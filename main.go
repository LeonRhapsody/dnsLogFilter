package main

import (
	"bytes"
	"fmt"
	"github.com/LeonRhapsody/DNSLogFilter/cmd"
	_ "net/http/pprof" // pprof包的init方法会注册5个uri pattern方法到runtime包中
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	valid   = true
	invalid = false
)

var (
	Commit    string
	GitLog    string
	BuildTime string
)

// 输出文件信息，用于判断截断条件
type fileInfo struct {
	fileName   string
	CreateTime time.Time
}

type RunStatus struct {
	StartTime        string         `json:"start_time"`
	AnalyzedFileNums int            `json:"analyzed_file_nums"`
	TaskMatchDetails map[string]int `json:"task_match_details"`
	statusLock       sync.Mutex
}

// logIndex 日志字段索引
type logIndex struct {
	RequestIPIndex int //请求IP
	DNSServerIndex int //DNS IP
	RCodeIndex     int //响应编码
	DomainIndex    int //请求域名
	ResultIndex    int //响应结果

}

type Tasks struct {
	EthName string `yaml:"eth_name"`
	hostIP  string

	AnalyzeThreads int `yaml:"analyze_threads"`
	FoundFilePath  chan string
	InputDir       string `yaml:"input_dir"`

	InputFormat string               `yaml:"input_format"`
	BackupDir   string               `yaml:"backup_dir"`
	TaskInfos   map[string]*TaskInfo `yaml:"task_infos"`

	OnlineMode bool `yaml:"online_mode"`
	adminMode  bool `yaml:"admin_mode"`

	//analyze模块中，每个任务单独一个buffer，用于存储单个文件的分析结果
	//此举是模拟了sync.pool方法，杜绝频繁申请内存的行为
	TempResultMap map[string]*bytes.Buffer

	//用于离线分析线程的主动终止逻辑
	wg sync.WaitGroup

	logIndex
	RunStatus

	//primaryDomainStatistics
	domainCounter       sync.Map
	DomainCounterEnable bool `yaml:"domain_counter_enable"`
	ExecutedHour        int  `yaml:"executed_hour"` //执行输出域名清单的时间
	lastExecutedDay     *time.Time

	//客户端IP

	//大一统规则池
	mapCache sync.Map
	//大一统规则池
	treeCache *TrieNode
}

type TaskInfo struct {
	//是否启用
	Enable bool `yaml:"enable"`

	TaskType string `yaml:"task_type"`
	TaskID   int

	//是否将IP过滤修改为解析IP
	IsMatchResolveIP bool `yaml:"is_match_resolve_ip"`

	//客户端IP
	FilterIpRuler []string `yaml:"filter_ip_ruler"`

	//ip过滤模式 1-v4 2-v6 3-v4+v6
	ipFilterMode int

	//请求域名
	FilterDomainRuler []string `yaml:"filter_domain_ruler"`

	//客户端IP
	IpFilterRuler *sync.Map

	IpFilterV6Ruler *TrieNode

	//泛域名
	DomainFilterRuler *TrieNode

	outPreFileName map[int]*fileInfo

	//1:domain only 2:ip only 3:all
	FilterTag int

	OutputDir      string `yaml:"output_dir"`
	OutputFileName string `yaml:"output_file_name"`

	OutputFormatString string `yaml:"output_format"`
	OutputFormat       []int

	IsGzip bool `yaml:"is_gzip"`

	FileMaxSizeString string `yaml:"file_max_size"`
	FileMaxSize       int
	FileMaxTime       time.Duration `yaml:"file_max_time"`
	taskMatchRule     *MatchRule

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

func main() {

	if runtime.GOOS == "darwin" {
		os.RemoveAll("/Users/leon/Documents/02-code/go/src/github.com/LeonRhapsody/DNSLogFilter/data")
	}

	cmd.Commit = Commit
	cmd.GitLog = GitLog
	cmd.BuildTime = BuildTime
	cmd.Execute()

	//go http.ListenAndServe("127.0.0.1:8080", nil)
	Run()

}
