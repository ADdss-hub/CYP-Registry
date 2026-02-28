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
	"sort"
	"strconv"
	"strings"

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

// TagData Tag信息（用于记录镜像标签对应的摘要及统计信息）
// Size 字段语义：镜像实际内容大小（所有层 size 之和），单位：字节，而不是 Manifest JSON 本身的大小。
type TagData struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType,omitempty"`
	Size      int64  `json:"size"`
}

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

// DeleteManifest 删除Manifest
// DELETE /v2/<name>/manifests/<reference>
func (r *Registry) DeleteManifest(ctx context.Context, project, reference string) error {
	// 确定是tag还是digest
	isDigest, ref := ParseReference(reference)

	if isDigest {
		// digest 引用：先根据 digest 删除底层 manifest 文件，
		// 再清理所有引用该 digest 的 tag 映射，确保 /tags/list 及项目统计信息与实际存储一致。
		alg, hexDigest, err := ParseDigest(ref)
		if err != nil {
			return ErrManifestNotFound
		}

		// 删除真正的 manifest 内容（按 hexDigest 落盘）
		manifestPath := BuildManifestPath(project, hexDigest)
		if err := r.storage.Delete(ctx, manifestPath); err != nil {
			return err
		}

		// 清理所有引用该 digest 的 tag 映射
		tagsPath := BuildManifestPath(project, "tags")
		entries, err := r.storage.List(ctx, tagsPath)
		if err == nil {
			fullDigest := fmt.Sprintf("%s:%s", alg, hexDigest)
			for _, entry := range entries {
				name := strings.TrimSuffix(entry, "/")
				if idx := strings.LastIndex(name, "/"); idx != -1 {
					name = name[idx+1:]
				}
				if name == "" {
					continue
				}
				tagPath := BuildManifestPath(project, "tags/"+name)
				tagData, err := r.getTagData(ctx, tagPath)
				if err != nil {
					continue
				}
				if tagData.Digest == fullDigest || tagData.Digest == ref {
					_ = r.storage.Delete(ctx, tagPath)
					// 同步从内存索引中移除
					r.removeTag(project, name)
				}
			}
		}
	} else {
		// tag 删除：先找到对应的 digest，并删除 tag 映射与内存索引，
		// 再尝试删除底层 manifest（如果不再被其他 tag 引用）。
		tagPath := BuildManifestPath(project, "tags/"+ref)
		tagData, err := r.getTagData(ctx, tagPath)
		if err != nil {
			return ErrManifestNotFound
		}

		// 删除 tag 映射
		if err := r.storage.Delete(ctx, tagPath); err != nil {
			return err
		}
		r.removeTag(project, ref)

		// 尝试删除对应的 manifest（如果没有其他 tag 引用它）
		_, hexDigest, err := ParseDigest(tagData.Digest)
		if err == nil && hexDigest != "" {
			manifestPath := BuildManifestPath(project, hexDigest)
			// 为安全起见，先检查是否仍有其他 tag 指向该 digest
			tagsPath := BuildManifestPath(project, "tags")
			entries, listErr := r.storage.List(ctx, tagsPath)
			shouldDelete := true
			if listErr == nil {
				for _, entry := range entries {
					name := strings.TrimSuffix(entry, "/")
					if idx := strings.LastIndex(name, "/"); idx != -1 {
						name = name[idx+1:]
					}
					if name == "" {
						continue
					}
					otherTagPath := BuildManifestPath(project, "tags/"+name)
					otherData, tdErr := r.getTagData(ctx, otherTagPath)
					if tdErr == nil && otherData.Digest == tagData.Digest {
						shouldDelete = false
						break
					}
				}
			}
			if shouldDelete {
				_ = r.storage.Delete(ctx, manifestPath)
			}
		}
	}
	return nil
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

// BlobExists 检查Blob是否存在（带摘要验证）
func (r *Registry) BlobExists(ctx context.Context, project, digest string) (bool, int64, error) {
	exists, err := r.CheckBlob(ctx, project, digest)
	if err != nil || !exists {
		return exists, 0, err
	}

	size, err := r.GetBlobSize(ctx, project, digest)
	if err != nil {
		return false, 0, err
	}

	return true, size, nil
}

// ParseContentRange 解析Content-Range头
// 格式: bytes start-end/total
func ParseContentRange(contentRange string) (start, end, total int64, err error) {
	// 格式: bytes start-end/total
	parts := strings.SplitN(contentRange, " ", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("invalid content-range format: %s", contentRange)
	}

	rangePart := parts[1]
	parts = strings.SplitN(rangePart, "/", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("invalid content-range format: %s", contentRange)
	}

	rangeBytes := parts[0]
	total, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid total in content-range: %w", err)
	}

	parts = strings.SplitN(rangeBytes, "-", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("invalid range in content-range: %s", rangeBytes)
	}

	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid start in content-range: %w", err)
	}

	end, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid end in content-range: %w", err)
	}

	return start, end, total, nil
}
