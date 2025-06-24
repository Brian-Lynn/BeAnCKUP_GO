# BeAnCKUP_GO 技术架构总览（2024重构版）

## 1. 高层架构
BeAnCKUP_GO 采用 **Wails 框架**，实现了 **Go 后端 + Web 前端** 的分离式架构。
- **后端（Go）**：负责所有核心业务逻辑，包括文件扫描、差异对比、备份清单管理、分包预估等。
- **前端（HTML/JS）**：负责用户界面和交互，通过 Wails 的 JS Bridge 调用后端方法，并通过事件机制接收进度和结果。

## 2. 数据流与核心流程
### 2.1 首次扫描（增量备份准备）
1. **用户操作**：在前端点击"首次扫描"按钮。
2. **参数收集**：前端收集工作区路径、交付路径、包大小上限等参数。
3. **后端入口**：前端调用 `StartBackupPreparation(workspacePath, maxPackageSizeGB, maxTotalSizeGB)`。
4. **后端处理**：
   - `task_manager` 调用 `indexer` 扫描所有文件。
   - `manifest_manager` 加载上一次的备份清单（manifest.json），如无则新建空清单。
   - `indexer.QuickScan` 对比新旧文件，找出所有"新增/修改/删除"文件。
   - 统计变更数量和总大小。
   - 预估分包（Episode），每包不超过设定上限。
   - 用 `tree_builder` 构建变更文件的目录树（TreeNode）。
   - 所有结果打包成 `BackupPreparationResult` 返回前端。
5. **前端渲染**：
   - 用 `result.fileTree` 渲染左侧文件树。
   - 用 `result.episodes` 渲染交付中心。
   - 用 `result.changeInfo` 更新状态栏。

### 2.2 进度反馈
- 后端扫描时通过 Wails 事件 `scan-progress` 实时向前端推送进度。
- 前端监听该事件，动态更新底部状态栏。

## 3. 核心模块职责
- **main.go**：程序入口，定义 App 结构体，负责前后端方法绑定。
- **backend/types/types.go**：定义所有核心数据结构（FileInfo、Manifest、TreeNode、Episode、BackupPreparationResult等）。
- **backend/indexer/indexer.go**：递归扫描目录，生成文件元数据，支持进度回调。实现 `QuickScan` 用于新旧清单对比。
- **backend/manifest_manager/manifest_manager.go**：负责清单（manifest.json）的加载与保存，自动处理首次备份和异常。
- **backend/tree_builder/tree_builder.go**：将变更文件列表转换为前端可用的目录树结构。
- **backend/task_manager/task_manager.go**：业务编排器，负责调用各模块完成"首次扫描"全流程，聚合所有结果。

## 4. 主要数据结构（types.go）
- **FileInfo**：单个文件的元数据（路径、大小、修改时间、状态等）。
- **Manifest**：一次备份的完整快照，记录所有文件、目录、哈希映射。
- **TreeNode**：前端文件树节点，支持递归嵌套。
- **Episode**：交付包（分包）信息。
- **BackupPreparationResult**：首次扫描后返回的聚合结果，包括分包、文件树、变更统计。

## 5. 前后端交互API
- `SelectDirectory()`：弹出目录选择框。
- `StartBackupPreparation(workspacePath, maxPackageSizeGB, maxTotalSizeGB)`：首次扫描/增量备份准备，返回所有变更、分包、文件树。
- `StartBackupExecution(...)`：启动实际备份（未重构）。
- `CopyToClipboard(text)`：复制文本到剪贴板。

## 6. 首次扫描完整流程（代码级）
1. 前端收集参数，调用 `StartBackupPreparation`。
2. `task_manager`：
   - 调用 `indexer.ScanWorkspace` 扫描所有文件（带进度回调）。
   - 调用 `manifest_manager.LoadLatestManifest` 加载旧清单。
   - 用 `indexer.QuickScan` 对比新旧，生成变更文件map。
   - 统计变更数量、总大小。
   - 用 `estimateEpisodes` 进行分包。
   - 用 `tree_builder.BuildTreeFromChanges` 生成文件树。
   - 聚合所有结果，返回 `BackupPreparationResult`。
3. 前端用结果渲染UI。

## 7. 设计亮点与健壮性
- **符号链接安全**：indexer遍历时自动跳过符号链接，防止死循环。
- **健壮的清单管理**：manifest_manager在清单缺失或损坏时自动新建空清单，保证流程不中断。
- **高可观测性**：所有关键步骤均有日志，进度实时推送前端。
- **极简API**：前端只需调用一个方法即可获得所有所需数据。

---

如需了解详细数据结构和方法签名，请直接查阅 backend/types/types.go 及各模块源码。
