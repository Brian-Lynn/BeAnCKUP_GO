# BeAnCKUP_GO 技术架构总览

本文档旨在提供 BeAnCKUP_GO 项目的详细技术剖析，包括其架构设计、核心模块、数据流程以及代码修改指引。
## 1. 高层架构
BeAnCKUP_GO 采用 **Wails 框架**，实现了 **Go 后端 + Web 前端** 的经典分离架构。
- **后端 (Go)**: 负责所有重型和核心的业务逻辑，包括文件 I/O、并发处理、数据计算和状态管理。其高性能和并发模型是项目实现高效备份的关键。
- **前端 (HTML/JS)**: 负责用户界面和交互。它是一个纯粹的视图层，通过 Wails 提供的 JavaScript Bridge 调用后端 Go 语言暴露的方法来触发业务逻辑，并通过 Wails Events 接收来自后端的实时状态更新（如进度条）。
**核心设计理念**:
整个备份过程被抽象为一个**任务驱动的并发流水线（Task-Driven Concurrent Pipeline）**。用户在 UI 上的操作会创建一个“备份任务”，该任务随后经过一系列定义清晰、职责单一的模块化处理器，最终完成备份。
## 2. 数据流：一次完整的备份过程
理解数据如何流经系统是理解整个架构的关键。
1.  **UI 触发**: 用户在前端界面选择源/目标目录后点击“备份”。前端 JavaScript 调用 `window.go.main.App.StartBackup(source, target)`。

2.  **任务启动 (`main.go`, `task_manager`)**:
    - `main.go` 中的 `App` 结构体是前端和后端的桥梁。`StartBackup` 方法接收到请求后，会实例化并启动一个 `task_manager.TaskManager`。
    - `TaskManager` 是整个备份流程的总指挥。

3.  **加载旧状态 (`state_manager`)**:
    - `TaskManager` 首先命令 `state_manager.StateManager` 去目标目录寻找并加载上一次成功备份的清单（`manifest.json`）。
    - 如果不存在，则判定为首次备份（全量备份）；否则，为增量备份。

4.  **文件索引 (`indexer`)**:
    - `TaskManager` 启动 `indexer.Indexer`。
    - `Indexer` 递归扫描源目录。对于每个文件，它会与从 `StateManager` 加载的旧清单进行比对。
    - **比对逻辑**: 通常基于文件修改时间（ModTime）和大小（Size）。如果文件是新增的，或者修改时间/大小发生了变化，它将被标记为“待处理”。
    - **输出**: `Indexer` 生成一个“待处理文件列表”。

5.  **并发处理 (`worker`, `file_processor`)**:
    - `TaskManager` 根据 `resource_manager` 获取的 CPU 核心数，初始化一个 `worker.WorkerPool`（并发工作池）。
    - `Indexer` 产出的“待处理文件列表”被作为任务项，分发给 `WorkerPool` 中的各个 Worker (goroutine)。
    - 每个 `Worker` 内部使用 `file_processor.FileProcessor` 来执行具体工作：
        - 读取文件内容。
        - 计算文件内容的哈希值（例如 SHA256），以确保数据完整性。
    - **输出**: 每个 `Worker` 处理完一个文件后，会产出一个包含文件路径、哈希值和原始文件数据的数据结构。

6.  **文件打包 (`packager`)**:
    - `packager.Packager` 从 `Worker` 的输出中接收处理好的文件数据。
    - 它会将这些数据块（可能是整个文件或文件的一部分）追加写入到目标目录下的一个或多个大型 `.pack` 文件中。
    - **目的**: 避免在目标目录中创建成千上万个小文件，提高 I/O 效率和文件系统性能。
    - **输出**: `.pack` 文件，以及每个数据块在 `.pack` 文件中的偏移量和大小。

7.  **构建新清单 (`tree_builder`, `manifest_manager`)**:
    - 在所有文件处理和打包的同时，`tree_builder.TreeBuilder` 会根据扫描到的文件信息，在内存中构建一棵完整的、代表源目录结构的文件树。
    - 当所有 `Worker` 完成工作后，`TaskManager` 指示 `manifest_manager.ManifestManager` 开始工作。
    - `ManifestManager` 接收 `TreeBuilder` 构建的目录树，并整合 `Packager` 提供的“文件块-Pack位置”映射关系。
    - **输出**: 在目标目录中生成一个新的 `manifest.json` 文件。该文件是本次备份的完整快照，精确记录了每个文件的元数据、哈希及其在 `.pack` 文件中的存储信息。

8.  **状态更新 (`state_manager`)**:
    - `ManifestManager` 成功写入新清单后，`TaskManager` 会通知 `StateManager` 本次备份成功。`StateManager` 会将新的清单视为下一次增量备份的基准。

9.  **通知前端**: 在整个流程中，`TaskManager` 会通过 Wails Events 系统将进度、日志等信息实时发送给前端，用于更新 UI。

## 3. 核心模块职责详解

- **`main.go`**:
    - **职责**: 程序入口。定义 `App` 结构体，作为 JS-Go 的通信接口。初始化并运行 Wails 应用。
    - **修改场景**: 需要向前端暴露新的 API（在 `App` 上添加新方法），或修改应用启动配置。

- **`backend/types/types.go`**:
    - **职责**: 定义全局共享的数据结构，如 `Manifest`, `FileNode`, `TaskInfo` 等。是整个项目的数据契约。
    - **修改场景**: 需要在清单中添加新字段，或修改核心数据结构时。

- **`backend/task_manager/task_manager.go`**:
    - **职责**: 流程编排。创建和协调所有其他后端模块，确保它们按正确的顺序执行。
    - **修改场景**: 需要改变备份/恢复的整体流程，例如增加一个新的处理步骤（如加密）。

- **`backend/indexer/indexer.go`**:
    - **职责**: 决定哪些文件需要备份。核心是“变更检测”逻辑。
    - **修改场景**: 修复“文件未被备份”或“文件被不必要地重复备份”等 Bug。修改文件比对逻辑（例如，从“时间戳”比对改为“哈希”比对）。

- **`backend/worker/worker.go` & `backend/file_processor/file_processor.go`**:
    - **职责**: `worker` 负责并发调度，`file_processor` 负责具体的文件操作（读文件、算哈希）。这是 CPU 密集型操作的核心。
    - **修改场景**: 优化性能（调整并发数、改进文件读取方式），或者改变文件哈希算法。

- **`backend/packager/packager.go`**:
    - **职责**: 将零散的文件数据写入大的归档文件。
    - **修改场景**: 改变打包策略（例如，设置 pack 文件的大小上限），或引入压缩功能。

- **`backend/manifest_manager/manifest_manager.go`**:
    - **职责**: 生成和解析 `manifest.json`。这是数据一致性和恢复能力的心脏。
    - **修改场景**: 清单格式发生变化（增删字段），或修复因清单损坏/解析错误导致的恢复失败问题。

- **`backend/state_manager/state_manager.go`**:
    - **职责**: 持久化和加载两次备份之间的状态（即上一次的清单）。
    - **修改场景**: 改变状态文件的存储位置或格式。

## 4. 代码修改指引

- **UI/交互 Bug**:
    1.  检查 `frontend/index.html` 的元素布局。
    2.  检查 `index.html` 内的 `<script>` 标签或关联的 `.js` 文件，查看前端逻辑。
    3.  跟踪调用的 `window.go.main.App.MethodName`，检查 `main.go` 中对应方法的逻辑是否正确。

- **文件没有被备份**:
    1.  **首要怀疑对象**: `backend/indexer/indexer.go`。
    2.  **排查**: Debug 其文件比对逻辑，确认目标文件是否被正确识别为“待处理”。检查 `ModTime` 和 `Size` 的比较是否符合预期。

- **备份性能问题**:
    1.  **检查点**: `backend/worker/worker.go` 和 `backend/resource_manager/resource_manager.go`。确认并发数设置是否合理。
    2.  **检查点**: `backend/file_processor/file_processor.go`。分析文件读取和哈希计算部分是否有优化空间。
    3.  **检查点**: `backend/packager/packager.go`。检查文件写入逻辑是否成为瓶颈。

- **恢复失败/数据损坏**:
    1.  **首要怀疑对象**: `backend/manifest_manager/manifest_manager.go`。
    2.  **排查**: 检查 `manifest.json` 的生成逻辑是否完整记录了所有必要信息。检查恢复时（如果已实现）解析清单的逻辑是否正确。
    3.  **排查**: 检查 `backend/packager/packager.go`，确认数据写入 `.pack` 文件时是否准确无误。

- **添加新功能（例如，备份前加密文件）**:
    1.  在 `backend/types/types.go` 中为 `FileNode` 添加加密相关的字段（如 `EncryptionMethod`, `Salt`）。
    2.  在 `backend/file_processor/file_processor.go` 的处理逻辑中，增加一步加密操作。
    3.  相应地，在 `backend/manifest_manager/manifest_manager.go` 中将加密信息写入清单。
    4.  最后，在恢复流程中添加对应的解密步骤。
