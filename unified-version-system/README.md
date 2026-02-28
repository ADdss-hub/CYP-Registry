# 通用版本管理系统使用指南

**作者**: CYP  
**联系方式**: nasDSSCYP@outlook.com  
**系统版本**: v2.1.0  
**规范标准**: [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html)

## 概述

本系统是一个完整的语义化版本管理解决方案，严格遵循 [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html) 规范，提供版本检查、更新、记录和自动管理功能。

## 项目结构

```
CYP-SemVer/
├── core/                           # 核心版本管理模块
│   ├── version-manager.js         # 统一版本管理器入口
│   ├── version-checker.js         # 版本检查和验证
│   ├── version-updater.js         # 版本更新和历史管理
│   ├── version-api.js             # 版本信息API服务
│   └── version-config.js          # 配置管理和项目适配
├── docs/                          # 项目文档
│   ├── README.md                  # 主要使用指南
│   ├── version-spec.md            # 版本规范说明
│   ├── extended-version-spec.md   # 扩展版本规范
│   ├── version-storage-design.md  # 存储设计文档
│   └── example-project.md         # 示例项目说明
├── .version/                      # 版本数据存储
│   ├── version.json               # 当前版本信息
│   └── changelog.json             # 版本变更历史
├── .version-config.json           # 项目配置文件
├── .version-record.json           # 版本记录数据
├── version-record-simple.js       # 版本记录管理工具
└── package.json                   # 项目配置
```

## 安装和初始化

1. 确保 Node.js 环境已安装
2. 在项目根目录下初始化版本管理系统：

```bash
# 初始化版本系统，默认版本为 v0.1.0
node core/version-manager.js init

# 或者指定初始版本号
node core/version-manager.js init v1.0.0
```

## 核心特性

### 严格的 SemVer 2.0.0 合规性

本系统完全遵循 [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html) 规范，实现了以下检查机制：

1. **版本格式验证** - 确保版本号符合 `vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]` 格式
2. **版本递增规则检查** - 验证版本号递增是否符合 SemVer 规范
3. **重复版本检测** - 防止发布已存在的版本号
4. **版本优先级验证** - 按照 SemVer 规范计算版本优先级
5. **变更类型规范检查** - 基于约定式提交（Conventional Commits）规范
6. **数字限制检查** - 验证版本号各部分是否为非负整数且无前导零

### 关键规则（基于 SemVer 2.0.0）

- **MUST（必须）**: 版本号各部分必须是非负整数，禁止前导零
- **MUST（必须）**: 主版本号递增时，次版本号和修订号必须重置为 0
- **MUST（必须）**: 次版本号递增时，修订号必须重置为 0
- **MUST（必须）**: 不兼容的 API 修改必须递增主版本号
- **MUST（必须）**: 向下兼容的功能性新增必须递增次版本号
- **MUST（必须）**: 向下兼容的问题修正必须递增修订号

## 基本命令

### 检查版本有效性

```bash
# 检查版本格式是否符合 SemVer 规范
node core/version-manager.js check v1.2.3

# 检查带有预发布标签的版本
node core/version-manager.js check v2.1.0-beta.1

# 检查版本并验证变更类型
node core/version-manager.js check v1.2.3 --changes '[{"type":"feat","description":"添加新功能"}]'

# 检查包含破坏性变更的版本
node core/version-manager.js check v2.0.0 --changes '[{"type":"feat","description":"添加新功能 BREAKING CHANGE: 不兼容的API修改"}]'
```

### 变更类型规范检查

系统支持验证变更类型是否符合约定式提交（Conventional Commits）规范：

```bash
# 检查无效的变更类型（会报错）
node core/version-manager.js check v1.2.3 --changes '[{"type":"invalid","description":"无效变更类型"}]'
```

### 版本递增检查

系统会自动检查版本递增是否符合 SemVer 规范：

```bash
# 检查有效的版本递增
node core/version-manager.js check v1.2.3  # 假设当前版本为 v1.2.2

# 检查无效的版本递增（会报错）
node core/version-manager.js check v2.1.0  # 假设当前版本为 v1.2.3，主版本号递增但次版本号未重置为0
```

### 重复版本检查

系统会检查要更新的版本是否已存在于历史记录中：

```bash
# 检查重复版本（会报错）
node core/version-manager.js check v1.2.3  # 假设版本v1.2.3已存在于历史记录中
```

### 更新版本

```bash
# 更新到指定版本
node core/version-manager.js update v1.2.3

# 更新版本并添加作者和变更描述
node core/version-manager.js update v1.2.4 --author "张三" --message "修复登录bug"

# 更新版本并记录 Git 提交哈希
node core/version-manager.js update v1.2.5 --commit "a1b2c3d"
```

### 自动递增版本

```bash
# 自动递增补丁版本 (v1.2.3 -> v1.2.4)
node core/version-manager.js increment patch

# 自动递增次版本 (v1.2.3 -> v1.3.0)
node core/version-manager.js increment minor

# 自动递增主版本 (v1.2.3 -> v2.0.0)
node core/version-manager.js increment major
```

### 查看版本历史

```bash
# 显示所有版本历史记录
node core/version-manager.js history
```

### 版本记录管理

```bash
# 扫描项目中所有文件的版本信息
node version-record-simple.js scan

# 自动同步和更新版本记录
node version-record-simple.js auto

# 导出版本记录信息
node version-record-simple.js export

# 查看版本记录详情
node version-record-simple.js info
```

## 文件结构

系统会在项目根目录下创建 `.version` 目录，包含以下文件：

```
.version/
├── version.json     # 当前版本信息
├── changelog.json   # 版本变更历史
└── versions/        # 版本快照目录（未来扩展）
```

### version.json 格式

```json
{
  "version": "v1.2.3",
  "releaseDate": "2025-12-13",
  "commit": "a1b2c3d4e5f",
  "tag": "v1.2.3"
}
```

### changelog.json 格式

```json
{
  "versions": [
    {
      "version": "v1.2.3",
      "date": "2025-12-13",
      "changes": [
        {
          "type": "fix",
          "description": "修复登录bug"
        }
      ],
      "author": "张三"
    }
  ]
}
```

## 集成到构建流程

可以在项目的 package.json 中添加脚本：

```json
{
  "scripts": {
    "version-check": "node core/version-manager.js check",
    "version-update": "node core/version-manager.js update",
    "pre-release": "node core/version-manager.js increment patch",
    "release-major": "node core/version-manager.js increment major",
    "release-minor": "node core/version-manager.js increment minor",
    "version-history": "node core/version-manager.js history",
    "version-record": "node version-record-simple.js scan",
    "version-record-auto": "node version-record-simple.js auto"
  }
}
```

## 开发指南

### API 使用

可以直接在代码中使用版本管理类：

```javascript
const VersionManager = require('./core/version-manager.js');

const manager = new VersionManager();

// 检查版本
const report = manager.checker.checkVersion('v1.2.3');
console.log(report.valid); // true 或 false

// 更新版本
manager.validateAndUpdate('v1.2.4', {
  author: '开发者',
  changes: [{ type: 'feat', description: '添加新功能' }]
});

// 自动递增版本
const newVersion = manager.autoIncrementVersion('minor');
console.log(newVersion); // 例如: v1.3.0
```

### 版本检查API

VersionChecker 类提供了详细的版本检查方法：

```javascript
const VersionChecker = require('./core/version-checker.js');
const checker = new VersionChecker({
  projectName: 'my-project',
  config: {}
});

// 检查版本格式
const isValid = checker.checkVersionFormat('v1.2.3');

// 比较版本号
const comparison = checker.compareVersions('v1.2.3', 'v1.2.2'); // 返回 1 (第一个版本更大)

// 检查版本递增是否符合规范
const incrementCheck = checker.validateIncrement('v1.2.3', 'v2.0.0'); // 检查主版本号递增

// 检查版本是否已存在于历史记录中
const consistencyCheck = checker.checkDuplicateVersion('v1.2.3', history);

// 全面的版本检查
const fullReport = checker.checkVersion('v1.2.3', changes, currentVersion, history);
```

## 最佳实践（基于 SemVer 2.0.0）

### 1. 版本递增原则（规范第 6、7、8 条）

根据 SemVer 2.0.0 规范：

- **PATCH（修订号）**: 向下兼容的问题修正
  - 修复了向下兼容的问题
  - 更新了文档
  - 优化了性能
  - 重构了代码（不影响功能）

- **MINOR（次版本号）**: 向下兼容的功能性新增
  - 添加了向下兼容的新功能
  - 标记了某些功能为弃用
  - 引入了大量内部改进

- **MAJOR（主版本号）**: 不兼容的 API 修改
  - 做了不兼容的 API 修改
  - 删除了已弃用的功能
  - 重大架构调整

### 2. 版本发布流程

**开发阶段（0.y.z）**：
- 使用 0.y.z 版本号
- 频繁迭代，快速验证
- 可以在次版本号递增时引入不兼容变更

**测试阶段**：
- 发布 alpha 版本进行内部测试（`v1.0.0-alpha.1`）
- 发布 beta 版本进行公开测试（`v1.0.0-beta.1`）
- 发布 rc 版本进行最后验证（`v1.0.0-rc.1`）

**正式发布（1.0.0+）**：
- 发布 1.0.0 版本界定公共 API
- 严格遵循 SemVer 2.0.0 规范
- 不兼容修改必须递增主版本号

### 3. 变更日志管理

每次版本发布**应该**记录以下信息：

- **版本号** - 遵循 SemVer 2.0.0 格式
- **发布日期** - ISO 8601 格式（YYYY-MM-DD）
- **变更类型** - 使用约定式提交规范
- **变更描述** - 清晰描述变更内容
- **破坏性变更** - 明确标记并详细说明
- **作者信息** - 记录变更作者

### 4. 自动化集成

- 在 CI/CD 流程中集成版本检查
- 在发布流程中自动更新版本号
- 使用 Git 标签标记版本发布
- 自动生成变更日志

### 5. 版本号选择指南

**何时递增 PATCH 版本：**
- 修复了向下兼容的问题 ✅
- 更新了文档 ✅
- 优化了性能 ✅
- 重构了代码（不影响功能）✅
- **限制**：PATCH 不得超过 30，达到 30 时必须递增 MINOR

**何时递增 MINOR 版本：**
- 添加了向下兼容的新功能 ✅
- 标记了某些功能为弃用 ✅
- 引入了大量内部改进 ✅
- PATCH 达到 30 时必须递增 MINOR ⚠️
- PATCH 超过 20 时建议递增 MINOR 💡
- **限制**：MINOR 不得超过 50，达到 50 时必须递增 MAJOR

**何时递增 MAJOR 版本：**
- 做了不兼容的 API 修改 ✅
- 删除了已弃用的功能 ✅
- 重大架构调整 ✅
- MINOR 达到 50 时必须递增 MAJOR ⚠️
- MINOR 超过 40 时建议递增 MAJOR 💡
- **限制**：MAJOR 不得超过 99

### 6. 版本号数字限制

为了保持版本号的可读性和语义化，本系统**强制执行**以下限制：

**强制限制（MUST）**：
- **MAJOR**: 0-99（超过 99 必须重新评估版本策略）
- **MINOR**: 0-50（超过 50 必须递增 MAJOR）
- **PATCH**: 0-30（超过 30 必须递增 MINOR）

**警告阈值（SHOULD）**：
- **PATCH > 20**: 建议尽快递增 MINOR
- **MINOR > 40**: 建议尽快递增 MAJOR
- **MAJOR > 50**: 建议重新评估版本策略

## 版本检查规则（SemVer 2.0.0 合规）

本系统严格按照 [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html) 规范进行版本检查：

### 格式验证（规范第 2 条）

- 版本号**必须**符合 `vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]` 格式
- 版本号各部分**必须**是非负整数
- 版本号各部分**禁止**前导零（例如：`v01.1.0` 是无效的）
- 版本号**必须**以 `v` 前缀开头（项目约定）

### 递增验证（规范第 6、7、8 条）

- **主版本号（MAJOR）递增**：次版本号和修订号**必须**重置为 0
  - 示例：`v1.2.3` → `v2.0.0` ✅
  - 示例：`v1.2.3` → `v2.1.0` ❌（次版本号未重置）
  
- **次版本号（MINOR）递增**：修订号**必须**重置为 0
  - 示例：`v1.2.3` → `v1.3.0` ✅
  - 示例：`v1.2.3` → `v1.3.1` ❌（修订号未重置）
  
- **修订号（PATCH）递增**：主版本号和次版本号**必须**保持不变
  - 示例：`v1.2.3` → `v1.2.4` ✅
  - 示例：`v1.2.3` → `v1.3.4` ❌（次版本号改变）

### 版本优先级（规范第 11 条）

版本优先级按以下规则判断：

1. 按主版本号、次版本号、修订号的数值比较
2. 有预发布版本的版本号优先级**低于**正式版本
3. 预发布版本按标识符从左到右比较

**优先级示例：**
```
v1.0.0-alpha < v1.0.0-alpha.1 < v1.0.0-beta < v1.0.0-rc.1 < v1.0.0
```

### 变更类型规范（基于约定式提交）

根据 SemVer 2.0.0 规范和约定式提交规范：

| 类型       | 说明                         | 对版本号的影响        | SemVer 规则 |
|------------|------------------------------|----------------------|-------------|
| feat       | 新功能                       | MINOR 版本**必须**递增 | 向下兼容的功能性新增 |
| fix        | 修复问题                     | PATCH 版本**必须**递增 | 向下兼容的问题修正 |
| docs       | 文档更新                     | PATCH 版本**应该**递增 | 不影响代码功能 |
| style      | 代码风格更新                 | PATCH 版本**应该**递增 | 不影响代码含义 |
| refactor   | 代码重构                     | PATCH 版本**应该**递增 | 既不修复问题也不添加功能 |
| perf       | 性能优化                     | PATCH 版本**应该**递增 | 提高性能的代码更改 |
| test       | 测试相关                     | PATCH 版本**可以**递增 | 添加或修正测试 |
| build      | 构建系统或外部依赖变更       | PATCH 版本**可以**递增 | 影响构建系统或依赖 |
| ci         | CI 配置文件和脚本变更        | PATCH 版本**可以**递增 | 持续集成配置更改 |
| chore      | 日常维护工作                 | PATCH 版本**可以**递增 | 其他不修改源码的更改 |
| revert     | 回滚之前的提交               | 根据被回滚的提交决定  | 撤销之前的提交 |

### 破坏性变更（规范第 8 条）

- 当你做了不兼容的 API 修改时，**必须**递增主版本号（MAJOR）
- 破坏性变更**可以**在任何类型的提交中声明
- 破坏性变更**必须**在提交信息中包含 `BREAKING CHANGE:` 字段

**示例：**
```
feat: 允许配置对象扩展其他配置

BREAKING CHANGE: `extends` 键现在用于扩展其他配置文件
```

### 开发阶段版本（规范第 4 条）

- 主版本号为零（0.y.z）的软件处于开发初始阶段
- 一切都可能随时被改变
- 公共 API 不应该被视为稳定版
- 1.0.0 版本号用于界定公共 API 的形成

## 版本记录自动查找机制

本系统提供了强大的版本记录自动查找和管理功能，能够自动扫描项目中所有JavaScript/TypeScript文件，提取版本信息并统一管理。

### 核心功能

1. **自动文件扫描**：递归扫描项目目录，识别包含版本信息的JavaScript/TypeScript文件
2. **版本信息提取**：从文件内容中提取版本号、支持者信息、变更类型等
3. **版本记录管理**：统一存储和管理所有文件的版本信息
4. **自动同步**：保持版本记录与实际文件版本信息的一致性
5. **格式验证**：验证版本号格式和递增规则的正确性

### 版本记录API

```javascript
const VersionRecordManager = require('./version-record-simple.js');

// 扫描并分析项目中的版本信息
const records = manager.scanProject();
console.log(`发现 ${records.length} 个版本记录`);

// 自动同步版本记录
manager.autoSync();

// 导出版本记录信息
const exportedData = manager.exportRecords();
```

### 版本记录文件格式

版本记录数据存储在 `.version-record.json` 文件中，格式如下：

```json
{
  "projectName": "CYP-SemVer",
  "lastUpdate": "2025-12-13T19:50:00.000Z",
  "totalFiles": 15,
  "files": [
    {
      "filePath": "core/version-manager.js",
      "version": "v2.1.0",
      "type": "module",
      "detected": "2025-12-13T19:50:00.000Z",
      "content": {
        "version": "v2.1.0",
        "author": "Universal Version Manager",
        "lastModified": "2025-12-13"
      }
    }
  ]
}
```

## 技术特性

### 项目自适应配置
- 自动检测项目类型（Node.js、TypeScript、前端项目等）
- 智能适配不同的版本标记格式和约定
- 动态配置验证规则和检查策略

### 跨文件版本同步
- 自动识别package.json中的版本信息
- 同步更新所有相关文件的版本号
- 保持版本一致性，防止版本漂移

### 完整的历史记录
- 详细的版本变更历史记录
- 支持回滚和版本恢复
- 变更类型和作者信息跟踪

### 集成友好
- 命令行接口便于CI/CD集成
- 模块化API支持直接代码调用
- 支持自定义钩子和回调函数


## SemVer 2.0.0 合规性声明

本版本管理系统严格遵循 [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html) 规范，实现了规范中定义的所有 MUST、SHOULD 和 MAY 规则。

### 已实现的 SemVer 2.0.0 规范条款

✅ **第 1 条** - 使用语义化版本控制的软件必须定义公共 API  
✅ **第 2 条** - 版本号格式必须为 X.Y.Z，各部分必须是非负整数，禁止前导零  
✅ **第 3 条** - 版本发布后内容禁止修改，任何修改必须发布为新版本  
✅ **第 4 条** - 主版本号为零（0.y.z）的软件处于开发初始阶段  
✅ **第 5 条** - 1.0.0 版本号用于界定公共 API  
✅ **第 6 条** - 修订号必须在做了向下兼容的问题修正时递增  
✅ **第 7 条** - 次版本号必须在做了向下兼容的功能性新增时递增  
✅ **第 8 条** - 主版本号必须在做了不兼容的 API 修改时递增  
✅ **第 9 条** - 预发布版本可以通过在修订号后添加连字符和标识符来表示  
✅ **第 10 条** - 构建元数据可以通过在修订号或预发布版本后添加加号和标识符来表示  
✅ **第 11 条** - 版本优先级按主版本号、次版本号、修订号的数值比较来计算  

### 扩展规范

除了严格遵循 SemVer 2.0.0 规范外，本系统还提供了以下扩展功能：

- **约定式提交集成** - 基于 [Conventional Commits](https://www.conventionalcommits.org/) 规范的变更类型管理
- **版本记录自动查找** - 自动扫描和同步项目中所有文件的版本信息
- **项目自适应配置** - 智能适配不同类型的项目（Node.js、TypeScript、前端项目等）
- **完整的历史记录** - 详细的版本变更历史记录和回滚支持
- **数字限制建议** - 提供版本号数字的推荐限制和警告阈值

### 规范文档

本系统提供了完整的规范文档：

- **[version-spec.md](docs/version-spec.md)** - 核心版本规范（基于 SemVer 2.0.0）
- **[extended-version-spec.md](docs/extended-version-spec.md)** - 扩展版本规范（项目特定实践）
- **[semver-2.0.0-spec.md](docs/semver-2.0.0-spec.md)** - SemVer 2.0.0 官方规范中文翻译

## 常见问题（FAQ）

### Q1: 如何处理修订号递增过多的情况？

**A:** 根据 SemVer 2.0.0 规范和最佳实践，如果修订号递增超过 20 次，建议递增次版本号。这表明项目可能积累了足够多的小改进，应该作为一个新的次版本发布。

### Q2: 预发布版本如何升级到正式版本？

**A:** 预发布版本升级到正式版本时，只需移除预发布标识符即可。例如：`v1.0.0-rc.1` → `v1.0.0`

### Q3: 构建元数据何时使用？

**A:** 构建元数据用于标识特定的构建，但不影响版本优先级。常用于 CI/CD 流程中标识构建时间、提交哈希等信息。例如：`v1.0.0+20130313144700`

### Q4: 如何处理已发布版本的错误？

**A:** 根据 SemVer 2.0.0 规范第 3 条，一旦发布，版本内容**禁止**修改。如果发现错误，**必须**发布新版本进行修复。

### Q5: 主版本号为 0 时如何管理版本？

**A:** 主版本号为 0 时，软件处于开发初始阶段，可以在次版本号递增时引入不兼容变更。建议在 API 稳定后发布 1.0.0 版本。

### Q6: 如何判断是否应该递增主版本号？

**A:** 当你做了不兼容的 API 修改时，**必须**递增主版本号。判断标准：
- 删除了公共 API
- 修改了公共 API 的行为
- 重命名了公共 API
- 修改了公共 API 的参数

### Q7: 版本号可以跳过吗？

**A:** 可以。SemVer 2.0.0 规范只要求版本号单调递增，不要求连续。例如：`v1.0.0` → `v1.2.0` 是有效的。

### Q8: 如何处理多个并行开发分支的版本号？

**A:** 建议使用预发布版本标识符来区分不同的开发分支。例如：
- 主分支：`v1.0.0`
- 开发分支：`v1.1.0-dev.1`
- 功能分支：`v1.1.0-feature.auth.1`

## 参考资料

### 官方规范

- [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html) - SemVer 官方规范（英文）
- [语义化版本 2.0.0](https://semver.org/lang/zh-CN/) - SemVer 官方规范（中文）
- [Conventional Commits](https://www.conventionalcommits.org/) - 约定式提交规范
- [Keep a Changelog](https://keepachangelog.com/) - 变更日志规范

### 相关工具

- [semver](https://www.npmjs.com/package/semver) - Node.js 的 SemVer 解析和比较库
- [standard-version](https://www.npmjs.com/package/standard-version) - 自动化版本管理和变更日志生成
- [semantic-release](https://www.npmjs.com/package/semantic-release) - 完全自动化的版本管理和发布

### 最佳实践

- [npm 版本管理最佳实践](https://docs.npmjs.com/about-semantic-versioning)
- [GitHub 版本发布指南](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [语义化版本控制实践指南](https://www.jvandemo.com/a-simple-guide-to-semantic-versioning/)

## 贡献指南

欢迎贡献代码、报告问题或提出建议！

### 贡献流程

1. Fork 本项目
2. 创建特性分支（`git checkout -b feature/AmazingFeature`）
3. 提交更改（`git commit -m 'feat: 添加某个功能'`）
4. 推送到分支（`git push origin feature/AmazingFeature`）
5. 开启 Pull Request

### 提交规范

请遵循约定式提交（Conventional Commits）规范：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型（type）：**
- feat: 新功能
- fix: 修复问题
- docs: 文档更新
- style: 代码风格
- refactor: 代码重构
- perf: 性能优化
- test: 测试相关
- build: 构建系统
- ci: CI 配置
- chore: 日常维护

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](../LICENSE) 文件。

## 联系方式

- **作者**: CYP
- **邮箱**: nasDSSCYP@outlook.com
- **项目主页**: [GitHub](https://github.com/your-repo/unified-version-system)

## 更新日志

### v2.1.0 (2025-12-28)

**新增功能：**
- ✅ 完全遵循 SemVer 2.0.0 规范
- ✅ 添加 SemVer 2.0.0 官方规范中文翻译文档
- ✅ 更新所有文档以符合 SemVer 2.0.0 规范
- ✅ 增强版本检查机制，严格验证 SemVer 规则
- ✅ 添加版本号数字限制检查和警告

**改进：**
- 📝 重写核心版本规范文档
- 📝 更新扩展版本规范文档
- 📝 完善 README 文档
- 🔧 优化版本检查逻辑
- 🔧 改进错误提示信息

**文档：**
- 📚 添加 SemVer 2.0.0 合规性声明
- 📚 添加常见问题解答（FAQ）
- 📚 添加参考资料和最佳实践
- 📚 添加贡献指南

### v2.0.0 (2025-12-13)

**新增功能：**
- 添加变更类型规范和验证
- 实现版本检查机制
- 支持约定式提交规范

### v1.0.0 (2025-12-01)

**初始版本：**
- 基础版本管理功能
- 版本格式验证
- 版本历史记录
