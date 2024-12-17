package manager

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

func readConf(filename string) *jsonData {
	configBytes, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}
	// 创建配置文件的结构体
	var tasks jsonData
	// 调用 Unmarshall 去解码文件内容
	// 注意要穿配置结构体的指针进去
	yaml.Unmarshal(configBytes, &tasks)
	err = yaml.Unmarshal(configBytes, &tasks)
	if err != nil {
		log.Fatalln(err)
	}

	return &tasks

}
