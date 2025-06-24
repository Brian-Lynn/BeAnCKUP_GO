package tree_builder

import (
	"beanckup/backend/types"
	"path/filepath"
	"sort"
	"strings"
)

// BuildTreeFromChanges 根据变更文件列表构建UI所需的文件树
func BuildTreeFromChanges(changedFiles map[string]*types.FileInfo, basePath string) []*types.TreeNode {
	nodes := make(map[string]*types.TreeNode)
	var rootNodes []*types.TreeNode

	// 确保所有父目录都存在
	for path, info := range changedFiles {
		// 先为文件自身创建一个节点
		if _, exists := nodes[path]; !exists {
			nodes[path] = &types.TreeNode{
				Name:   info.Name,
				Path:   path,
				IsDir:  false,
				Status: info.Status,
			}
		}

		// 逐级创建父目录节点
		currentPath := filepath.Dir(path)
		for {
			relPath, err := filepath.Rel(basePath, currentPath)
			if err != nil || relPath == "." || relPath == ".." {
				break
			}

			if _, exists := nodes[currentPath]; !exists {
				nodes[currentPath] = &types.TreeNode{
					Name:     filepath.Base(currentPath),
					Path:     currentPath,
					IsDir:    true,
					Status:   types.StatusUnchanged, // 目录本身不标记状态，其状态由子节点决定
					Children: make([]*types.TreeNode, 0),
				}
			}

			if currentPath == basePath {
				break
			}

			currentPath = filepath.Dir(currentPath)
		}
	}

	// 将节点连接成树状结构
	for path, node := range nodes {
		parentPath := filepath.Dir(path)
		parent, parentExists := nodes[parentPath]

		if parentExists && parentPath != path {
			parent.Children = append(parent.Children, node)
		} else {
			// 如果没有父节点，说明它是一个根节点
			rootNodes = append(rootNodes, node)
		}
	}

	// 对每个层级的子节点进行排序
	for _, node := range nodes {
		if node.IsDir {
			sortNodes(node.Children)
		}
	}
	sortNodes(rootNodes)

	return rootNodes
}

// sortNodes 对树节点进行排序：文件夹在前，文件在后，同类型按名称排序
func sortNodes(nodes []*types.TreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir // isDir (true) > !isDir (false), so folders come first
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
}
