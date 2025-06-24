package tree_builder

import (
	"path/filepath"
	"strings"

	"beanckup/backend/types"
)

// BuildTreeFromChanges 将变更文件列表构建成树状结构
func BuildTreeFromChanges(changedFiles map[string]*types.FileInfo, workspacePath string) []*types.TreeNode {
	// 创建根节点映射
	rootNodes := make(map[string]*types.TreeNode)

	// 遍历所有变更文件
	for filePath, fileInfo := range changedFiles {
		// 获取相对于工作区的路径
		relPath, err := filepath.Rel(workspacePath, filePath)
		if err != nil {
			// 如果无法获取相对路径，跳过该文件
			continue
		}

		// 分割路径
		pathParts := strings.Split(relPath, string(filepath.Separator))

		// 构建目录树
		currentPath := ""
		var parentNode *types.TreeNode

		// 处理目录部分
		for i := 0; i < len(pathParts)-1; i++ {
			part := pathParts[i]
			if part == "" {
				continue
			}

			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			// 检查目录节点是否已存在
			if parentNode == nil {
				// 根级目录
				if node, exists := rootNodes[part]; exists {
					parentNode = node
				} else {
					parentNode = &types.TreeNode{
						Name:     part,
						Path:     filepath.Join(workspacePath, currentPath),
						IsDir:    true,
						Status:   types.StatusUnchanged,
						Children: make([]*types.TreeNode, 0),
					}
					rootNodes[part] = parentNode
				}
			} else {
				// 查找或创建子目录
				var childNode *types.TreeNode
				for _, child := range parentNode.Children {
					if child.Name == part && child.IsDir {
						childNode = child
						break
					}
				}

				if childNode == nil {
					childNode = &types.TreeNode{
						Name:     part,
						Path:     filepath.Join(workspacePath, currentPath),
						IsDir:    true,
						Status:   types.StatusUnchanged,
						Children: make([]*types.TreeNode, 0),
					}
					parentNode.Children = append(parentNode.Children, childNode)
				}
				parentNode = childNode
			}
		}

		// 创建文件节点
		fileName := pathParts[len(pathParts)-1]
		fileNode := &types.TreeNode{
			Name:   fileName,
			Path:   filePath,
			IsDir:  false,
			Status: fileInfo.Status,
		}

		// 将文件节点添加到父目录
		if parentNode == nil {
			// 文件在根目录
			if node, exists := rootNodes[fileName]; exists {
				// 如果根节点已存在且是目录，添加文件到该目录
				if node.IsDir {
					node.Children = append(node.Children, fileNode)
				} else {
					// 替换根节点
					rootNodes[fileName] = fileNode
				}
			} else {
				rootNodes[fileName] = fileNode
			}
		} else {
			// 检查是否已存在同名文件
			fileExists := false
			for i, child := range parentNode.Children {
				if child.Name == fileName && !child.IsDir {
					parentNode.Children[i] = fileNode
					fileExists = true
					break
				}
			}

			if !fileExists {
				parentNode.Children = append(parentNode.Children, fileNode)
			}
		}
	}

	// 将根节点映射转换为切片
	var result []*types.TreeNode
	for _, node := range rootNodes {
		result = append(result, node)
	}

	return result
}
