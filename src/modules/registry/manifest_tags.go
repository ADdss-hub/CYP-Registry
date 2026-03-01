// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/cyp-registry/registry/src/pkg/response"
)

// TagData Tag信息（用于记录镜像标签对应的摘要及统计信息）
// Size 字段语义：镜像实际内容大小（所有层 size 之和），单位：字节，而不是 Manifest JSON 本身的大小。
type TagData struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType,omitempty"`
	Size      int64  `json:"size"`
}

// getTagData 获取Tag数据
func (r *Registry) getTagData(ctx context.Context, path string) (*TagData, error) {
	reader, _, err := r.storage.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var tagData TagData
	if err := json.Unmarshal(data, &tagData); err != nil {
		return nil, err
	}

	return &tagData, nil
}

// ListTags 列出所有Tag
// GET /v2/<name>/tags/list
func (r *Registry) ListTags(ctx context.Context, project string) ([]string, error) {
	// 1) 先取进程内索引（当前进程内所有 push 都会写入索引）
	indexTags := r.getTagsFromIndex(project)

	// 2) 再从底层存储扫描 tags 目录（用于进程重启后的历史 tag 恢复）
	// 使用更宽松的解析逻辑，兼容不同存储驱动返回的路径格式
	tagsPath := BuildManifestPath(project, "tags")

	entries, err := r.storage.List(ctx, tagsPath)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			// 没有落盘 tag：仍然返回索引中的 tag（如果有）
			return indexTags, nil
		}
		return nil, err
	}

	// 合并并去重
	seen := make(map[string]struct{}, len(entries)+len(indexTags))
	for _, t := range indexTags {
		if t != "" {
			seen[t] = struct{}{}
		}
	}

	for _, entry := range entries {
		// 典型返回示例（不同驱动可能略有差异）：
		// - "pat-test/small/manifests/tags/5mb"
		// - "pat-test/small/manifests/tags/5mb/"
		// - "5mb"

		// 1) 去掉末尾的 "/"
		name := strings.TrimSuffix(entry, "/")

		// 2) 只取最后一段（文件名或子目录名）
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			name = name[idx+1:]
		}

		// 3) 过滤掉无效条目（空名称）
		//    注意：不能过滤 "latest" 等合法标签名，否则前端/客户端将看不到这些标签
		if name == "" {
			continue
		}

		seen[name] = struct{}{}
	}

	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	// 保持稳定输出，便于分页与测试
	sort.Strings(tags)
	return tags, nil
}

// GetTag 获取指定Tag的Manifest信息
func (r *Registry) GetTag(ctx context.Context, project, tag string) (*TagData, error) {
	tagPath := BuildManifestPath(project, "tags/"+tag)
	return r.getTagData(ctx, tagPath)
}
