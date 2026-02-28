# 版本记录和存储机制设计

## 1. 版本记录文件格式

### version.json (当前版本文件)
```json
{
  "version": "1.0.0",
  "releaseDate": "2025-12-13",
  "commit": "abc123def456",
  "tag": "v1.0.0"
}
```

### changelog.json (变更历史记录)
```json
{
  "versions": [
    {
      "version": "1.0.0",
      "date": "2025-12-13",
      "changes": [
        {
          "type": "feat",
          "description": "初始版本发布"
        }
      ],
      "author": "system"
    },
    {
      "version": "0.1.0",
      "date": "2025-12-10",
      "changes": [
        {
          "type": "feat",
          "description": "基础框架搭建"
        }
      ],
      "author": "developer"
    }
  ]
}
```

## 2. 存储机制

### 文件结构
```
project/
├── .version/
│   ├── version.json     # 当前版本信息
│   ├── changelog.json   # 版本变更历史
│   └── versions/        # 版本快照目录
│       ├── v1.0.0.json
│       └── v0.1.0.json
├── version-check.js     # 版本检查脚本
└── CHANGELOG.md         # Markdown格式的变更日志(可选)
```

## 3. 版本记录内容

每条版本记录应包含：
- 版本号 (version)
- 发布日期 (date)
- 变更列表 (changes)
  - 变更类型 (type): feat(新功能), fix(修复), chore(日常维护), docs(文档), refactor(重构), perf(性能优化), test(测试)
  - 描述 (description)
- 作者 (author)
- 提交哈希 (commit, 可选)

## 4. 存储策略

1. **本地存储**：默认存储在项目根目录下的 `.version` 目录中
2. **远程存储**：可扩展支持Git标签、数据库等方式
3. **备份机制**：每次版本更新时自动备份历史版本信息