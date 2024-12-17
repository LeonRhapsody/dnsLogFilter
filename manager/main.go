package manager

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ConfigStruct struct {
	jsonData
	IPFiles     []string
	DomainFiles []string
}

type jsonData struct {
	EthName        string              `yaml:"eth_name" json:"eth_name"`
	AnalyzeThreads int                 `yaml:"analyze_threads" json:"analyze_threads,string"`
	InputDir       string              `yaml:"input_dir" json:"input_dir"`
	InputFormat    string              `yaml:"input_format" json:"input_format"`
	BackupDir      string              `yaml:"backup_dir" json:"backup_dir"`
	TaskInfos      map[string]TaskInfo `yaml:"task_infos" json:"task_infos"`
	OnlineMode     bool                `yaml:"online_mode" json:"online_mode,string"`
}

type TaskInfo struct {
	Enable             bool          `yaml:"enable" json:"enable,string"`
	FilterIpRuler      []string      `yaml:"filter_ip_ruler" json:"filter_ip_ruler"`
	FilterDomainRuler  []string      `yaml:"filter_domain_ruler" json:"filter_domain_ruler"`
	OutputDir          string        `yaml:"output_dir" json:"output_dir"`
	OutputFormatString string        `yaml:"output_format" json:"output_format"`
	IsGzip             bool          `yaml:"is_gzip" json:"is_gzip,string"`
	FileMaxSizeString  string        `yaml:"file_max_size" json:"file_max_size"`
	FileMaxTime        time.Duration `yaml:"file_max_time" json:"file_max_time,string"`
}

var (
	ipFiles     []string
	domainFiles []string
	configData  map[string]interface{}
	mutex       sync.Mutex
)

func parseJSON(input map[string]string) (jsonData, error) {
	var result jsonData
	taskInfos := make(map[string]*TaskInfo)

	result.EthName = input["eth_name"]
	result.AnalyzeThreads = 1
	result.InputDir = input["input_dir"]
	result.InputFormat = input["input_format"]
	result.BackupDir = input["backup_dir"]
	result.OnlineMode = input["online_mode"] == "true"

	for key, value := range input {
		if strings.HasPrefix(key, "taskInfo_") {
			taskName := strings.Split(key, "_")[1]

			if _, exists := taskInfos[taskName]; !exists {
				taskInfos[taskName] = &TaskInfo{}
			}

			switch {
			case key == "taskInfo_"+taskName+"_Enable":
				taskInfos[taskName].Enable = value == "true"
			case key == "taskInfo_"+taskName+"_FilterIpRuler":
				taskInfos[taskName].FilterIpRuler = []string{value}
			case key == "taskInfo_"+taskName+"_FilterDomainRuler":
				taskInfos[taskName].FilterDomainRuler = []string{value}
			case key == "taskInfo_"+taskName+"_outputDir":
				taskInfos[taskName].OutputDir = value
			case key == "taskInfo_"+taskName+"_outputFormatString":
				taskInfos[taskName].OutputFormatString = value
			case key == "taskInfo_"+taskName+"_IsGzip":
				taskInfos[taskName].IsGzip = value == "true"
			case key == "taskInfo_"+taskName+"_FileMaxSizeString":
				taskInfos[taskName].FileMaxSizeString = value
			case key == "taskInfo_"+taskName+"_FileMaxTime":
				duration, err := time.ParseDuration(value)
				if err == nil {
					taskInfos[taskName].FileMaxTime = duration
				}
			}
		}
	}

	result.TaskInfos = make(map[string]TaskInfo)
	for taskName, taskInfo := range taskInfos {
		result.TaskInfos[taskName] = *taskInfo
	}
	return result, nil
}

func NewRouter(bindIP, bindPort string) *GinConfig {
	gin.SetMode(gin.ReleaseMode)

	return &GinConfig{
		bindIP:   bindIP,
		bindPort: bindPort,
		Router:   gin.Default(),
	}
}

type GinConfig struct {
	bindIP   string
	bindPort string
	Router   *gin.Engine
}

func (G *GinConfig) AddRouter() {
	loadFiles()
	loadConfig()

	G.Router.GET("/", homePage)
	G.Router.GET("/show", showFileContent)
	G.Router.POST("/save", saveFile)
	G.Router.DELETE("/delete", deleteFile)
	G.Router.GET("/config", showConfig)
	G.Router.POST("/updateConfig", updateConfig)
}

func (G *GinConfig) Run() {
	G.Router.Run(G.bindIP + ":" + G.bindPort)
}

func loadFiles() {
	mutex.Lock()
	defer mutex.Unlock()

	ipFiles = nil
	domainFiles = nil

	files, err := os.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".list" {
			ipFiles = append(ipFiles, file.Name())
		} else if filepath.Ext(file.Name()) == ".list2" {
			domainFiles = append(domainFiles, file.Name())
		}
	}
}

func loadConfig() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(data, &configData)
	if err != nil {
		panic(err)
	}
}

func homePage(c *gin.Context) {
	tmpl := `<!DOCTYPE html>
<html lang="zh">
<head>
	<meta charset="UTF-8">
	<title>配置管理系统</title>
	<link href="https://fonts.googleapis.com/css2?family=Roboto:wght@400;500&display=swap" rel="stylesheet">
	<script>
		function showContent(file) {
			if (file) {
				fetch("/show?file=" + file)
					.then(response => response.text())
					.then(data => {
						document.getElementById("outputDisplay").innerHTML = "<h2>" + file + "</h2><pre>" + data + "</pre>";
					});
				clearOutput(); 
				document.getElementById("addForm").style.display = "none";
			}
		}

		function showConfig() {
			fetch("/config")
				.then(response => response.text())
				.then(data => {
					document.getElementById("outputDisplay").innerHTML = data;
				});
			clearOutput(); 
			document.getElementById("addForm").style.display = "none"; 
		}

		function saveFile() {
			const name = document.getElementById("fileName").value;
			const content = document.getElementById("fileContentInput").value;
			if (name && content) {
				fetch("/save", {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify({ name: name, content: content })
				}).then(response => {
					if (response.ok) {
						alert("保存成功！");
						document.getElementById("fileName").value = '';
						document.getElementById("fileContentInput").value = '';
						loadFiles();
						clearOutput(); 
					} else {
						alert("保存失败！");
					}
				});
			} else {
				alert("请填写文件名称和内容！");
			}
		}

		function deleteFile(file) {
			if (confirm("您确定要删除这个文件吗？")) {
				fetch("/delete?file=" + file, {
					method: "DELETE"
				}).then(response => {
					if (response.ok) {
						alert("删除成功！");
						loadFiles();
						clearOutput(); 
					} else {
						alert("删除失败！");
					}
				});
			}
		}

		function updateConfig() {
			const formData = new FormData(document.getElementById("configForm"));
			const jsonData = {};
			formData.forEach((value, key) => {
				jsonData[key] = value;
			});

			fetch("/updateConfig", {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify(jsonData)
			})
			.then(response => {
				if (response.ok) {
					alert("配置已保存！");
					return response.text();
				}
				throw new Error("网络错误");
			})
			.then(data => {
				console.log(data);
			})
			.catch(error => {
				console.error("错误:", error);
			});
		}

		function loadFiles() {
			fetch("/")
				.then(response => response.json())
				.then(data => {
					const ipList = document.getElementById("ipFileList");
					const domainList = document.getElementById("domainFileList");
					ipList.innerHTML = '';
					domainList.innerHTML = '';

					data.ipFiles.forEach(file => {
						ipList.innerHTML += '<div class="menu-item" onclick="showContent(\'' + file + '\')">' +
							'<span class="file-name">' + file + '</span>' + 
							'<button class="delete-btn" onclick="deleteFile(\'' + file + '\'); event.stopPropagation();">删除</button>' + 
							'</div>';
					});
					data.domainFiles.forEach(file => {
						domainList.innerHTML += '<div class="menu-item" onclick="showContent(\'' + file + '\')">' + 
							'<span class="file-name">' + file + '</span>' +
							'<button class="delete-btn" onclick="deleteFile(\'' + file + '\'); event.stopPropagation();">删除</button>' + 
							'</div>';
					});
				});
		}

		function showAddForm(type) {
			clearOutput(); 
			document.getElementById("addForm").style.display = "block";
			document.getElementById("fileName").value = '';
			document.getElementById("fileContentInput").value = '';
		}

		function clearOutput() {
			document.getElementById("outputDisplay").innerHTML = "";
		}

		function toggleSubMenu(id) {
			var submenu = document.getElementById(id);
			submenu.style.display = submenu.style.display === "none" || submenu.style.display === "" ? "block" : "none";
		}
	</script>
	<style>
		body {
			font-family: 'Roboto', sans-serif;
			background-color: #f0f4f8;
			color: #333;
			margin: 0;
			padding: 0;
		}
		.container {
			display: flex;
			height: 100vh;
		}
		.sidebar {
			width: 200px;
			background-color: #2c3e50;
			color: white;
			padding: 20px;
			box-shadow: 2px 0 5px rgba(0, 0, 0, 0.1);
		}
		.sidebar h3, .sidebar h4 {
			cursor: pointer;
			margin: 10px 0;
			transition: color 0.3s;
		}
		.sidebar h3:hover, .sidebar h4:hover {
			color: #3498db;
		}
		.content {
			flex: 1;
			padding: 20px;
			background-color: #ecf0f1;
			overflow-y: auto;
		}
		.submenu {
			display: none;
			margin-left: 10px;
			padding-left: 10px;
			border-left: 2px solid #3498db;
		}
		.menu-item {
			display: flex;
			justify-content: space-between;
			align-items: center;
			cursor: pointer;
			background-color: #bdc3c7;
			margin: 5px 0;
			padding: 10px;
			border-radius: 5px;
			transition: background-color 0.3s;
		}
		.file-name {
			flex-grow: 1;
		}
		.menu-item:hover {
			background-color: #95a5a6;
		}
		.delete-btn {
			padding: 5px 8px;
			border: none;
			background-color: #e74c3c;
			color: white;
			border-radius: 3px;
			font-size: 12px;
			cursor: pointer;
			transition: background-color 0.3s;
			margin-left: 10px;
		}
		.delete-btn:hover {
			background-color: #c0392b;
		}
		#fileContent h2 {
			margin: 0 0 20px;
		}
		#addForm {
			background-color: white;
			border-radius: 5px;
			padding: 20px;
			box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
			display: none;
		}
		#fileName, #fileContentInput, input[type="text"], select {
			width: calc(100% - 22px);
			padding: 10px;
			border: 1px solid #bdc3c7;
			border-radius: 5px;
			margin-bottom: 10px;
			box-sizing: border-box;
		}
		button {
			padding: 10px 15px;
			border: none;
			background-color: #3498db;
			color: white;
			border-radius: 5px;
			cursor: pointer;
			transition: background-color 0.3s;
			width: 100%;
			box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2);
		}
		button:hover {
			background-color: #2980b9;
		}
		#outputDisplay {
			background-color: white;
			border-radius: 5px;
			padding: 20px;
			box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
			margin-top: 20px;
		}
	</style>
</head>
<body>
<div class="container">
	<div class="sidebar">
		<h3 onclick="toggleSubMenu('ipSubmenu')">IP清单配置</h3>
		<div id="ipSubmenu" class="submenu">
			<h4 onclick="toggleSubMenu('viewIP')">查看清单</h4>
			<div id="viewIP" class="submenu">
				<div id="ipFileList">
					{{range .IPFiles}}<div class="menu-item" onclick="showContent('{{.}}')"><span class="file-name">{{.}}</span><button class="delete-btn" onclick="deleteFile('{{.}}'); event.stopPropagation();">删除</button></div>{{end}}
				</div>
			</div>
			<h4 onclick="showAddForm('ip')">添加清单</h4>
		</div>
		<h3 onclick="toggleSubMenu('domainSubmenu')">域名清单配置</h3>
		<div id="domainSubmenu" class="submenu">
			<h4 onclick="toggleSubMenu('viewDomain')">查看清单</h4>
			<div id="viewDomain" class="submenu">
				<div id="domainFileList">
					{{range .DomainFiles}}<div class="menu-item" onclick="showContent('{{.}}')"><span class="file-name">{{.}}</span><button class="delete-btn" onclick="deleteFile('{{.}}'); event.stopPropagation();">删除</button></div>{{end}}
				</div>
			</div>
			<h4 onclick="showAddForm('domain')">添加清单</h4>
		</div>
		<h3 onclick="showConfig()">任务配置</h3>
	</div>
	<div class="content">
		<div id="outputDisplay"></div>
		<div id="addForm">
			<h2>添加清单</h2>
			<input type="text" id="fileName" placeholder="文件名">
			<textarea id="fileContentInput" rows="10" placeholder="文件内容"></textarea>
			<button onclick="saveFile()">保存</button>
		</div>
	</div>
</div>
</body>
</html>

	`
	c.Header("Content-Type", "text/html")
	c.Status(http.StatusOK)
	t := template.Must(template.New("index").Parse(tmpl))
	t.Execute(c.Writer, struct {
		IPFiles     []string
		DomainFiles []string
	}{ipFiles, domainFiles})
}

func showFileContent(c *gin.Context) {
	file := c.Query("file")
	if file == "" {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	content, err := os.ReadFile(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "无法读取文件")
		return
	}

	c.String(http.StatusOK, string(content))
}

func saveFile(c *gin.Context) {
	var data struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.String(http.StatusBadRequest, "请求格式错误")
		return
	}

	err := os.WriteFile(data.Name, []byte(data.Content), 0644)
	if err != nil {
		c.String(http.StatusInternalServerError, "无法保存文件")
		return
	}
	loadFiles() // 重新加载文件
	c.String(http.StatusOK, "文件保存成功")
}

func showConfig(c *gin.Context) {

	tmpl := `
<h2>运行主配置</h2>
<form id="configForm">
    <label for="ethName">eth_name:</label>
    <input type="text" id="ethName" name="eth_name" value="{{.EthName}}"><br>

    <label for="analyzeThreads">analyze_threads:</label>
    <input type="text" id="analyzeThreads" name="analyze_threads" value="{{.AnalyzeThreads}}"><br>

    <label for="inputDir">input_dir:</label>
    <input type="text" id="inputDir" name="input_dir" value="{{.InputDir}}"><br>

    <label for="inputFormat">input_format:</label>
    <input type="text" id="inputFormat" name="input_format" value="{{.InputFormat}}"><br>

    <label for="backupDir">backup_dir:</label>
    <input type="text" id="backupDir" name="backup_dir" value="{{.BackupDir}}"><br>

    <label for="onlineMode">online_mode:</label>
    <select id="onlineMode" name="online_mode">
        <option value="true" {{if .OnlineMode}}selected{{end}}>在线模式</option>
        <option value="false" {{if not .OnlineMode}}selected{{end}}>离线模式</option>
    </select><br>

    <h3>task_infos</h3>
    {{range $task, $info := .TaskInfos}}
    <div style="margin-left: 20px;">
        <h4>{{$task}}</h4>
        <div style="margin-left: 20px;">
            <label for="taskInfo_{{$task}}_Enable">enable:</label>
            <select id="taskInfo_{{$task}}_Enable" name="taskInfo_{{$task}}_Enable">
                <option value="true" {{if $info.Enable}}selected{{end}}>启用</option>
                <option value="false" {{if not $info.Enable}}selected{{end}}>关闭</option>
            </select><br>

			<label for="taskInfo_{{$task}}_FilterDomainRuler">filter_domain_ruler:</label>
			<select id="taskInfo_{{$task}}_FilterDomainRuler" name="taskInfo_{{$task}}_FilterDomainRuler">
				{{if .FilterDomainRuler}}
					{{range .FilterDomainRuler}}
						<option value="{{.}}">{{.}}</option>
					{{end}}
				{{else}}
					<option value="" selected>空</option>
				{{end}}
                {{range $.DomainFiles}}
                <option value="{{.}}">{{.}}</option>
                {{end}}
                <option value="">空</option>
			</select><br>

			<label for="taskInfo_{{$task}}_FilterIpRuler">filter_ip_ruler:</label>
			<select id="taskInfo_{{$task}}_FilterIpRuler" name="taskInfo_{{$task}}_FilterIpRuler">
				{{if .FilterIpRuler}}
					{{range .FilterIpRuler}}
						<option value="{{.}}">{{.}}</option>
					{{end}}
				{{else}}
					<option value="" selected>空</option>
				{{end}}
				{{range $.IPFiles}}
                <option value="{{.}}">{{.}}</option>
                {{end}}
                <option value="">空</option>
			</select><br>

 

            <label for="taskInfo_{{$task}}_OutputDir">output_dir:</label>
            <input type="text" id="taskInfo_{{$task}}_OutputDir" name="taskInfo_{{$task}}_OutputDir" value="{{$info.OutputDir}}"><br>

            <label for="taskInfo_{{$task}}_OutputFormatString">output_format:</label>
            <input type="text" id="taskInfo_{{$task}}_OutputFormatString" name="taskInfo_{{$task}}_OutputFormatString" value="{{$info.OutputFormatString}}"><br>

            <label for="taskInfo_{{$task}}_IsGzip">is_gzip:</label>
            <select id="taskInfo_{{$task}}_IsGzip" name="taskInfo_{{$task}}_IsGzip">
                <option value="true" {{if $info.IsGzip}}selected{{end}}>压缩</option>
                <option value="false" {{if not $info.IsGzip}}selected{{end}}>不压缩</option>
            </select><br>

            <label for="taskInfo_{{$task}}_FileMaxSizeString">file_max_size:</label>
            <input type="text" id="taskInfo_{{$task}}_FileMaxSizeString" name="taskInfo_{{$task}}_FileMaxSizeString" value="{{$info.FileMaxSizeString}}"><br>

            <label for="taskInfo_{{$task}}_FileMaxTime">file_max_time:</label>
            <input type="text" id="taskInfo_{{$task}}_FileMaxTime" name="taskInfo_{{$task}}_FileMaxTime" value="{{$info.FileMaxTime}}"><br>

        </div>
    </div>
    {{end}}

    <button type="button" onclick="updateConfig()">保存配置</button>
</form>
`
	c.Header("Content-Type", "text/html")

	tasks1 := readConf("config.yaml") // 假设你有一个 readConf 函数
	tasks := ConfigStruct{
		*tasks1,
		ipFiles,
		domainFiles,
	}
	t := template.Must(template.New("config").Parse(tmpl))
	t.Execute(c.Writer, tasks)

}

func updateConfig(c *gin.Context) {
	var inputData map[string]string
	// 绑定 JSON 数据到临时结构体
	if err := c.ShouldBindJSON(&inputData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newConfig, _ := parseJSON(inputData)
	fmt.Println(newConfig)

	data, err := yaml.Marshal(newConfig)
	if err != nil {
		c.String(http.StatusInternalServerError, "无法更新配置")
		return
	}

	err = os.WriteFile("config2.yaml", data, 0644) // 假设配置文件名为 config.yaml
	if err != nil {
		c.String(http.StatusInternalServerError, "无法保存配置")
		return
	}

	loadConfig() // 重新加载配置
	c.String(http.StatusOK, "配置更新成功")
}

func deleteFile(c *gin.Context) {
	file := c.Query("file")
	if file == "" {
		c.String(http.StatusBadRequest, "文件名不能为空")
		return
	}

	err := os.Remove(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "无法删除文件")
		return
	}

	loadFiles() // 重新加载文件
	c.String(http.StatusOK, "文件删除成功")
}
