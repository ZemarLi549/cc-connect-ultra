### 启动部署命令
cd web
npm install
npm run build
cd ..
go run ./cmd/cc-connect --config .\config.toml
<!-- go run -tags no_web -->

### 添加dify 对话支持
