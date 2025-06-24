package main

import (
	"beanckup/backend/indexer"
	"beanckup/backend/task_manager"
	"context"
	"embed"
	"log"
	"os"

	"beanckup/backend/types"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend
var assets embed.FS

// App 结构体是程序的核心，负责处理所有前端的调用
type App struct {
	ctx         context.Context
	taskManager *task_manager.Manager
}

// NewApp 创建一个新的 App 实例
func NewApp() *App {
	// 在这里进行所有后端模块的初始化
	// TODO: 在重构其他模块时，会在这里添加初始化逻辑
	taskManager := task_manager.NewManager()
	return &App{
		taskManager: taskManager,
	}
}

// startup 在应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("BeanCKUP App started successfully.")
}

// SelectDirectory 打开一个对话框让用户选择目录
func (a *App) SelectDirectory() (string, error) {
	log.Println("Frontend called: SelectDirectory")
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "请选择一个目录"})
}

// ScanWorkspace 扫描工作区以显示文件变更树
// 这是一个轻量级操作，用于UI展示
func (a *App) ScanWorkspace(path string) (interface{}, error) {
	log.Printf("Frontend called: ScanWorkspace with path: %s\n", path)

	indexer := indexer.NewManager()

	// 定义进度回调函数
	progressCallback := func(processedCount int, totalCount int) {
		// 向前端发送进度事件
		runtime.EventsEmit(a.ctx, "scan-progress", map[string]interface{}{
			"processed": processedCount,
			"total":     totalCount,
		})
		log.Printf("Scan progress: %d/%d", processedCount, totalCount)
	}

	// 执行扫描
	_, err := indexer.ScanWorkspace(path, progressCallback)
	if err != nil {
		log.Printf("Error during workspace scan: %v", err)
		return nil, err
	}

	// TODO: 调用 tree_builder 将扫描结果转换为文件树
	// TODO: 加载旧清单并与当前扫描结果对比，以确定文件状态

	log.Println("ScanWorkspace finished.")
	return nil, nil
}

// StartBackupPreparation 接收备份参数，进行预处理
// 这是一个重量级操作，对应"首次扫描"
func (a *App) StartBackupPreparation(workspacePath string, maxPackageSizeGB, maxTotalSizeGB float64) (*types.BackupPreparationResult, error) {
	log.Printf("Frontend called: StartBackupPreparation with workspace: %s\n", workspacePath)
	return a.taskManager.StartBackupPreparation(workspacePath, maxPackageSizeGB, maxTotalSizeGB)
}

// StartBackupExecution 启动实际的备份流程
func (a *App) StartBackupExecution(workspacePath, deliveryPath string, maxPackageSizeGB, maxTotalSizeGB float64, password string) (interface{}, error) {
	log.Printf("Frontend called: StartBackupExecution with workspace: %s, deliveryPath: %s\n", workspacePath, deliveryPath)
	// TODO: 调用重构后的 task_manager 模块
	// 暂时返回空数据
	return nil, nil
}

// CopyToClipboard 将文本复制到系统剪贴板
func (a *App) CopyToClipboard(text string) {
	log.Println("Frontend called: CopyToClipboard")
	runtime.ClipboardSetText(a.ctx, text)
}

func main() {
	// 设置日志输出到文件
	logFile, err := os.OpenFile("beanckup.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("无法打开日志文件: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println("Application starting...")

	app := NewApp()

	err = wails.Run(&options.App{
		Title:  "BeAnCKUP",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		OnShutdown: func(ctx context.Context) {
			log.Println("Application shutting down.")
		},
	})

	if err != nil {
		log.Fatalln("Error:", err.Error())
	}
}
