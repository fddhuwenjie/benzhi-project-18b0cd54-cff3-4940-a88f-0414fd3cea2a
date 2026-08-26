# 厌氧考古土壤样本转运放行台

本项目为考古现场出土的厌氧土壤微生物样本提供可追溯的 HTTP JSON API。系统覆盖批次建档、带交接确认的保存方案冻结、连续转运检查、超限隔离与复测、版本化污染检测、独立质量复核、培养放行及只读证据归档。

所有写请求都必须携带 `Idempotency-Key` 请求头，并在 JSON 请求体中提供 `expected_revision`。前者保证请求跨重启重试时返回相同结果，后者阻止陈旧客户端覆盖较新的批次状态。数据写入原子 JSON 快照，审计记录使用只追加 SHA-256 摘要链；服务启动恢复和归档读取都会验证快照结构、恢复元数据和审计连续性。

## 构建、运行和测试

项目要求 Go 1.22 或更高版本，不依赖第三方模块。

```bash
go build ./cmd/anaerobic-release
go test ./...
go run ./cmd/anaerobic-release -addr=127.0.0.1:19081 -data=data/snapshot.json
```

服务默认监听 `127.0.0.1:19081`。可以使用 `-addr=127.0.0.1:<port>` 指定高位回环端口，也可以设置 `PORT` 环境变量；为避免意外暴露样本数据，程序拒绝非回环 IP 和 1024 以下的端口。`-data` 用于指定持久化快照路径。

运行会自动结束的真实业务自检：

```bash
go run ./cmd/anaerobic-release -self-check -addr=127.0.0.1:19081
```

该命令会在临时目录创建快照，通过真实回环 HTTP 请求完成建档、方案冻结、转运检查、污染检测、独立复核、证书签发和证据读取，然后优雅停止服务并清理临时数据。

## API 流程

API 使用 `application/json`，单个请求体最大为 1 MiB，未知字段会被拒绝。时间字段使用 RFC 3339 格式，证据摘要使用 64 位 SHA-256 十六进制文本。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/healthz` | 验证服务和当前审计链 |
| `POST` | `/v1/batches` | 建立批次，初始 `expected_revision` 为 `0` |
| `GET` | `/v1/batches/{batch_id}` | 查询批次聚合及当前 `revision` |
| `POST` | `/v1/batches/{batch_id}/preservation-plan` | 校验初始读数并冻结容器、阈值和交接确认 |
| `POST` | `/v1/batches/{batch_id}/checkpoints` | 登记氧浓度、温度、密封状态和现场证据 |
| `POST` | `/v1/batches/{batch_id}/deviations/{deviation_id}/resolve` | 登记纠正措施并在同一命令中提交合格复测 |
| `POST` | `/v1/batches/{batch_id}/contamination-tests` | 登记培养前污染检测 |
| `POST` | `/v1/batches/{batch_id}/reviews` | 由职责独立的质量人员复核 |
| `POST` | `/v1/batches/{batch_id}/release` | 由独立签发员签发证书并转为只读归档 |
| `GET` | `/v1/batches/{batch_id}/evidence` | 读取证书、证据摘要和可验证审计证明 |

冻结请求除保存方案字段外还必须提交 `handover_at` 和 `handover_evidence_digest`。交接时间不得早于采样或晚于当前时间，责任交接人不得与采样员相同；初始氧浓度或温度不符合拟定阈值时，批次保持 `draft`。查询结果会返回交接确认、`preservation_plan_summary` 和 `plan_frozen_at`。

检查点时间必须严格递增且不得早于方案冻结时间，同一批次的 `checkpoint_id` 和 `evidence_digest` 均不可复用。查询中的检查点包含连续 `sequence`、`previous_checkpoint_id` 和起点标记，`transfer_continuity` 汇总最新时间、节点数、断裂及超限情况。若氧浓度、温度或密封状态不符合冻结方案，批次会自动进入 `quarantined` 并创建偏差；只有带新检查点证据的合格复测才能闭环。

污染检测与质量复核分别追加到 `contamination_history` 和 `review_history`，旧结论不会被覆盖。`current_release_version` 明确标识当前配对；复核驳回后必须产生更新的检测和批准复核。复核员不得是采样员、当前检测员或偏差纠正人员，签发员也必须独立于采样员和复核员。

成功签发后状态为 `released_archived`，所有写接口都会拒绝继续修改。证书的 `evidence_manifest` 以稳定顺序逐项绑定方案、检查点、已闭环偏差、当前检测、当前复核和签发证据及其审计事件；`integrity_digest` 覆盖清单、决定、revision 和签发前审计头。归档证据读取会重新交叉核验快照、索引、证书、清单和审计链，成功时返回 `verification: "verified"`，不一致时返回可定位的 `integrity_error`。

## 持久化和错误

快照默认位于 `data/snapshot.json`。每次提交先构造并校验完整副本，再同步临时文件并原子替换正式文件；失败的事务不会改变内存或磁盘状态。请在生产部署中对该文件实施访问控制和备份。

错误统一返回 `error.code`、中文 `error.message` 和可用时的 `error.request_id`。常见代码包括 `validation_error`、`revision_conflict`、`idempotency_conflict`、`checkpoint_conflict`、`evidence_conflict`、`invalid_state`、`release_blocked` 和 `integrity_error`。
