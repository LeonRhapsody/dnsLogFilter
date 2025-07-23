package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configFileName string

var initConfigFileCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a template config file",
	Long:  `Generate a template config file with default values.`,
	Run: func(cmd *cobra.Command, args []string) {
		content := `
#eth_name: 业务网卡名称（非流量网卡，仅用于获取本机IP，定义文件名称）
#analyze_threads： 分析线程数目（建议从小到大调试）
#input_dir：日志输入目录
#input_format： 指定输入目录的格式
#backup_dir：日志处理完成之后，日志移动的位置，为空则不移动
#online_mode： 在线分析/离线分析（在线分析只分析增量文件，分析完成会一直等待新文件产生/离线分析仅分析存量文件，分析完成后退出）

eth_name: "en0"
analyze_threads: 1
input_dir: "./input"
input_format: "r,12,3,4,1,2,5,6,7,14,19,15,13"
backup_dir: "./backup"
online_mode: true

#enable：模块开关（true开启过滤）
#output_dir：结果文件输出目录
#outpur_format: 输出格式，有三种（1、jituan:drms转集团日志格式输出；2、full:直接输出源格式；3、自定义字段格式）
#is_gzip：结果文件是否压缩
#filter_domain_ruler: 域名过滤清单，为空代表不过滤
#filter_ip_ruler: ip过滤清单，为空代表不过滤
#file_max_size: 不填写默认 200M
#file_max_time: 不填写默认为9999h，即永不截断
#domain_exact_match：域名精准过滤。默认会将过滤清单中的域名视为泛域名，如果为true则视为精确域名

task_infos:
    apt:
        enable: true
        filter_ip_ruler:
        filter_domain_ruler:
            - "./apt.list"
        output_dir: "./data/apt"
        output_format: "6,12,17,18"
        is_gzip: true
        file_max_size: 200M
        file_max_time: 1m
        output_file_name: 2500_time_ip

    aliyun:
        enable: true
        filter_domain_ruler:
            - "./1.txt"
        filter_ip_ruler:
            - "ip_rule_1.txt"
        output_dir: "./data/aliyun"
        output_format: "full"
        domain_exact_match: false
        is_upload: false
        is_gzip: false
        file_max_size: 200M
        file_max_time: 1m
`
		if err := os.WriteFile(configFileName, []byte(content), 0644); err != nil {
			fmt.Println("Error writing file:", err)
		} else {
			fmt.Printf("Template config file %s created successfully\n", configFileName)
		}
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(initConfigFileCmd)
	initConfigFileCmd.Flags().StringVarP(&configFileName, "output", "o", "config.yaml.template", "output config file name")
}
