package main

import (
	"bytes"
	_ "net/http/pprof" // pprof包的init方法会注册5个uri pattern方法到runtime包中
	"sync"
	"time"
)

const (
	valid   = true
	invalid = false
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
	RequestIPIndex   int //请求IP
	DNSServerIndex   int //DNS IP
	RequestTypeIndex int
	RCodeIndex       int //响应编码
	DomainIndex      int //请求域名
	ResultIndex      int //响应结果

}

type Tasks struct {
	EthName string `yaml:"eth_name"`
	hostIP  string

	LogType string `yaml:"log_type"`

	AnalyzeThreads int `yaml:"analyze_threads"`
	NewFilePath    chan string

	InputDir string `yaml:"input_dir"`

	InputFormat string               `yaml:"input_format"`
	BackupDir   string               `yaml:"backup_dir"`
	IsDelete    bool                 `yaml:"is_delete"`
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

	CountDomainMode bool   `yaml:"count_domain_mode"`
	CountDomainCron string `yaml:"count_domain_cron"`
	DomainCounter   *DomainCounter
}

// DomainCounter 管理跨线程的域计数
type DomainCounter struct {
	counts     chan string    // 通道用于传递域名
	countsFile string         // 计数存储的文件路径
	merged     map[string]int // 共享 map 用于存储计数
	flushReq   chan struct{}  // 信号通道，通知收集器暂停
	flushDone  chan struct{}  // 信号通道，通知收集器暂停完成
	flushing   bool           // 标记是否正在执行 write
	cond       *sync.Cond     // 条件变量，同步写进程和收集器
	mu         sync.Mutex     // 保护 merged map
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

	Upload uploadInfo `yaml:"upload"`

	UmpMysqlHost      string        `yaml:"ump_mysql_host"`
	UmpMysqlPort      int           `yaml:"ump_mysql_port"`
	UmpMysqlUser      string        `yaml:"ump_mysql_user"`
	UmpMysqlPass      string        `yaml:"ump_mysql_pass"`
	DbName            string        `yaml:"db_name"`
	ForceDomainSql    string        `yaml:"force_domain_sql"`
	ForceDomainMode   bool          `yaml:"force_domain_mode"`
	ForceDomainList   string        `yaml:"force_domain_list"`
	ForceDomainUpdate time.Duration `yaml:"force_domain_update"`

	//写意图声明,用于数据库更新锁
	writeFlag int32 // 0: no writer, 1: writer wants lock)
	writeLock sync.RWMutex
}

type uploadInfo struct {
	sftpUploadManager *UploadManager
	IsUpload          bool          `yaml:"is_upload"`
	SFTPHost          string        `yaml:"sftp_host"`
	SFTPPort          int           `yaml:"sftp_port"`
	SFTPUser          string        `yaml:"sftp_user"`
	SFTPPass          string        `yaml:"sftp_pass"`
	SFTPPath          string        `yaml:"sftp_path"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
	RateLimitKBps     int           `yaml:"rate_limit_kbps"`
	Timeout           time.Duration `yaml:"timeout"`
}

func main() {

	//if runtime.GOOS == "darwin" {
	//	os.RemoveAll("/Users/leon/Documents/02-code/go/src/github.com/LeonRhapsody/DNSLogFilter/data")
	//}

	Execute()
	//go http.ListenAndServe("127.0.0.1:8080", nil)

}
