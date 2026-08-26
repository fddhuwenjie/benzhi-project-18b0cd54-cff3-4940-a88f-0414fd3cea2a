# BENZHI_README

## 项目说明
- 项目：benzhi-project-18b0cd54-cff3-4940-a88f-0414fd3cea2a
- 项目用途：已完整实现厌氧考古土壤样本从建档、封存转运、偏差复测、污染复核到培养放行和只读证据归档的版本化 HTTP JSON API。
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 项目描述
- 项目名称：厌氧考古土壤样本转运放行台
- 项目介绍：为考古现场出土的厌氧土壤微生物样本提供从建档、封存转运、偏差纠正到培养放行的一条可追溯 HTTP API 流程。服务校验氧温读数和证据摘要，支持并发控制、幂等重试及最终证据归档。
- 项目概述：为考古现场出土的厌氧土壤微生物样本提供从建档、封存转运、偏差纠正到培养放行的一条可追溯 HTTP API 流程。服务校验氧温读数和证据摘要，支持并发控制、幂等重试及最终证据归档。
- 核心工作流：样本批次建档并冻结厌氧保存方案，记录封装和转运检查点；若氧浓度或温度超限则隔离批次、执行纠正并复测，通过污染检测和独立复核后签发培养放行证书并封存证据。
- 对外接口：版本化 HTTP JSON API（监听地址支持 -addr=127.0.0.1:<port> 或 PORT 环境变量，默认 127.0.0.1:19081）

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...

cd /app && GOTOOLCHAIN=local go run ./cmd/anaerobic-release -self-check -addr=127.0.0.1:19081

cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh

./build_benzhi_docker.sh benzhi-project-18b0cd54-cff3-4940-a88f-0414fd3cea2a-amd64 linux/amd64

./build_benzhi_docker.sh benzhi-project-18b0cd54-cff3-4940-a88f-0414fd3cea2a-arm64 linux/arm64

docker run -it benzhi-project-18b0cd54-cff3-4940-a88f-0414fd3cea2a-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/anaerobic-release -self-check -addr=127.0.0.1:19081`
