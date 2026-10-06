# OTP Auth Server — Agentic Development Lab

這不是一個單純練習 Go OTP Server 的專案。

這個專案的真正目標是：

> **透過一個小型 Go API 專案，實際體驗如何逐步把軟體開發工作交給 Claude Code、Codex 等 CLI Agent。**

整個實驗分成四個 Phase：

1. **AI 寫，我看得懂**
2. **AI 寫，機器幫我驗證**
3. **AI 自己跑完整 Workflow**
4. **Human 管目標與例外**

最終目標不是「完全不看程式碼」，而是理解：

> **哪些工作可以安全交給 AI？  
> 我要建立哪些驗證機制，才敢把更多工作交出去？**

---

# 1. 最終想驗證的假設

這個實驗要驗證：

```text
如果：

1. Human 能清楚描述目標
2. Agent 有足夠 Context / Rules
3. 系統有可靠的 Tests / Contract / Validation
4. Agent 有固定 Workflow
5. CI 能阻止錯誤結果進入

那麼：

Human 是否可以逐漸減少直接寫 Code，
把工作重心移到 Goal、Constraints、Verification 和 Exceptions？
```

四個 Phase 不是四套不同專案。

而是同一個專案逐步演進：

```text
Phase 1
AI 寫，我看得懂
        │
        ▼
Phase 2
AI 寫，機器幫我驗證
        │
        ▼
Phase 3
AI 自己跑完整 Workflow
        │
        ▼
Phase 4
Human 管目標與例外
```

---

# 2. 實驗原則

## Rule 1 — 不提前建立「完美 AI Workflow」

一開始不要直接建立：

```text
CLAUDE.md
AGENTS.md
.claude/rules/
skills/
agents/
hooks/
OpenAPI
CI
Evals
大量 scripts
```

只有當實驗真的遇到問題時，才加入解決問題的工具。

例如：

```text
AI 一直犯相同錯誤
        ↓
考慮加入 Rule


不知道 AI 是否做對
        ↓
加入 Test / Contract


每次都手動執行相同 commands
        ↓
加入 Makefile / Script


AI 每次都需要我告訴它下一步
        ↓
建立 Workflow


AI 說完成，但我不確定
        ↓
加入更完整的 Verification / CI
```

每增加一個工具，都要知道：

> **它正在解決什麼問題？**

---

## Rule 2 — 一次只增加一層 Delegation

不要從：

```text
Human 寫所有 Code
```

直接跳到：

```text
AI 自己開發整個系統
```

我們要刻意經歷中間過程：

```text
Human 寫
   ↓
AI 寫 + Human Review
   ↓
AI 寫 + Machine Verify + Human Review
   ↓
AI Implement + Test + Verify
   ↓
AI 跑 Workflow + Human Approval
```

---

## Rule 3 — 不以「AI 說完成」作為完成條件

下面這句：

```text
Implementation completed successfully.
```

不是證據。

必須問：

```text
你怎麼證明它完成？
```

隨著 Phase 推進，證據可能包含：

```text
Tests
Contract
Static Analysis
API Tests
CI
Code Review
Smoke Test
Evals
```

---

## Rule 4 — 不追求 Production Ready

這是學習專案。

優先順序：

```text
理解 Workflow
    >
理解 Verification
    >
理解 Agent Delegation
    >
漂亮 Architecture
    >
Production Ready
```

不要因為「真正 OTP Server 應該用 Redis」就立刻加入 Redis。

除非它對某個實驗有明確價值。

---

# 3. OTP Server Scope

建立一個最小的 Go HTTP API Server。

只有三個 endpoints：

```text
POST /send-otp

POST /verify-otp

POST /login
```

第一版使用：

```text
Go
net/http
in-memory storage
fake OTP sender
Go tests
```

暫時不要加入：

```text
PostgreSQL
Redis
AWS
Kubernetes
真正 Email / SMS
複雜 Authentication Framework
```

---

# 4. 基本 User Flow

系統流程：

```text
User
 │
 │ email
 ▼
POST /send-otp
 │
 │
 ▼
Generate OTP
 │
 ▼
Store OTP in memory
 │
 ▼
Fake Send OTP
 │
 │
 ▼
POST /verify-otp
 │
 │ email + otp
 ▼
Verify OTP
 │
 ▼
Mark User Verified
 │
 │
 ▼
POST /login
 │
 ▼
Check Verification
 │
 ▼
Return Token
```

---

# 5. 初始 API Contract

這只是實驗的初始需求。

之後如果要修改 Contract，必須明確記錄原因。

## POST `/send-otp`

Request：

```json
{
  "email": "user@example.com"
}
```

Success：

```json
{
  "message": "otp sent"
}
```

---

## POST `/verify-otp`

Request：

```json
{
  "email": "user@example.com",
  "otp": "123456"
}
```

Success：

```json
{
  "verified": true
}
```

Invalid OTP：

```json
{
  "error": "invalid otp"
}
```

---

## POST `/login`

Request：

```json
{
  "email": "user@example.com"
}
```

如果已完成 OTP verification：

```json
{
  "token": "..."
}
```

如果尚未完成：

```json
{
  "error": "email not verified"
}
```

---

# 6. Phase 1 — AI 寫，我看得懂

## Goal

第一階段不要追求自動化。

目標是：

> **AI 寫 Code，但 Human 能理解它做了什麼。**

目前 Human 還是主要控制者。

```text
Human
  │
  │ 小 Task
  ▼
Claude Code
  │
  │ Implementation
  ▼
Human Review
  │
  ├── 看 Diff
  ├── 理解 Flow
  └── 提問題
  │
  ▼
Run Tests
```

---

## Experiment 1.1 — 建立最小 HTTP Server

不要叫 Agent：

```text
Build the entire OTP server.
```

只要求：

```text
建立最小 Go HTTP server。

目前不要實作 OTP endpoints。

完成後：
1. 告訴我新增哪些 files
2. 說明 HTTP server 從哪裡啟動
3. 不要繼續下一個 task
```

完成後，Human 必須自己找到：

```text
main()
  ↓
HTTP Server
  ↓
Router / Handler
```

### Checkpoint

確認自己能回答：

- Server 從哪個 function 啟動？
- Port 在哪裡設定？
- Request 如何進入 Handler？
- 如何停止 Server？

回答不出來就問 Agent。

不要直接進下一題。

---

# 7. Experiment 1.2 — `/send-otp`

要求 Agent 只完成：

```text
POST /send-otp
```

不要同時做 `/verify-otp`。

完成後查看：

```bash
git diff
```

Human 嘗試沿著 Code 找：

```text
POST /send-otp
       ↓
Handler
       ↓
Parse JSON
       ↓
Validate
       ↓
Generate OTP
       ↓
Store OTP
       ↓
Response
```

### Human Exercise

不要先問 AI。

自己嘗試回答：

1. Request JSON 在哪裡 parse？
2. Email validation 在哪裡？
3. OTP 在哪裡產生？
4. OTP 存在哪裡？
5. Response 在哪裡產生？

再請 Agent：

```text
不要修改任何 code。

請檢查我的理解是否正確：

[寫下自己的理解]
```

---

# 8. Experiment 1.3 — `/verify-otp`

要求 Agent：

```text
新增 POST /verify-otp。

只完成這個 endpoint。
不要修改 /send-otp contract。
不要實作 /login。

完成後停止。
```

Human 再沿著：

```text
Request
 ↓
Handler
 ↓
Store
 ↓
Compare OTP
 ↓
Mark Verified
 ↓
Response
```

理解一次。

---

# 9. Experiment 1.4 — `/login`

最後加入：

```text
POST /login
```

完成：

```text
send OTP
   ↓
verify OTP
   ↓
login
```

整條 Flow。

---

# 10. Phase 1 Final Challenge

現在不要問 Agent。

Human 自己解釋：

```text
POST /send-otp
```

Request 如何一路走到 Response。

接著：

```text
POST /verify-otp
```

最後：

```text
POST /login
```

如果能說出：

```text
HTTP Request
    ↓
Handler
    ↓
Validation
    ↓
Business Logic
    ↓
Store
    ↓
Response
```

就算 Phase 1 通過。

## Phase 1 Exit Criteria

我不需要能從零寫出整個 Server。

但我要：

- 看得懂主要 Go syntax
- 找得到 Handler
- 找得到 Business Logic
- 找得到 Storage
- 找得到 Response
- 能解釋三個 Endpoint 的資料流
- 看得懂主要 Git Diff

通過後才進 Phase 2。

---

# 11. Phase 2 — AI 寫，機器幫我驗證

現在開始改變思考方式。

Phase 1：

```text
AI 寫
 ↓
我看 Code
 ↓
我覺得 OK
```

Phase 2：

```text
Requirement
 ↓
AI 寫
 ↓
Machine Verification
 ↓
Evidence
 ↓
Human Review
```

核心問題變成：

> **我怎麼證明 AI 做對了？**

---

# 12. Experiment 2.1 — 建立 Behaviour Tests

先列出 `/send-otp` 的行為。

例如：

```text
valid email
→ success

missing email
→ error

invalid email
→ error
```

`/verify-otp`：

```text
correct OTP
→ verified

wrong OTP
→ rejected

unknown email
→ rejected
```

`/login`：

```text
verified email
→ token

unverified email
→ rejected
```

請 Agent 為這些行為建立 Tests。

---

# 13. Experiment 2.2 — 故意製造 Bug

這是非常重要的實驗。

故意要求 Agent 或自己做一個錯誤。

例如：

```text
讓 /verify-otp 接受任何 OTP。
```

先不要修改 Test。

執行：

```bash
go test ./...
```

預期：

```text
FAIL
```

如果 Test 沒有 Fail：

> Test 沒有保護我們真正關心的 Behaviour。

修正 Test。

再重複一次。

目標不是讓 Test 全綠。

而是親眼看到：

```text
Bad Implementation
        ↓
Test
        ↓
FAIL
```

這就是 Machine Verification 開始產生信任的地方。

---

# 14. Experiment 2.3 — 建立統一 Verification Entry Point

當需要執行的 Command 開始增加：

```bash
gofmt
go vet
go test ./...
...
```

不要繼續手動一個一個輸入。

建立統一入口，例如：

```bash
make verify
```

概念：

```text
make verify
    │
    ├── Format Check
    ├── Static Analysis
    ├── Unit Tests
    └── API / Integration Tests
```

實際包含什麼，以專案當時真正需要的項目為準。

不要為了看起來完整而增加沒有用途的工具。

---

# 15. Experiment 2.4 — API Contract

現在重新看三個 endpoints：

```text
/send-otp
/verify-otp
/login
```

思考：

> 如果 Agent 不小心把 Response 改掉，我怎麼知道？

例如：

原本：

```json
{
  "verified": true
}
```

Agent 不小心改成：

```json
{
  "success": true
}
```

程式可能：

```text
Compile ✓
Unit Tests ✓
Server runs ✓
```

但 Client 已經壞掉。

因此這時候開始研究：

```text
API Contract
```

並考慮加入 OpenAPI。

目標不是學完整 OpenAPI。

只要能保護：

```text
Method
Path
Request
Response
Status Code
```

---

# 16. Phase 2 Final Challenge

要求 Agent 做一個小改動。

例如：

```text
OTP 必須剛好是 6 位數字。
```

完成後不要先問：

> 改了哪些 Code？

先問：

> **請提供證據證明 Requirement 已經滿足。**

期待看到類似：

```text
Requirement
    ↓
Tests
    ↓
make verify
    ↓
PASS
```

## Phase 2 Exit Criteria

我開始習慣：

```text
Requirement
    ↓
Verification
```

而不是：

```text
Implementation
    ↓
Looks Good
```

並且專案至少有一個統一的 Verification Entry Point。

---

# 17. Phase 3 — AI 自己跑完整 Workflow

現在 Human 不再指導每一個步驟。

目標：

> **給 Agent 一個小 Requirement，讓 Agent 自己 Explore → Plan → Implement → Verify。**

Workflow：

```text
Requirement
     ↓
   Explore
     ↓
    Plan
     ↓
 Implement
     ↓
   Tests
     ↓
  Verify
     ↓
  Review
     ↓
   Result
```

---

# 18. Experiment 3.1 — 第一個完整 Agent Task

Requirement：

```text
OTP 60 秒後失效。
```

這次不要告訴 Agent：

```text
修改哪個 file
建立哪個 struct
加入哪個 field
使用哪個 function
```

只給：

```text
Goal:
OTP expires after 60 seconds.

Constraints:
- Do not change existing API contract.
- Existing tests must continue to pass.

Before implementation:
1. Explore the relevant code.
2. Create a plan.
3. Show me the plan.
4. Do not implement yet.
```

Human Review Plan。

確認方向沒有明顯問題後，再讓 Agent Implement。

---

# 19. Experiment 3.2 — Agent 自己 Verification

下一個 Requirement：

```text
OTP 只能成功驗證一次。
```

這次要求：

```text
Explore
→ Plan
→ Implement
→ Add/Update Tests
→ Verify
→ Report
```

Human 不介入中間實作。

如果 Verification Fail：

```text
Agent
 ↓
分析 Failure
 ↓
修正
 ↓
重新 Verify
```

Human 不要立即幫忙。

只有 Agent 卡住才介入。

---

# 20. Experiment 3.3 — Independent Review

現在加入第二個 Agent。

例如：

```text
Claude Code
    ↓
Implementation
    ↓
Codex
    ↓
Independent Review
```

Codex 不負責重新實作。

它負責找：

```text
Bug
Regression
Missing Test
Contract Violation
Unexpected Scope
```

Human 比較：

```text
Claude 認為完成

vs.

Codex Review 發現什麼
```

這個實驗要回答：

> 第二個獨立 Agent 是否能降低第一個 Agent 的盲點？

---

# 21. Experiment 3.4 — CI

當 Local Verification 已經穩定後，再把 Verification 放進 CI。

例如：

```text
Agent
 ↓
Commit / PR
 ↓
GitHub
 ↓
CI
 ↓
make verify
 ↓
PASS / FAIL
```

重點：

Local 和 CI 儘量執行相同 Verification。

不要出現：

```text
Local 驗證 A

CI 驗證 B

Human 腦中還有 C
```

理想上：

```text
make verify
```

成為共同入口。

---

# 22. Phase 3 Final Challenge

給 Agent：

```text
同一個 email 在 OTP 發送後 30 秒內不能再次要求 OTP。

Requirements:
- Existing API contract must remain compatible.
- Existing behaviour must not regress.
- Add verification for the new behaviour.
```

然後讓 Agent自己：

```text
Explore
 ↓
Plan
 ↓
Implement
 ↓
Test
 ↓
Verify
 ↓
Review
```

Human 最後才看：

```text
Goal 是否完成？
Tests 是否證明 Behaviour？
Verification 是否 PASS？
Review 是否有問題？
有沒有超出 Scope？
```

## Phase 3 Exit Criteria

Human 不需要：

- 指定每個 File
- 指定每個 Function
- 告訴 Agent 下一個 Command
- 手動處理正常的 Test Failure

Agent 已經能自己推進一個小型 Requirement。

---

# 23. Phase 4 — Human 管目標與例外

這一階段開始刻意減少 Implementation Instructions。

Human 主要提供：

```text
Goal

Constraints

Acceptance Criteria
```

Agent 決定：

```text
Files
Functions
Implementation
Tests
Commands
Workflow
```

---

# 24. Phase 4 Experiment

給 Agent 一個比較接近 Product Requirement 的需求：

## Goal

降低 OTP 被頻繁要求造成的問題。

## Constraints

```text
API contract 不可破壞。

現有 Client 不需要修改。

不增加 External Dependency。

現有 Tests 必須繼續通過。
```

## Acceptance Criteria

```text
OTP expires after 5 minutes.

Same email cannot request another OTP
within 60 seconds.

OTP can only be successfully verified once.
```

Human 不提供：

```text
File Name
Struct
Function
Algorithm
Package
Implementation Detail
```

---

# 25. Human 在 Phase 4 做什麼？

Human 主要看：

### Goal

我們解決的是正確問題嗎？

### Constraints

Agent 是否破壞既有限制？

### Acceptance Criteria

結果是否符合需求？

### Verification

證據是否足夠？

### Exception

有沒有 Agent 無法自行決定的事情？

例如 Agent 發現：

```text
Requirement 有兩種合理解讀。
```

這時才回來問 Human。

---

# 26. Phase 4 Target Workflow

最終希望達到：

```text
                  Human
                    │
          Goal / Constraints
                    │
                    ▼
               AI Workflow
                    │
          ┌─────────┴─────────┐
          │                   │
       Explore              Plan
          │                   │
          └─────────┬─────────┘
                    ▼
                Implement
                    │
                    ▼
                  Tests
                    │
                    ▼
                 Verify
                    │
                    ▼
             Independent Review
                    │
                    ▼
                   CI
                    │
                    ▼
                  Human
                    │
              ┌─────┴─────┐
              ▼           ▼
           Approve       Reject
```

---

# 27. Phase 4 Exit Criteria

Human 主要處理：

```text
Goal
Priority
Constraints
Acceptance Criteria
Trade-offs
Exceptions
Approval
```

Agent 主要處理：

```text
Explore
Planning
Implementation
Tests
Debugging
Verification
Routine Review
PR Preparation
```

Machine 主要處理：

```text
Tests
Contract Validation
Static Checks
CI
Repeatable Verification
```

---

# 28. 每個 Phase 都記錄 Delegation

完成每個 Phase 後，回答下面問題。

## Human 做了什麼？

```text
-
-
-
```

## Claude Code 做了什麼？

```text
-
-
-
```

## Codex 做了什麼？

```text
-
-
-
```

## Machine Verification 做了什麼？

```text
-
-
-
```

## 哪些事情我仍然不敢交給 AI？

```text
-
-
-
```

## 為什麼？

```text
-
-
-
```

這部分非常重要。

因為實驗的目的不是追求：

```text
AI 寫了多少 % Code？
```

而是觀察：

```text
我為什麼敢把這件事情交出去？
```

---

# 29. 遇到 Agent 做錯時

不要只修正當前問題。

使用下面流程：

```text
Agent 做錯
    ↓
先修正問題
    ↓
問：為什麼會發生？
    ↓
分類
```

可能原因：

```text
Requirement 不清楚
        ↓
改善 Requirement


Agent 不知道 Project Rule
        ↓
增加 Rule / Context


沒有 Test 抓到
        ↓
改善 Test


Contract 被偷偷改掉
        ↓
改善 Contract Validation


Agent 忘記跑 Verification
        ↓
改善 Workflow


CI 沒擋住
        ↓
改善 CI


同樣問題一直發生
        ↓
考慮 Rule / Eval
```

這就是這個 Lab 最重要的 Feedback Loop：

```text
        ┌──────────────────────────┐
        │                          │
        ▼                          │
Delegate → Failure → Understand → Improve
                              │
                              ▼
                       Better System
                              │
                              └──────────┐
                                         │
                                         ▼
                                  Delegate More
```

---

# 30. 不要把所有問題都變成 Rule

Rule 不是垃圾桶。

如果問題可以由 Test 保護：

```text
→ Test
```

如果問題是 API 格式：

```text
→ Contract
```

如果問題是重複 Commands：

```text
→ Script / Makefile
```

如果問題是 Agent 不知道流程：

```text
→ Workflow
```

如果問題是 Coding Convention：

```text
→ Rule
```

如果問題是 Requirement：

```text
→ Spec / Acceptance Criteria
```

選擇最接近問題本質的工具。

---

# 31. 最終回顧

完成四個 Phase 後，重新做一次最初的比較。

## Phase 1

```text
AI 寫
 ↓
Human 看 Code
 ↓
Human 判斷
```

## Phase 2

```text
AI 寫
 ↓
Machine Verify
 ↓
Human 看 Evidence
```

## Phase 3

```text
Requirement
 ↓
AI Workflow
 ↓
Machine Verify
 ↓
Independent Review
 ↓
Human Approval
```

## Phase 4

```text
Human
 │
 │ Goal
 │ Constraints
 │ Acceptance Criteria
 ▼
Agent System
 │
 │ Explore
 │ Plan
 │ Implement
 │ Test
 │ Debug
 │ Verify
 │ Review
 ▼
Human
 │
 ├── Approve
 └── Exception Handling
```

---

# 32. 最終問題

完成 Lab 後，不要只問：

> 我現在是不是可以不寫 Code？

回答下面這些問題：

### 1.

哪些工作我現在已經願意完全交給 Agent？

### 2.

哪些工作我仍然需要 Review？

### 3.

哪些工作我一定要自己決定？

### 4.

哪些 Verification 讓我敢少看 Code？

### 5.

Agent 最常在哪裡失敗？

### 6.

哪些錯誤已經被 Workflow 永久消除？

### 7.

如果把這套方法搬到真正的 Production Project，還缺什麼？

---

# 33. Success

這個專案成功的標準不是：

```text
OTP Server Works
```

真正成功是：

```text
我知道：

為什麼可以把某件事情交給 AI。

AI 做錯時，
哪一層應該抓到它。

如果沒有抓到，
我要改善哪一層。

改善完成後，
下一次就不需要再人工處理同樣的事情。
```

最終我們追求的不是：

> **Human 不寫 Code。**

而是：

> **Human 不再親手處理那些已經可以被可靠描述、執行、驗證與重複的工作。**

---

# Start Here

現在不要建立完整 Workflow。

不要先建立大量 Rules。

不要先設定 CI。

不要先加入 OpenAPI。

不要先讓 Agent 建完整 OTP Server。

從：

```text
Phase 1
Experiment 1.1
```

開始。

第一個任務只有：

```text
建立一個最小 Go HTTP Server。

不要實作任何 OTP endpoint。

完成後停止，
並讓 Human 理解 Server 是怎麼啟動的。
```

完成它、理解它、確認它。

然後才進入下一個 Experiment。

**一次只往前一步。**
