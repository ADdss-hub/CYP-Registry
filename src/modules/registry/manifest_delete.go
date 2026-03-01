// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"context"
	"fmt"
	"strings"
)

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
