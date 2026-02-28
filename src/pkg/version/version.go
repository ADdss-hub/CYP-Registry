package version

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Info 表示统一版本系统生成的当前版本信息
// 结构参考 unified-version-system 文档中的 .version/version.json 示例
type Info struct {
	Version     string `json:"version"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	Commit      string `json:"commit,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Project     string `json:"project,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

var (
	versionInfo Info
	once        sync.Once
)

// loadVersion 从项目根目录下的 .version/version.json 读取版本信息。
// 读取失败时不会导致应用启动失败，而是回退到开发占位版本。
func loadVersion() {
	const fallbackVersion = "v0.0.0-dev"

	wd, err := os.Getwd()
	if err != nil {
		log.Printf("version: 获取工作目录失败，将使用占位版本 %s: %v", fallbackVersion, err)
		versionInfo.Version = fallbackVersion
		return
	}

	versionPath := filepath.Join(wd, ".version", "version.json")
	data, err := os.ReadFile(versionPath)
	if err != nil {
		log.Printf("version: 读取版本文件失败（%s），将使用占位版本 %s: %v", versionPath, fallbackVersion, err)
		versionInfo.Version = fallbackVersion
		return
	}

	if err := json.Unmarshal(data, &versionInfo); err != nil {
		log.Printf("version: 解析版本文件失败，将使用占位版本 %s: %v", fallbackVersion, err)
		versionInfo.Version = fallbackVersion
		return
	}

	if versionInfo.Version == "" {
		log.Printf("version: 版本文件中缺少 version 字段，将使用占位版本 %s", fallbackVersion)
		versionInfo.Version = fallbackVersion
	}
}

// GetInfo 返回当前版本信息（只读拷贝）。
func GetInfo() Info {
	once.Do(loadVersion)
	return versionInfo
}

// GetVersion 返回当前版本号字符串。
func GetVersion() string {
	return GetInfo().Version
}
