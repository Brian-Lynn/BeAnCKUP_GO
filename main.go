package main

import (
	"beanckup/backend/indexer"
	"beanckup/backend/manifest_manager"
	"beanckup/backend/packager"
	"beanckup/backend/resource_manager"
	"beanckup/backend/task_manager"
	"beanckup/backend/tree_builder"
	"beanckup/backend/types"
	"beanckup/backend/worker"
	"context"
	"crypto/rand"
	"embed"
	"math/big"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend
var assets embed.FS

// App 结构体
type App struct {
	ctx             context.Context
	manifestManager *manifest_manager.Manager
	indexer         *indexer.Manager
	resourceManager *resource_manager.Manager
	workerManager   *worker.Manager
	packager        *packager.Manager
	taskManager     *task_manager.Manager
}

// NewApp 创建新的应用实例
func NewApp() *App {
	manifestManager := manifest_manager.NewManager()
	indexer := indexer.NewManager()
	resourceManager := resource_manager.NewManager()
	workerManager := worker.NewManager()
	packager := packager.NewManager()
	taskManager := task_manager.NewManager(indexer, manifestManager, resourceManager, workerManager, packager)

	return &App{
		manifestManager: manifestManager,
		indexer:         indexer,
		resourceManager: resourceManager,
		workerManager:   workerManager,
		packager:        packager,
		taskManager:     taskManager,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SelectDirectory 允许用户选择一个目录
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "请选择一个目录"})
}

// ScanWorkspace 扫描工作区并返回真实的变更文件树
func (a *App) ScanWorkspace(path string) ([]*types.TreeNode, error) {
	previousManifest, err := a.manifestManager.LoadLatestManifest(path)
	if err != nil {
		return nil, err
	}
	currentFiles, err := a.indexer.ScanWorkspace(path)
	if err != nil {
		return nil, err
	}
	changedFiles := a.indexer.CompareWithManifest(currentFiles, previousManifest)
	treeNodes := tree_builder.BuildTreeFromChanges(changedFiles, path)
	return treeNodes, nil
}

// StartBackupPreparation 开始备份准备
func (a *App) StartBackupPreparation(workspacePath string, maxPackageSizeGB, maxTotalSizeGB float64) ([]*types.Episode, error) {
	return a.taskManager.StartBackupPreparation(workspacePath, maxPackageSizeGB, maxTotalSizeGB)
}

// StartBackupExecution 启动完整的备份流程
func (a *App) StartBackupExecution(workspacePath, deliveryPath string, maxPackageSizeGB, maxTotalSizeGB float64, password string) (*task_manager.BackupResult, error) {
	return a.taskManager.StartBackupExecution(workspacePath, deliveryPath, maxPackageSizeGB, maxTotalSizeGB, password, a.ctx)
}

// GeneratePassword 生成一个安全的随机密码
func (a *App) GeneratePassword() (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^&*"
	ret := make([]byte, 16)
	for i := 0; i < 16; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}

// CopyToClipboard 将文本复制到系统剪贴板
func (a *App) CopyToClipboard(text string) error {
	runtime.ClipboardSetText(a.ctx, text)
	return nil
}

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
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
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
