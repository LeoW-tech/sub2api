# CRS 分桶协议扩展：账号顶层 `proxy_name`

本文档定义 sub2api 对 CRS 账号同步协议的增量扩展。

目标很简单：

- 继续使用现有 CRS 分桶结构
- 不新增第二套导入协议
- 在每个账号对象顶层增加一个可选字段 `proxy_name`
- sub2api 拉取后按本地代理 `name` 严格匹配
- 匹配成功则自动绑定 `proxy_id`
- 未提供 `proxy_name` 或匹配失败时，账号仍正常导入

## 1. 适用范围

适用于以下所有 CRS 账号分桶：

- `claudeAccounts`
- `claudeConsoleAccounts`
- `openaiOAuthAccounts`
- `openaiResponsesAccounts`
- `geminiOAuthAccounts`
- `geminiApiKeyAccounts`

## 2. 新增字段

在每个账号对象顶层新增一个可选字段：

```json
{
  "id": "acc_openai_001",
  "name": "账号A",
  "proxy_name": "🇭🇰 香港W01",
  "credentials": {
    "api_key": "sk-xxxx"
  }
}
```

字段规则：

- 字段名固定为 `proxy_name`
- 类型为字符串
- 可选
- 推荐值直接等于 sub2api 本地代理显示名称
- 未提供时，sub2api 不会因为缺少该字段而报错

## 3. 匹配规则

sub2api 在同步时只按下面的规则处理 `proxy_name`：

- 仅使用账号顶层 `proxy_name`
- 仅与本地代理 `name` 做严格字符串匹配
- 不做大小写归一化
- 不做空格裁剪后重比
- 不做 emoji 清洗
- 不做模糊匹配
- 不做别名映射
- 不使用 `proxy_key`、`proxy_external_key` 兜底匹配

结果规则：

- 命中 1 条本地代理：绑定该代理
- 命中 0 条：账号继续导入，但不绑定代理
- 命中多条：账号继续导入，但不绑定代理

## 4. 兼容性说明

本扩展不会改变现有 CRS 字段要求：

- 旧上游不传 `proxy_name` 时，仍可继续使用
- 账号凭证字段要求保持不变
- 现有 CRS 中的 `proxy` 对象仍可保留

当 `proxy_name` 未提供时：

- sub2api 仍按原有 CRS 同步逻辑处理账号
- 如果管理员勾选了“同步代理”，仍会沿用现有按 `host/port/username/password` 匹配或创建代理的行为

当 `proxy_name` 已提供时：

- sub2api 优先按 `proxy_name` 绑定本地代理
- 不再回退到基于 CRS `proxy` 对象的新建/匹配逻辑

## 5. 顶层结构示例

顶层结构保持 CRS 现有格式不变：

```json
{
  "success": true,
  "data": {
    "exportedAt": "2026-04-09T05:00:00Z",
    "claudeAccounts": [],
    "claudeConsoleAccounts": [],
    "openaiOAuthAccounts": [],
    "openaiResponsesAccounts": [],
    "geminiOAuthAccounts": [],
    "geminiApiKeyAccounts": []
  }
}
```

## 6. 各分桶最小示例

### 6.1 `claudeAccounts`

```json
{
  "kind": "claude",
  "id": "claude-oauth-001",
  "name": "Claude OAuth 账号",
  "authType": "oauth",
  "proxy_name": "🇭🇰 香港W01",
  "isActive": true,
  "schedulable": true,
  "priority": 10,
  "status": "active",
  "credentials": {
    "access_token": "token-value"
  },
  "extra": {}
}
```

### 6.2 `claudeConsoleAccounts`

```json
{
  "kind": "claude-console",
  "id": "claude-console-001",
  "name": "Claude Console 账号",
  "proxy_name": "🇭🇰 香港W01",
  "isActive": true,
  "schedulable": true,
  "priority": 10,
  "status": "active",
  "maxConcurrentTasks": 3,
  "credentials": {
    "api_key": "sk-ant-xxx"
  }
}
```

### 6.3 `openaiOAuthAccounts`

```json
{
  "kind": "openai-oauth",
  "id": "openai-oauth-001",
  "name": "OpenAI OAuth 账号",
  "proxy_name": "🇭🇰 香港W01",
  "isActive": true,
  "schedulable": true,
  "priority": 10,
  "status": "active",
  "credentials": {
    "access_token": "access-token",
    "refresh_token": "refresh-token"
  },
  "extra": {}
}
```

### 6.4 `openaiResponsesAccounts`

```json
{
  "kind": "openai-responses",
  "id": "openai-key-001",
  "name": "OpenAI API Key 账号",
  "proxy_name": "🇭🇰 香港W01",
  "isActive": true,
  "schedulable": true,
  "priority": 10,
  "status": "active",
  "credentials": {
    "api_key": "sk-xxx",
    "base_url": "https://api.openai.com"
  }
}
```

### 6.5 `geminiOAuthAccounts`

```json
{
  "kind": "gemini-oauth",
  "id": "gemini-oauth-001",
  "name": "Gemini OAuth 账号",
  "proxy_name": "🇭🇰 香港W01",
  "isActive": true,
  "schedulable": true,
  "priority": 10,
  "status": "active",
  "credentials": {
    "refresh_token": "refresh-token"
  },
  "extra": {}
}
```

### 6.6 `geminiApiKeyAccounts`

```json
{
  "kind": "gemini-apikey",
  "id": "gemini-key-001",
  "name": "Gemini API Key 账号",
  "proxy_name": "🇭🇰 香港W01",
  "isActive": true,
  "schedulable": true,
  "priority": 10,
  "status": "active",
  "credentials": {
    "api_key": "AIza..."
  },
  "extra": {}
}
```

## 7. 凭证最低要求

`proxy_name` 只是代理关联字段，不改变账号本身的凭证要求。

最低要求仍沿用当前 sub2api 的 CRS 解析逻辑：

- `claudeAccounts` / `openaiOAuthAccounts`：至少提供 `access_token`
- `geminiOAuthAccounts`：至少提供 `refresh_token`
- `claudeConsoleAccounts` / `openaiResponsesAccounts` / `geminiApiKeyAccounts`：至少提供 `api_key`

## 8. 同步结果说明

### 8.1 预览接口

`POST /api/v1/admin/accounts/sync/crs/preview`

每条账号预览结果会附带：

- `proxy_name`
- `matched_proxy_id`
- `proxy_match_status`
- `warnings`

`proxy_match_status` 可能值：

- `matched`
- `missing`
- `not_found`
- `conflict`

### 8.2 正式同步接口

`POST /api/v1/admin/accounts/sync/crs`

整体结果会附带：

- `proxy_matched`
- `proxy_unmatched`

每条结果项会附带：

- `proxy_name`
- `matched_proxy_id`
- `warnings`

## 9. 常见告警

- `proxy_name not found: xxx`
  说明：本地没有名称完全一致的代理

- `proxy_name is ambiguous: xxx`
  说明：本地存在多个同名代理，无法唯一绑定

这些情况都不会阻止账号导入，只会跳过代理绑定。
