package packager

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"beanckup/backend/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Packager 打包器接口
type Packager interface {
	// 使用7zr.exe创建压缩包
	CreateArchiveWith7zr(filesToPack []*types.FileInfo, targetPath string, workspacePath string, password string) error

	// 获取打包进度
	GetPackProgress() float64

	// 获取打包状态
	GetPackStatus() string
}

// Manager 打包器实现
type Manager struct {
	packProgress float64
	packStatus   string
	packedFiles  int
	totalFiles   int
	ctx          context.Context
}

// NewManager 创建新的打包器
func NewManager() *Manager {
	return &Manager{
		packProgress: 0.0,
		packStatus:   "就绪",
	}
}

// CreateArchiveWith7zr 使用7zr.exe创建压缩包
func (m *Manager) CreateArchiveWith7zr(filesToPack []*types.FileInfo, targetPath string, workspacePath string, password string) error {
	if len(filesToPack) == 0 {
		return fmt.Errorf("没有文件需要打包到 %s", filepath.Base(targetPath))
	}

	// 7zr.exe应该和主程序在同一目录下
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取程序路径: %w", err)
	}
	sevenZipPath := filepath.Join(filepath.Dir(executablePath), "7zr.exe")
	if _, err := os.Stat(sevenZipPath); os.IsNotExist(err) {
		return fmt.Errorf("关键组件丢失: 7zr.exe 未在程序目录找到")
	}

	// 创建临时文件列表
	tempFile, err := os.CreateTemp("", "beanckup_filelist_*.txt")
	if err != nil {
		return fmt.Errorf("创建临时文件列表失败: %w", err)
	}
	defer os.Remove(tempFile.Name())

	writer := bufio.NewWriter(tempFile)
	for _, file := range filesToPack {
		// 写入文件的绝对路径
		if _, err := writer.WriteString(file.Path + "\n"); err != nil {
			tempFile.Close()
			return fmt.Errorf("写入文件列表失败: %w", err)
		}
	}
	writer.Flush()
	tempFile.Close()

	// 构建7zr命令参数
	args := []string{
		"a",                   // 添加到压缩包
		"-t7z",                // 明确使用7z格式
		targetPath,            // 输出的压缩包完整路径
		"@" + tempFile.Name(), // 从文件列表读取要压缩的文件
	}
	if password != "" {
		args = append(args, "-p"+password, "-mhe=on") // 如果有密码，则添加密码并加密头部
	}

	// 创建命令
	cmd := exec.Command(sevenZipPath, args...)
	// **核心修复**: 设置正确的工作目录，让压缩包内的目录结构是相对于工作区的
	cmd.Dir = workspacePath

	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果7zr执行失败，返回包含详细输出的错误信息
		return fmt.Errorf("7zr执行失败: %w, 输出: %s", err, string(output))
	}

	return nil
}

// createTempFileListWithAbsolutePaths 创建包含绝对路径的临时文件列表
func (m *Manager) createTempFileListWithAbsolutePaths(filesToPack []*types.FileInfo, tempDir string) (string, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "beanckup_filelist_*.txt")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// 写入文件路径
	writer := bufio.NewWriter(tempFile)

	// 先写入manifest（相对路径）
	writer.WriteString("manifest.json\n")

	// 再写入用户文件（绝对路径）
	for _, file := range filesToPack {
		if _, err := writer.WriteString(file.Path + "\n"); err != nil {
			return "", err
		}
	}

	if err := writer.Flush(); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

// readOutput 读取7zr的输出并解析进度
func (m *Manager) readOutput(stdout, stderr io.ReadCloser) {
	// 读取标准输出
	go func() {
		scanner := bufio.NewScanner(stdout)
		startTime := time.Now()

		for scanner.Scan() {
			line := scanner.Text()

			// 解析进度信息
			if progress := m.parseProgress(line); progress > 0 {
				m.packProgress = progress
				m.packedFiles = int(progress * float64(m.totalFiles))

				// 计算速度和预估时间
				elapsed := time.Since(startTime).Seconds()
				speed := 0.0
				estimated := 0.0

				if elapsed > 0 {
					speed = float64(m.packedFiles) / elapsed // 文件/秒
					if progress > 0 {
						estimated = elapsed / progress * (1 - progress)
					}
				}

				// 推送进度事件
				if m.ctx != nil {
					runtime.EventsEmit(m.ctx, "backup-progress", map[string]interface{}{
						"progress":      progress,
						"speed":         speed,
						"elapsedTime":   int64(elapsed),
						"estimatedTime": int64(estimated),
						"phase":         "压缩中",
					})
				}
			}
		}
	}()

	// 读取标准错误，只显示错误信息
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// 只打印包含Error或WARNING的行
			if strings.Contains(strings.ToUpper(line), "ERROR") || strings.Contains(strings.ToUpper(line), "WARNING") {
				fmt.Printf("7zr stderr: %s\n", line)
			}
		}
	}()
}

// parseProgress 解析7z输出中的进度信息
func (m *Manager) parseProgress(line string) float64 {
	// 7z输出格式示例：
	// " 45% = 1234567890" 或 "45%" 或 "Compressing  filename.ext"

	// 尝试匹配百分比
	percentRegex := regexp.MustCompile(`(\d+)%`)
	matches := percentRegex.FindStringSubmatch(line)
	if len(matches) >= 2 {
		if percent, err := strconv.Atoi(matches[1]); err == nil {
			return float64(percent) / 100.0
		}
	}

	// 如果没有找到百分比，尝试通过文件计数估算
	if strings.Contains(line, "Compressing") || strings.Contains(line, "Adding") {
		m.packedFiles++
		if m.totalFiles > 0 {
			return float64(m.packedFiles) / float64(m.totalFiles)
		}
	}

	return 0.0
}

// GetPackProgress 获取打包进度
func (m *Manager) GetPackProgress() float64 {
	return m.packProgress
}

// GetPackStatus 获取打包状态
func (m *Manager) GetPackStatus() string {
	return m.packStatus
}

// Validate7zrPath 验证7zr.exe路径
func (m *Manager) Validate7zrPath(sevenZipPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(sevenZipPath); os.IsNotExist(err) {
		return fmt.Errorf("7zr.exe不存在: %s", sevenZipPath)
	}

	// 尝试执行7zr --help来验证
	cmd := exec.Command(sevenZipPath, "--help")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("7zr.exe无法执行: %w", err)
	}

	return nil
}
