### 启动部署命令
cd web
npm install
npm run build
cd ..
go run ./cmd/cc-connect --config ./config.toml
<!-- go run -tags no_web -->

### 添加dify 对话支持


### 打包
go build -o ./dist/xuan-connect.exe ./cmd/cc-connect
go build -o ./dist/xuan-connect ./cmd/cc-connect

libc异常打包方式
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./dist/xuan-connect ./cmd/cc-connect

###  异常总结
超时请在agent.option  添加time_secs 参数控制
agent会话启动失败 大概率config.toml工作目录work_dir没找到