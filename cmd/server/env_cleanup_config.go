package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadDotEnvDefaults 从指定的 .env 文件中加载环境变量：
// - 仅在当前进程环境中“未显式设置”的键上生效（与单镜像入口脚本保持一致）
// - 支持 "export KEY=VAL"、KEY=VAL、KEY="VAL" 等常见格式
func loadDotEnvDefaults(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		if strings.HasPrefix(strings.ToLower(key), "export ") {
			key = strings.TrimSpace(key[len("export "):])
		}
		if key == "" {
			continue
		}

		// 去掉可能存在的引号
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		val = strings.TrimSpace(val)

		// 不覆盖已存在的环境变量
		if os.Getenv(key) == "" && val != "" {
			_ = os.Setenv(key, val)
		}
	}
}

// detectCleanupConfig 检测 CLEANUP_ON_SHUTDOWN 配置来源与值：
// - 同时读取进程环境变量与根级 .env（全局配置中心）
// - 如两处均设置且不一致，按照“环境变量优先”生效并返回冲突状态
// 返回值：
//   - cleanupEnv: 实际生效的值（优先环境变量）
//   - shouldCleanup: 是否启用清理模式（仅当 cleanupEnv == "1" 时为 true）
//   - source: "env" | ".env" | "env+.env" | "none"
//   - conflict: 当同时配置且取值不同时时为 true
//   - dotEnvVal: .env 中解析到的值（无则为空）
func detectCleanupConfig() (cleanupEnv string, shouldCleanup bool, source string, conflict bool, dotEnvVal string) {
	// 1. 读取进程环境变量
	envVal := strings.TrimSpace(os.Getenv("CLEANUP_ON_SHUTDOWN"))

	// 2. 读取 .env（全局配置中心）
	//    为兼容不同运行方式（直接运行二进制 / Docker 容器 / Windows 容器），尝试多个候选路径：
	//    - 当前工作目录下的 .env
	//    - 可执行文件所在目录下的 .env
	//    - 容器内约定的 /app/.env（单镜像镜像默认工作目录）
	if dotEnvVal == "" {
		candidates := []string{".env"}

		if wd, err := os.Getwd(); err == nil {
			candidates = append(candidates, filepath.Join(wd, ".env"))
		}
		if execPath, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(execPath), ".env"))
		}
		// 单镜像容器内的固定路径（在 Windows 宿主 + Linux 容器场景下更直观）
		candidates = append(candidates, "/app/.env")

		for _, p := range candidates {
			if v, ok := readCleanupFromDotEnv(p); ok {
				dotEnvVal = strings.TrimSpace(v)
				break
			}
		}
	}

	switch {
	// 同时存在且不一致：以环境变量为准，但标记冲突
	case envVal != "" && dotEnvVal != "" && envVal != dotEnvVal:
		return envVal, envVal == "1", "env+.env", true, dotEnvVal
	// 同时存在且一致：以环境变量为准，标记为联合作用
	case envVal != "" && dotEnvVal != "" && envVal == dotEnvVal:
		return envVal, envVal == "1", "env+.env", false, dotEnvVal
	// 仅环境变量
	case envVal != "":
		return envVal, envVal == "1", "env", false, dotEnvVal
	// 仅 .env
	case dotEnvVal != "":
		return dotEnvVal, dotEnvVal == "1", ".env", false, dotEnvVal
	// 都未配置：默认未配置（安全优先，关闭清理模式）
	default:
		return "", false, "none", false, dotEnvVal
	}
}

// readCleanupFromDotEnv 仅解析 .env 中的 CLEANUP_ON_SHUTDOWN 一项，避免引入额外依赖。
// 支持如下格式：
//
//	CLEANUP_ON_SHUTDOWN=1
//	CLEANUP_ON_SHUTDOWN = "1"
//	export CLEANUP_ON_SHUTDOWN=1
func readCleanupFromDotEnv(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])

			if strings.HasPrefix(strings.ToLower(key), "export ") {
				key = strings.TrimSpace(key[len("export "):])
			}

			if key != "CLEANUP_ON_SHUTDOWN" {
				continue
			}

			// 去掉可能存在的引号
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}

			return strings.TrimSpace(val), true
		}
	}

	return "", false
}
