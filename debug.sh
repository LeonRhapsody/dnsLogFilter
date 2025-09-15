#!/bin/bash
# 直接使用dlv进行调试，绕过IDE按钮问题
# 确保已安装dlv: go install github.com/go-delve/delve/cmd/dlv@latest

dlv debug main.go --headless --listen=:2345 --api-version=2

# 使用说明:
# 1. 保存文件后执行: chmod +x debug.sh
# 2. 运行调试服务器: ./debug.sh
# 3. 在Trae IDE中使用"Attach to Process"功能连接到localhost:2345