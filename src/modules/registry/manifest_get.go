// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/cyp-registry/registry/src/pkg/response"
)

// GetManifest 获取Manifest
// GET /v2/<name>/manifests/<reference>
func (r *Registry) GetManifest(ctx context.Context, project, reference string) (*Manifest, string, error) {
	data, digest, err := r.GetManifestRaw(ctx, project, reference)
	if err != nil {
		return nil, "", err
	}

	// 解析Manifest
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 设置媒体类型（如果未设置）
	if manifest.MediaType == "" {
		manifest.MediaType = MediaTypeDocker2Manifest
	}

	return &manifest, digest, nil
}

// GetManifestRaw 获取原始 Manifest 数据（不解析）
// 返回原始字节、digest 和 error
func (r *Registry) GetManifestRaw(ctx context.Context, project, reference string) ([]byte, string, error) {
	// 确定是tag还是digest
	isDigest, ref := ParseReference(reference)

	var path string
	if isDigest {
		// digest 引用：存储层按 hexDigest（不含 sha256: 前缀）落盘
		// 这里必须去掉前缀，否则会读取到不存在的路径并导致 Docker 客户端在 HEAD manifests/digest 时失败。
		_, hexDigest, err := ParseDigest(ref)
		if err != nil {
			return nil, "", ErrManifestNotFound
		}
		path = BuildManifestPath(project, hexDigest)
	} else {
		// tag需要找到对应的digest
		tagPath := BuildManifestPath(project, "tags/"+ref)
		tagData, err := r.getTagData(ctx, tagPath)
		if err != nil {
			return nil, "", ErrManifestNotFound
		}
		// 通过 tag 映射到真实 digest 的 manifest 路径
		// tagData.Digest 格式为 "sha256:xxx"，需要提取 hex 部分
		_, hexDigest, err := ParseDigest(tagData.Digest)
		if err != nil {
			return nil, "", fmt.Errorf("invalid digest in tag data: %w", err)
		}
		path = BuildManifestPath(project, hexDigest)
	}

	// 获取Manifest内容
	reader, _, err := r.storage.Get(ctx, path)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, "", ErrManifestNotFound
		}
		return nil, "", err
	}

	// 读取内容
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read manifest: %w", err)
	}

	// 计算摘要
	digest, _, err := CalculateDigest(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}

	return data, digest, nil
}

// GetManifestByDigest 根据digest获取Manifest
func (r *Registry) GetManifestByDigest(ctx context.Context, project, digest string) (*Manifest, error) {
	_, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return nil, err
	}

	path := BuildManifestPath(project, hexDigest)
	reader, _, err := r.storage.Get(ctx, path)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, ErrManifestNotFound
		}
		return nil, err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}
