// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/cyp-registry/registry/src/pkg/response"
)

// ManifestReferrers 获取引用此Manifest的列表
// GET /v2/<name>/manifests/<digest>/referrers
func (r *Registry) ManifestReferrers(ctx context.Context, project, digest string) ([]Descriptor, error) {
	// 验证摘要格式
	_, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return nil, err
	}

	// 查找引用此manifest的列表
	referrersPath := BuildManifestPath(project, "referrers/"+hexDigest)

	reader, _, err := r.storage.Get(ctx, referrersPath)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return []Descriptor{}, nil
		}
		return nil, err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var referrers []Descriptor
	if err := json.Unmarshal(data, &referrers); err != nil {
		return nil, err
	}

	return referrers, nil
}

// Catalog 列出所有仓库
// GET /v2/_catalog
func (r *Registry) Catalog(ctx context.Context, n int, last string) ([]string, error) {
	// 列出顶级目录
	entries, err := r.storage.List(ctx, "")
	if err != nil {
		return nil, err
	}

	var repos []string
	seen := make(map[string]bool)

	for _, entry := range entries {
		// 清理路径
		entry = strings.TrimPrefix(entry, "/")
		parts := strings.SplitN(entry, "/", 2)

		// 只取第一级目录（项目名）
		if len(parts) > 0 {
			project := parts[0]
			if !seen[project] && project != "" {
				// 跳过非项目目录
				if strings.Contains(project, "manifests") || strings.Contains(project, "blobs") ||
					strings.Contains(project, "uploads") {
					continue
				}
				seen[project] = true
				repos = append(repos, project)
			}
		}
	}

	// 应用分页
	if n > 0 && len(repos) > n {
		offset := 0
		if last != "" {
			// 找到last的位置
			for i, repo := range repos {
				if repo == last {
					offset = i + 1
					break
				}
			}
		}
		if offset >= len(repos) {
			return []string{}, nil
		}
		if offset+n > len(repos) {
			return repos[offset:], nil
		}
		return repos[offset : offset+n], nil
	}

	return repos, nil
}

// ListProjectManifests 列出项目中的所有Manifest（按digest）
func (r *Registry) ListProjectManifests(ctx context.Context, project string) ([]string, error) {
	manifestsPath := BuildManifestPath(project, "")

	entries, err := r.storage.List(ctx, manifestsPath)
	if err != nil {
		return nil, err
	}

	var manifests []string
	for _, entry := range entries {
		if strings.Contains(entry, "manifests/") {
			digest := strings.TrimPrefix(entry, manifestsPath+"/")
			if digest != "" && digest != "tags" && digest != "referrers" {
				manifests = append(manifests, digest)
			}
		}
	}

	return manifests, nil
}
