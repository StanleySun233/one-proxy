# Issue 1 Plan

## 目标

围绕 `issue_1.md` 中列出的缺口，分阶段把后端从“初版骨架”推进到“功能闭环、合同一致、可验证”。

## 修复原则

- 先修正合同，再补实现
- 先保证控制面正确，再扩展节点侧
- 先做可验证闭环，再做增强能力
- 每一阶段结束后同步文档与清单

## 阶段 1：校正合同与完成标准

### 任务

- 更新 `docs/12-control-plane-api-payloads.md`
- 更新 `openapi.yaml`
- 更新 `docs/3-api-design.md`
- 更新 `todolist.md`
- 明确哪些能力属于：
  - 已实现
  - 部分实现
  - 未实现

### 交付标准

- API 文档与当前后端一致
- todo 清单能反映真实状态
- 后续实现不再基于过时合同

## 阶段 2：补齐控制面关键缺口

### 任务

- 实现 bootstrap 随机管理员密码生成
- 设计一次性展示机制
- 增加管理员首次启动输出记录方案
- 将 session/bootstrap/node token TTL 统一收口到配置与业务层
- 补充更细的业务错误码

### 交付标准

- `admin` bootstrap 符合产品定义
- 控制面初始化流程具备最小安全性

## 阶段 3：完成 enrollment 审批与 trust 模型

### 任务

- 为 node enrollment 增加待审批状态
- 区分 bootstrap token、pending node、active node
- 设计并落地 trust material 存储结构
- 增加 enrollment approve API
- 节点只有在审批通过后才能拿到长期 node token

### 交付标准

- 节点接入链路与产品定义一致
- 多节点信任模型开始成立

## 阶段 4：重构策略编译与节点快照分发

### 任务

- 将 policy compiler 从全局快照改为按节点编译
- 在 snapshot 中裁剪无关 hop、node、rule 视图
- 为每个节点持久化已分配 revision 和 snapshot
- 明确 disabled/revoked node 的编译与分发规则

### 交付标准

- `compile snapshot per node` 真正成立
- 发布逻辑满足文档 failure rules

## 阶段 5：补齐节点本地持久化与恢复

### 任务

- 为 node agent 增加本地策略存储
- 持久化：
  - current revision
  - last known good snapshot
  - cert material reference
- sync 失败时继续使用 last known good
- 节点重启后自动恢复活动配置

### 交付标准

- 节点具备基本 failure recovery
- 不再依赖纯内存策略状态

## 阶段 6：扩展路由能力与数据面闭环

### 任务

- 增加 `cidr` 匹配
- 增加 `protocol` 匹配
- 明确 default deny/default direct 行为
- 校正 chain forwarding 的边界行为
- 验证 WebSocket 转发路径
- 补充 listener 与 relay 结构

### 交付标准

- 路由规则能力与文档一致
- 节点数据面达到可运行最小闭环

## 阶段 7：完成证书自动化

### 任务

- 设计 edge 公网证书来源
- 设计私有 trust 证书来源
- 落地实际续签任务
- 将 scheduler 从“状态刷新”升级为“真实生命周期编排”
- 补充证书状态和分发更新逻辑

### 交付标准

- 证书功能不再是占位接口
- 控制面和节点侧都能消费真实证书状态

## 阶段 8：验证与收尾

### 任务

- 编译后端各入口
- 做最小集成验证：
  - login
  - publish policy
  - enroll node
  - node sync
  - direct route
  - chain route
  - cert maintenance
- 修正文档残差
- 更新 `todolist.md`

### 交付标准

- 后端能力和文档一致
- 关键链路有实际验证证据

## 当前执行顺序

1. 阶段 1：校正合同与清单
2. 阶段 2：bootstrap 安全化
3. 阶段 3：enrollment 审批与 trust
4. 阶段 4：按节点编译策略
5. 阶段 5：节点持久化与恢复
6. 阶段 6：扩展路由与数据面
7. 阶段 7：证书自动化
8. 阶段 8：验证与收尾
