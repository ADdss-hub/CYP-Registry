// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// PutManifest 上传Manifest
// PUT /v2/<name>/manifests/<reference>
// 注意：manifest 参数仅用于获取 MediaType，实际存储使用 rawData
func (r *Registry) PutManifest(ctx context.Context, project, reference string, manifest *Manifest) (string, error) {
	// 序列化Manifest
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// 计算摘要（基于 Manifest JSON 本身）
	digest, size, err := CalculateDigest(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	// 计算镜像实际大小：累加所有层的 size，若计算失败则退回为 Manifest JSON 大小
	imageSize := size
	if manifest != nil && len(manifest.Layers) > 0 {
		var total int64
		for _, layer := range manifest.Layers {
			// 兼容 size 字段为 0 或未设置的情况
			if layer.Size > 0 {
				total += layer.Size
			}
		}
		if total > 0 {
			imageSize = total
		}
	}

	// 存储Manifest
	manifestPath := BuildManifestPath(project, reference)
	reader := bytes.NewReader(data)
	if err := r.storage.Put(ctx, manifestPath, reader, size); err != nil {
		return "", fmt.Errorf("failed to store manifest: %w", err)
	}

	// 如果是tag（非digest引用），更新tag映射
	isDigest, _ := ParseReference(reference)
	if !isDigest {
		tagPath := BuildManifestPath(project, "tags/"+reference)
		tagData := TagData{
			Digest:    digest,
			MediaType: manifest.MediaType,
			Size:      imageSize,
		}
		tagDataBytes, _ := json.Marshal(tagData)
		_ = r.storage.Put(ctx, tagPath, bytes.NewReader(tagDataBytes), int64(len(tagDataBytes)))

		// 更新latest标签
		latestPath := BuildManifestPath(project, "tags/latest")
		_ = r.storage.Put(ctx, latestPath, bytes.NewReader(tagDataBytes), int64(len(tagDataBytes)))

		// 同步写入内存 tag 索引，供 /v2/<name>/tags/list 使用
		r.addTag(project, reference)
	}

	return digest, nil
}

// PutManifestRaw 上传Manifest（使用原始字节）
// PUT /v2/<name>/manifests/<reference>
// 使用原始请求体计算 digest，避免重新序列化导致的 digest 不匹配
func (r *Registry) PutManifestRaw(ctx context.Context, project, reference string, rawData []byte, mediaType string) (string, error) {
	// 使用原始数据计算摘要
	digest, size, err := CalculateDigest(bytes.NewReader(rawData))
	if err != nil {
		return "", err
	}

	// 默认使用 Manifest JSON 大小时长；若能成功解析 Manifest，则改为镜像层总大小
	imageSize := size
	var manifest Manifest
	if err := json.Unmarshal(rawData, &manifest); err == nil && len(manifest.Layers) > 0 {
		var total int64
		for _, layer := range manifest.Layers {
			if layer.Size > 0 {
				total += layer.Size
			}
		}
		if total > 0 {
			imageSize = total
		}
	}

	// 提取 digest 的 hex 部分（去掉 sha256: 前缀）
	_, hexDigest, _ := ParseDigest(digest)

	// 始终以 digest 路径存储 manifest（这是规范要求的）
	digestPath := BuildManifestPath(project, hexDigest)
	if err := r.storage.Put(ctx, digestPath, bytes.NewReader(rawData), size); err != nil {
		return "", fmt.Errorf("failed to store manifest: %w", err)
	}

	// 如果是tag（非digest引用），更新tag映射
	isDigest, _ := ParseReference(reference)
	if !isDigest {
		// 版本号标签：若已存在则禁止覆盖（仅允许新增），例如 v1.0.0
		if isImmutableTag(reference) {
			tagPath := BuildManifestPath(project, "tags/"+reference)
			if _, _, err := r.storage.Get(ctx, tagPath); err == nil {
				return "", ErrImmutableTag
			}
		}

		tagPath := BuildManifestPath(project, "tags/"+reference)
		tagData := TagData{
			Digest:    digest,
			MediaType: mediaType,
			Size:      imageSize,
		}
		tagDataBytes, _ := json.Marshal(tagData)
		_ = r.storage.Put(ctx, tagPath, bytes.NewReader(tagDataBytes), int64(len(tagDataBytes)))

		// 同步写入内存 tag 索引，供 /v2/<name>/tags/list 使用
		r.addTag(project, reference)
	}

	return digest, nil
}
