# 全平台Git与GitHub工作流管理规范

||**作者**：CYP | **联系方式**：nasDSSCYP@outlook.com |
||---|---|
|| **版本**：v1.0.0 | **最后更新**：2026年2月24日 |

---

## 文档修订记录

|| 版本 | 日期 | 修订内容 | 修订人 |
||------|------|----------|--------|
|| v1.0.0 | 2026年2月24日 | 初始版本：统一全平台 Git 与 GitHub 工作流规范 | CYP |

---

提示：本规范适用于个人在 Windows / macOS / Linux 等多平台使用 Git 与 GitHub 进行代码托管和协作的场景，侧重约束分支策略、提交信息、Tag 与发布流程等关键环节，避免因随意操作导致历史不可追溯、分支混乱或线上事故。

## 目录

- [1. 规范目的与适用范围](#1-规范目的与适用范围)
- [2. 前置配置（全平台通用）](#2-前置配置全平台通用)
- [3. 分支管理规范](#3-分支管理规范)
- [4. 提交规范（Commit）](#4-提交规范commit)
- [5. Tag管理规范（版本标记）](#5-tag管理规范版本标记)
- [6. 远程操作规范（GitHub同步）](#6-远程操作规范github同步)
- [7. 全平台适配补充说明](#7-全平台适配补充说明)
- [9. GitHub Actions配置与优化规范](#9-github-actions配置与优化规范)
- [10. GitHub分支保护与安全规则](#10-github分支保护与安全规则)
- [11. 代码评审与Pull Request规范](#11-代码评审与pull-request规范)
- [12. Git LFS大文件管理规范](#12-git-lfs大文件管理规范)
- [13. Git Hooks最佳实践](#13-git-hooks最佳实践)
- [14. 常见问题与解决方案](#14-常见问题与解决方案)
- [15. 附则](#15-附则)

## 使用说明

### 一、适配核心原则

1. 场景适配：在启用本规范前，应明确项目类型（单人项目 / 多人协作 / 多服务仓库等）以及使用方式（单仓库、多仓库、Monorepo），仅启用与当前项目匹配的分支模型与保护规则，避免过度约束影响开发效率。
2. 一致性优先：同一项目组内所有仓库的分支命名、提交信息格式、Tag 规则必须保持一致，禁止出现不同仓库采用不同前缀或版本号体系的情况。
3. 最小必要：在保障可追溯与安全的前提下，优先选择最小可行的一套流程（例如仅强制 main/dev 保护与 PR 审核），避免引入过多强制检查导致开发阻塞。

### 二、落地与维护方式

1. 首次落地：新建或接手项目时，优先完成 Git 全局配置（用户名、邮箱、换行符策略）、远程仓库关联以及基础分支结构（main/dev/feature/* 等）的创建。
2. 规则固化：将本规范中的关键规则配置到仓库级别（如 GitHub Branch Protection、必需的 Status Checks、PR 模板、Issue 模板），避免仅依赖“口头约定”。
3. 文档同步：当分支策略、PR 审核流程或 CI 检查项发生重要变更时，需同步更新本规范版本号与修订记录，并在仓库 README 或 CONTRIBUTING 文档中给出简要说明。

### 三、自查与培训

1. 自查频率：每个迭代周期结束后，对照自查清单检查是否存在直接在 main 分支提交、未按规范命名分支 / Tag、跳过代码评审等行为，必要时回溯修正。
2. 新人培训：新加入的协作者在获得仓库写入权限前，必须阅读本规范并完成一次基于测试仓库的演练（如创建分支、提交、发起 PR、合并、打 Tag）。
3. 工具辅助：推荐结合 Git Hooks、lint-staged、Commitlint 等工具，将提交信息格式、代码风格检查等自动化，减少人工审查成本。

#### 规范自查清单（Git 与 GitHub 工作流）

- [ ] 已按本规范完成 Git 全局配置（用户名、邮箱、换行符策略、默认编辑器等）
- [ ] 仓库分支结构仅包含 main/dev 及规范命名的 feature/fix/hotfix 分支，无长期未清理的临时分支
- [ ] 所有提交信息均符合约定格式（如 Conventional Commits），且描述清晰可追溯
- [ ] 所有合入 main/dev 的变更均通过 Pull Request 且至少完成一次代码评审
- [ ] 版本发布均通过 Tag 完成，并有明确的版本号与变更说明
- [ ] GitHub 仓库已配置基础分支保护规则（禁止强制推送、禁止直接推送到 main 等）
- [ ] CI（如 GitHub Actions）已接入基础检查（构建、测试、Lint），且 PR 合并前必须通过
- [ ] 已为大文件或二进制资源统一采用 Git LFS 管理（如有需要）
- [ ] 常见故障（如误删分支、错误 Rebase、强制推送等）均在团队内进行过演练及应急预案说明

## 1. 规范目的与适用范围

### 1.1 目的

统一全平台（Windows/macOS/Linux）Git使用规范与GitHub协作流程，保障代码版本可追溯、协作高效有序，减少冲突与操作风险，适配多端应用（网页端、管理端等）开发场景。

### 1.2 适用范围

本规范适用于所有基于Git版本控制、GitHub托管代码的项目，涵盖单人开发、多人协作、多环境部署等场景，适配全栈开发中的前端、后端、运维脚本等各类代码管理。

## 2. 前置配置（全平台通用）

### 2.1 Git安装与全局配置

2.1.1 安装要求：确保全平台安装Git 2.30+版本，安装包从[Git官方网站](https://git-scm.com/)获取，验证命令：`git --version`。

2.1.2 全局配置（仅首次执行）：

```bash

# 配置用户名（与GitHub账号一致）
git config --global user.name "你的GitHub用户名"
# 配置邮箱（与GitHub账号绑定邮箱一致）
git config --global user.email "你的GitHub绑定邮箱"
# 配置默认编辑器（可选，推荐VS Code）
git config --global core.editor code
# 跨平台换行符处理（关键配置，避免格式冲突）
git config --global core.autocrlf true  # Windows系统
git config --global core.autocrlf input  # macOS/Linux系统
# 配置提交时忽略文件权限变更
git config --global core.fileMode false
```

### 2.2 GitHub认证配置

优先使用SSH认证（免密码推送，全平台通用），步骤如下：

1. 生成SSH密钥：`ssh-keygen -t ed25519 -C "你的GitHub绑定邮箱"`，全程回车默认配置即可。

2. 获取公钥内容：
        

    - Windows：`cat ~/.ssh/id_ed25519.pub`（Git Bash中执行）

    - macOS/Linux：`pbcopy < ~/.ssh/id_ed25519.pub`（直接复制到剪贴板）

3. 在GitHub网页端「Settings → SSH and GPG keys」中添加公钥，标题标注设备信息（如「MacBook-Pro」）。

4. 验证连接：`ssh -T git@github.com`，出现「Hi 用户名! You've successfully authenticated」即为成功。

备用方案（HTTPS认证）：每次推送需输入GitHub账号密码（或Personal Access Token，推荐有效期90天以上）。

## 3. 分支管理规范

### 3.1 分支命名规则

采用「前缀-描述-编号」格式，前缀区分分支类型，适配多端开发模块划分：

|分支类型|前缀|命名示例|适用场景|
|---|---|---|---|
|主分支|main/master|main|存放正式发布版本，禁止直接提交，仅通过合并更新|
|开发分支|dev|dev|团队协作开发主分支，整合功能分支，定期同步main|
|功能分支|feature|feature/web-login、feature/admin-dashboard|新增功能/优化，按「端-功能」命名，从dev分支创建|
|修复分支|fix/hotfix|fix/web-form-validation、hotfix/api-crash|fix：开发中Bug修复；hotfix：线上紧急Bug修复|
|测试分支|test|test/v1.2.0|版本测试专用，从dev分支创建，测试通过后合并回dev|
### 3.2 分支操作流程

1. 分支创建：
        `# 从dev创建功能分支
git checkout dev
git pull origin dev
git checkout -b feature/web-login`

2. 分支合并：功能开发完成后，通过GitHub Pull Request（PR）合并到dev分支，需至少1人代码评审通过。

3. 分支清理：合并完成后，删除本地及远程功能分支，避免冗余。

禁止在main/dev分支直接开发，所有修改必须通过子分支提交；多人协作时，每日开发前需拉取对应分支最新代码。

## 4. 提交规范（Commit）

### 4.1 提交信息格式

遵循Conventional Commits规范，格式：`类型(作用域)：描述信息 [可选正文] [可选脚注]`，全平台统一格式，便于版本追溯。

### 4.2 类型与作用域说明

|类型|说明|作用域示例（适配多端）|
|---|---|---|
|feat|新增功能|web、admin、api、db|
|fix|修复Bug|web、admin、api、db|
|docs|文档更新（README、注释等）|docs、readme|
|style|代码格式调整（不影响逻辑）|web、admin|
|refactor|代码重构（既无新增功能也无修复Bug）|api、db|
|perf|性能优化|api、web|
|test|添加/修改测试用例|unit、e2e|
|chore|构建/依赖/工具配置变更|build、deps|
### 4.3 提交操作示例

```bash

# 查看变更
git status
git diff

# 添加变更（精准添加，避免误加）
git add src/pages/login.vue  # 单个文件
git add src/components/  # 整个目录

# 提交（描述简洁，不超过50字，中文优先）
git commit -m "feat(web)：实现登录页面表单验证功能"
git commit -m "fix(api)：修复用户列表接口分页异常问题 Closes #123"  # 关联GitHub Issue

# 修正上次提交（未推送远程时）
git commit --amend -m "feat(web)：实现登录页面表单验证与记住密码功能"
```

推荐使用commitlint工具（免费开源）校验提交信息格式，全平台可通过npm安装配置，集成到开发工具中。

### 4.4 提交类型与作用域补充说明

| 组合场景 | 正确示例 | 说明 |
|----------|----------|------|
| 多模块变更 | `feat(web,api)：统一用户认证模块` | 多个作用域用逗号分隔 |
| 重大变更 | `feat!: 重构登录引擎` | 使用`!`标记BREAKING CHANGE |
| 脚注关联 | `fix(api)：修复数据库连接池泄露 Closes #456 Fixes #789` | 可关联多个Issue |
| 版本升降级 | `chore(deps)：升级Node.js版本至20.x` | 依赖变更明确标注版本 |

## 5. Tag管理规范（版本标记）

### 5.1 Tag命名规则

遵循语义化版本（SemVer），格式：`v主版本.次版本.修订版本`，适配正式发布与预发布场景：

- 正式版本：v1.0.0、v2.3.1（主版本：重大变更；次版本：新增功能；修订版本：Bug修复）

- 预发布版本：v1.0.0-alpha.1、v2.0.0-beta.3（alpha：内部测试；beta：公开测试；rc：候选版本）

### 5.2 Tag操作流程

1. 创建Tag（推荐注解Tag，包含版本信息）：
        `# 创建注解Tag并添加描述
git tag -a v1.0.0 -m "v1.0.0 正式发布：实现核心功能（登录、权限管理、数据展示）"
# 为指定提交创建Tag（回溯版本时使用）
git tag -a v0.9.0 6a8b2c4 -m "v0.9.0 测试版本"`

2. 查看Tag：
        `git tag  # 列出所有Tag
git show v1.0.0  # 查看指定Tag详情`

3. 推送Tag到GitHub：
        `git push origin v1.0.0  # 推送单个Tag
git push origin --tags  # 推送所有未推送的Tag`

4. 删除Tag：
        `# 删除本地Tag
git tag -d v1.0.0
# 删除远程Tag
git push origin --delete v1.0.0`

Tag创建后不可随意删除/修改，对应正式发布版本的Tag需与GitHub Release关联，标注版本更新日志。

## 6. 远程操作规范（GitHub同步）

### 6.1 远程仓库关联与同步

```bash

# 关联远程仓库（首次克隆后无需操作）
git remote add origin git@github.com:用户名/仓库名.git
# 查看远程关联
git remote -v

# 拉取远程分支最新代码（优先使用fetch+merge，避免自动合并冲突）
git fetch origin dev  # 拉取dev分支更新，不合并
git merge origin/dev  # 手动合并到本地dev分支
# 简化拉取（自动合并，冲突需手动解决）
git pull origin dev

# 推送本地分支到远程
git push origin feature/web-login  # 首次推送
git push  # 后续推送（已关联追踪分支）
```

### 6.2 冲突处理规则

1. 拉取代码时若出现冲突，Git会标记冲突文件（含<<<<<<<、=======、>>>>>>>标记）。

2. 冲突解决步骤：
        

    - 打开冲突文件，根据业务逻辑保留正确代码，删除冲突标记。

    - 解决完成后，执行`git add 冲突文件路径`标记为已解决。

    - 提交解决结果：`git commit -m "chore：解决dev分支合并冲突"`。

3. 避免冲突策略：多人协作时，细分开发模块，每日至少拉取1次对应分支代码；小步提交，减少单次变更范围。

### 6.3  Fork与PR协作（开源/跨团队场景）

1. Fork目标仓库到个人GitHub账号，克隆到本地：`git clone git@github.com:个人用户名/仓库名.git`。

2. 添加上游仓库（同步原仓库更新）：`git remote add upstream git@github.com:原仓库用户名/仓库名.git`。

3. 创建分支开发完成后，推送至个人仓库，在GitHub网页端发起PR，目标分支选择原仓库dev分支。

4. PR通过后，同步上游仓库更新到本地：`git pull upstream dev`。

### 6.4 GitHub分支保护配置（安全规范）

在仓库「Settings → Branches」中配置分支保护规则，防止重要分支被误操作：

| 保护规则 | 配置说明 | 适用分支 |
|----------|----------|----------|
| Require pull request | 合并前必须创建PR，禁止直接推送 | main、dev |
| Require approvals | 至少1-2人评审通过后才能合并 | main（建议2人）、dev（建议1人）|
| Require status checks | 必须通过CI测试后才能合并 | main、dev、test |
| Require signed commits | 必须使用GPG签名提交 | main |
| Allow force push | 允许强制推送（仅限管理员） | 仅开发分支 |

### 6.5 提交前检查清单（推荐流程）

```bash

# 提交前执行以下检查
# 1. 查看变更内容
git diff --stat

# 2. 检查是否有敏感信息泄露
git diff --name-only | xargs grep -l "password\|api_key\|secret" 2>/dev/null

# 3. 运行本地测试（如果有）
npm run test:unit  # 或 mvn test

# 4. 确保代码格式化正确
npm run lint  # 或 mvn spotless:check

# 5. 验证提交信息格式（若配置了commitlint）
npx commitlint --edit $COMMIT_MSG
```

### 6.6 撤销操作规范

| 场景 | 命令 | 说明 |
|------|------|------|
| 撤销暂存 | `git reset HEAD <文件>` | 将文件从暂存区移出 |
| 撤销工作区修改 | `git checkout -- <文件>` | 丢弃工作区未暂存的修改 |
| 撤销最近提交（未推送） | `git reset --soft HEAD~1` | 保留修改在暂存区 |
| 撤销最近提交（已推送） | `git revert <提交ID>` | 创建新提交抵消修改 |
| 回退文件到历史版本 | `git checkout <提交ID> -- <文件>` | 恢复单个文件 |

## 7. 全平台适配补充说明

### 7.1 工具适配

推荐使用跨平台Git客户端（免费）：Git Bash（Windows）、Terminal（macOS/Linux）、SourceTree、GitKraken（基础版免费），确保命令与操作一致性。

### 7.2 忽略文件（.gitignore）

全平台统一配置.gitignore文件，涵盖IDE配置、依赖目录、日志文件等，示例如下（适配多端开发）：

```plain text

# IDE配置
.idea/
.vscode/
*.swp
*.swo

# 依赖与构建目录
node_modules/
dist/
build/
vendor/

# 日志与缓存
logs/
*.log
.cache/

# 环境变量文件
.env
.env.local
.env.production

# 系统文件
.DS_Store  # macOS
Thumbs.db  # Windows
```

### 7.3 版本回溯操作

```bash

# 查看提交历史（全平台一致）
git log --oneline --graph --decorate  # 简洁图形化展示
git log -p  # 查看详细变更

# 回溯到指定版本（保留变更，可重新提交）
git reset --soft 提交ID
# 强制回溯（丢弃后续所有变更，谨慎使用，未推送远程时）
git reset --hard 提交ID
# 恢复已删除的文件（从指定版本）
git checkout 提交ID -- 文件名路径
```

## 9. GitHub Actions配置与优化规范

### 9.1 基础配置原则

9.1.1 工作流文件组织：所有GitHub Actions配置文件统一放置于项目根目录`.github/workflows/`下，命名格式为「功能-场景.yml」，示例：`test-build.yml`、`deploy-auto.yml`，便于快速定位功能。

9.1.2 权限控制：明确工作流所需最小权限，在配置文件中通过`permissions`字段限制权限范围（如仅允许读取代码、写入PR状态），避免过度授权导致安全风险。

9.1.3 兼容性适配：工作流配置需兼容全栈项目技术栈（前端、后端、脚本），选用官方或广泛验证的Action组件，优先选择支持多平台运行的版本，避免平台特异性问题。

9.1.4 免费额度管控：基于GitHub Actions免费额度（公共仓库无限制，私有仓库每月有额度上限）规划任务，避免冗余流程消耗额度，优先使用轻量Runner执行任务。

### 9.2 自动化测试流程集成

9.2.1 测试触发时机：配置多场景自动触发规则，确保代码质量在提交、合并全流程受控：

1. 提交触发：针对`dev`、`feature/*`、`fix/*`分支，推送代码时自动执行测试流程，拦截不合格代码。

2. PR触发：创建或更新PR（目标分支为`main`、`dev`）时触发测试，测试通过后方可合并，需在PR页面显示测试结果状态。

3. 定时触发：每日凌晨执行全量测试（适配多端集成测试），提前发现潜在兼容性问题，配置示例：`schedule: - cron: '0 0 * * *'`。

9.2.2 测试类型覆盖：根据全栈项目特点，至少集成以下测试类型，确保多维度验证代码质量：

- 单元测试：覆盖前端组件、后端接口逻辑，使用对应技术栈工具（如Jest、JUnit），要求核心代码测试覆盖率不低于70%。

- 集成测试：验证前后端接口联调、数据库交互等场景，确保模块间协作正常。

- E2E测试：针对网页端、管理端关键业务流程（如登录、数据提交），使用Cypress等工具模拟真实用户操作。

9.2.3 测试结果处理：测试失败时，工作流需生成详细日志（含错误堆栈、测试报告），并通过GitHub通知功能同步给提交者；测试通过后，自动更新PR状态为「可合并」，避免人工干预延迟。

### 9.3 流程加速优化方案

9.3.1 依赖缓存优化：通过缓存依赖包减少重复下载耗时，适配不同技术栈的缓存策略：

```yaml

# 前端npm依赖缓存示例
- name: 缓存npm依赖
  uses: actions/cache@v3
  with:
    path: ~/.npm
    key: ${{ runner.os }}-npm-${{ hashFiles('**/package-lock.json') }}
    restore-keys: |
      ${{ runner.os }}-npm-

# 后端Maven依赖缓存示例
- name: 缓存Maven依赖
  uses: actions/cache@v3
  with:
    path: ~/.m2/repository
    key: ${{ runner.os }}-maven-${{ hashFiles('**/pom.xml') }}
    restore-keys: |
      ${{ runner.os }}-maven-
```

9.3.2 Runner选择优化：优先使用GitHub托管Runner（免费、跨平台），针对复杂任务（如大型后端构建）可选用性能更优的Runner（如Ubuntu-latest，启动速度快于Windows、macOS）；私有项目若有特殊需求，可搭建自建Runner（本地服务器部署，减少网络延迟），但需保障Runner稳定性与安全性。

9.3.3 任务并行执行：将独立任务拆分并行运行，缩短整体流程耗时，示例：前端构建与后端测试并行、多模块测试并行，通过`jobs`字段定义并行任务，使用`needs`字段控制依赖关系。

9.3.4 冗余步骤清理：移除工作流中不必要的步骤（如冗余日志打印、重复文件检查），简化构建流程；针对大型项目，可拆分工作流（如测试工作流与部署工作流分离），避免单一流程过于冗长。

9.3.5 网络加速补充：对于需访问国内资源的场景，可通过配置镜像源加速（如npm镜像、Maven镜像），在工作流中设置环境变量指定镜像地址，避免网络瓶颈。

### 9.4 常用场景配置示例

9.4.1 前端测试与构建加速工作流（`.github/workflows/frontend-test-build.yml`）：

```yaml

name: 前端测试与构建
on:
  push:
    branches: [ dev, 'feature/**' ]
  pull_request:
    branches: [ main, dev ]

jobs:
  test-build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - name: 拉取代码
        uses: actions/checkout@v4

      - name: 配置Node.js
        uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: 'npm' # 自动缓存npm依赖

      - name: 安装依赖（加速）
        run: npm ci --registry=https://registry.npmmirror.com # 使用国内镜像

      - name: 执行单元测试
        run: npm run test:unit
        continue-on-error: false # 测试失败终止流程

      - name: 构建项目
        run: npm run build

      - name: 上传构建产物
        uses: actions/upload-artifact@v4
        with:
          name: frontend-build
          path: dist/
```

9.4.2 后端测试与缓存优化工作流（适配Java项目）：

```yaml

name: 后端测试与缓存
on:
  push:
    branches: [ dev, 'fix/**' ]
  pull_request:
    branches: [ main, dev ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: 配置JDK
        uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'
          cache: maven # 缓存Maven依赖

      - name: 执行测试与构建
        run: mvn clean test package -Dmaven.repo.local=https://maven.aliyun.com/repository/public # 镜像加速

      - name: 上传测试报告
        uses: actions/upload-artifact@v4
        with:
          name: test-report
          path: target/surefire-reports/
```

### 9.5 注意事项

1. 工作流配置需定期维护，及时更新Action组件版本（避免使用过时版本导致兼容性问题），定期清理无效缓存策略。

2. 敏感信息（如密钥、镜像仓库账号）需通过GitHub仓库「Settings → Secrets and variables → Actions」配置，工作流中通过`${{ secrets.XXX }}`引用，禁止硬编码。

3. 自建Runner需定期更新系统与依赖，限制仅受信任仓库使用，避免恶意代码执行风险。

## 10. GitHub分支保护与安全规则

### 10.1 分支保护配置

在GitHub仓库「Settings → Branches → Branch protection rules」中配置：

| 配置项 | 推荐设置 | 作用 |
|--------|----------|------|
| Branch name pattern | `main`、`dev`、`release/*` | 匹配需要保护的分支 |
| Require pull request | ✅ 启用，至少1人审批 | 禁止直接推送代码 |
| Required approvals | `main: 2人`，`dev: 1人` | 强制代码评审 |
| Require status checks | ✅ 启用CI测试通过 | 防止不合格代码合并 |
| Require signed commits | ✅（仅main） | 确保提交来源可信 |
| Allow force push | ❌ 禁用 | 防止历史记录被篡改 |
| Restrict who can push | ✅ 仅限核心团队 | 限制直接操作权限 |

### 10.2 安全扫描集成

在GitHub Actions中集成安全扫描工具：

```yaml
# .github/workflows/security-scan.yml
name: 安全扫描
on:
  push:
    branches: [ main, dev ]
  pull_request:
    branches: [ main, dev ]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      # 敏感信息检测
      - name: TruffleHog扫描
        uses: trufflesecurity/trufflehog-action@main
        with:
          extra_args: --regex --entropy=false
          
      # 依赖漏洞扫描
      - name: OWASP依赖检查
        uses: dependency-check/Dependency-Check_Action@main
        with:
          project: 'my-project'
          format: 'HTML'
```

### 10.3 仓库安全设置

| 安全项 | 推荐配置 | 说明 |
|--------|----------|------|
| 可见性 | 私有（私有项目）或公开（开源项目） | 根据项目性质设置 |
| 协作者权限 | 按角色分配（Admin/Write/Read） | 最小权限原则 |
| Two-Factor Authentication | 强制所有协作者启用 | 账户安全加固 |
| Automated security fixes | ✅ 启用 | 自动修复已知漏洞 |
| Dependabot alerts | ✅ 启用 | 依赖漏洞告警 |

## 11. 代码评审与Pull Request规范

### 11.1 PR模板配置

在仓库根目录创建`.github/PULL_REQUEST_TEMPLATE.md`：

```markdown
## PR描述

### 变更类型
- [ ] 新功能（feat）
- [ ] Bug修复（fix）
- [ ] 文档更新（docs）
- [ ] 代码重构（refactor）
- [ ] 其他（chore）

### 关联Issue
Closes #XXX / Fixes #XXX

### 变更说明
<!-- 描述本次变更的内容和原因 -->

### 测试情况
- [ ] 已添加单元测试
- [ ] 已进行手动测试
- [ ] 测试覆盖率变化

### 检查清单
- [ ] 代码符合项目规范
- [ ] 无敏感信息泄露
- [ ] 新增API有文档说明
- [ ] 变量命名清晰合理

### 截图/演示（如果适用）
<!-- 粘贴UI变更截图或录屏 -->
```

### 11.2 评审Checklist

**代码质量检查项：**

- [ ] 代码逻辑正确，无明显Bug
- [ ] 命名规范，变量/函数名清晰表达意图
- [ ] 注释必要，解释复杂逻辑
- [ ] 无硬编码配置，提取到配置文件
- [ ] 无重复代码，遵循DRY原则
- [ ] 边界情况和异常处理完整

**代码规范检查项：**

- [ ] 格式化符合项目规范（Prettier/ESLint/Prettier）
- [ ] 提交信息符合Conventional Commits
- [ ] 单元测试覆盖率达标
- [ ] 无console.log/debugger等调试代码
- [ ] 无敏感信息（密码、密钥、Token）

**协作规范检查项：**

- [ ] PR描述完整，变更目的清晰
- [ ] 相关文档已同步更新
- [ ] 已通知相关人员Review
- [ ] 关注变更对其他模块的影响

### 11.3 合并策略选择

| 合并方式 | 命令/操作 | 适用场景 | 优缺点 |
|----------|-----------|----------|--------|
| **Merge commit** | `git merge --no-ff` | 保留完整分支历史 | 历史清晰，但会产生合并提交 |
| **Squash merge** | GitHub按钮操作 | 多个小提交整合 | 历史简洁，但丢失分支细节 |
| **Rebase merge** | `git rebase` | 保持线性历史 | 历史整洁，但需解决冲突 |

**推荐策略：**
- `main`分支使用 **Squash merge**（保持发布版本简洁）
- `dev`分支使用 **Merge commit**（保留功能开发轨迹）
- 个人分支可使用 **Rebase**（保持与上游同步）

### 11.4 评审反馈规范

**评审者应：**
- 24小时内响应PR请求
- 提供建设性反馈，而非批评
- 区分「必须修改」和「建议修改」
- 标注重要问题为「Blocking」，次要为「Nitpick」

**提交者应：**
- 及时响应反馈意见
- 大型修改分多次提交Review
- 保持沟通，解释设计决策

## 12. Git LFS大文件管理规范

### 12.1 适用场景

Git LFS（Large File Storage）适用于以下文件类型：

| 文件类型 | 示例 | 建议 |
|----------|------|------|
| 设计资源 | `.psd`、`.sketch`、`.fig` | 设计稿源文件 |
| 媒体文件 | `.mp4`、`.mov`、`.wav` | 视频/音频素材 |
| 二进制包 | `.jar`、`.zip`、`.exe` | 构建产物 |
| 大型数据集 | `.csv`（>100MB） | 训练数据 |
| 字体文件 | `.ttf`、`.otf` | 定制字体 |

**不适用场景：** 纯文本代码、小型配置文件、图片素材（<5MB）

### 12.2 LFS配置与操作

```bash

# 1. 初始化Git LFS（仓库首次使用）
git lfs install

# 2. 跟踪大文件类型
git lfs track "*.psd"      # 跟踪PSD文件
git lfs track "*.mp4"      # 跟踪视频文件
git lfs track "*.jar"      # 跟踪JAR包
git lfs track "*.zip"

# 3. 查看当前跟踪配置
cat .gitattributes

# 4. 常规Git操作（透明使用LFS）
git add 设计稿/mockup.psd
git commit -m "feat(assets)：添加新版UI设计稿"
git push origin dev

# 5. 克隆包含LFS文件的仓库
git lfs install                    # 确保LFS已安装
git clone git@github.com:xxx/repo.git
git lfs pull                       # 拉取LFS文件

# 6. 查看LFS文件状态
git lfs ls-files
```

### 12.3 LFS最佳实践

```bash

# .gitattributes 示例配置（.git/LFS配置分离）
# 放在仓库根目录
*.psd filter=lfs diff=lfs merge=lfs -text
*.mp4 filter=lfs diff=lfs merge=lfs -text
*.zip filter=lfs diff=lfs merge=lfs -text
!*.zip.lfsverify  # 排除LFS验证文件

# LFS存储配额管理（GitHub免费1GB，超出付费）
# 查看使用量：GitHub仓库 → Settings → Billing
```

**注意事项：**
- LFS文件变更会生成新的指针文件，保留完整版本历史
- 不要将LFS用于频繁变更的文件（如日志、缓存）
- 定期清理不再使用的大文件：`git lfs prune`

## 13. Git Hooks最佳实践

### 13.1 客户端Hooks配置

Git Hooks在特定操作时自动执行脚本，提升代码质量：

```bash

# 初始化husky（推荐方式）
npm install -D husky
npx husky install
npx husky add .husky/pre-commit "npm run lint"
npx husky add .husky/commit-msg 'npx --no -- commitlint --edit "$1"'

# 预提交Hook示例：.husky/pre-commit
#!/bin/sh
echo "正在执行代码检查..."

# 运行ESLint
npm run lint
if [ $? -ne 0 ]; then
  echo "❌ ESLint检查未通过，请修复后再提交"
  exit 1
fi

# 运行单元测试
npm run test:unit -- --passWithNoTests
if [ $? -ne 0 ]; then
  echo "❌ 单元测试未通过，请检查后再提交"
  exit 1
fi

echo "✅ 检查通过，可以提交"
```

### 13.2 commit-msg规范检查

```bash

# 安装commitlint
npm install -D @commitlint/cli @commitlint/config-conventional

# 创建配置文件：commitlint.config.js
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [2, 'always', [
      'feat', 'fix', 'docs', 'style', 'refactor',
      'perf', 'test', 'chore', 'build', 'ci'
    ]],
    'subject-max-length': [2, 'always', 50],
    'body-max-line-length': [2, 'always', 72]
  }
}
```

### 13.3 pre-push安全检查

```bash

# pre-push Hook示例：防止推送敏感信息
#!/bin/bash

while read local_ref local_sha remote_ref remote_sha
do
  if [[ "$remote_ref" == "refs/heads/main" ]]; then
    # 检查是否包含敏感关键词
    if git diff --stat $local_sha | grep -E "(password|secret|api_key|token)" > /dev/null; then
      echo "❌ 检测到可能的敏感信息，请检查后再推送"
      exit 1
    fi
  fi
done
```

### 13.4 Hooks管理最佳实践

| Hook类型 | 执行时机 | 推荐用途 | 强制程度 |
|----------|----------|----------|----------|
| pre-commit | 提交前 | 代码检查、格式化 | 推荐 |
| commit-msg | 编写提交信息后 | 提交信息规范 | 推荐 |
| pre-push | 推送到远程前 | 敏感信息检查、安全扫描 | 可选 |
| post-checkout | 检出后 | 环境初始化、依赖安装 | 可选 |
| post-merge | 合并后 | 安装依赖、数据库迁移 | 可选 |

## 14. 常见问题与解决方案

### 14.1 基础操作问题

| 问题 | 解决方案 |
|------|----------|
| **合并冲突无法解决** | `git merge --abort` 取消合并，`git pull` 后手动处理冲突 |
| **提交后发现忘记文件** | `git commit --amend --no-edit` 追加文件，或 `git add` 后 `git commit --amend` |
| **推送被拒绝（远程更新）** | `git pull --rebase` 拉取并变基，或 `git pull` 合并后推送 |
| **想修改上条提交信息** | `git commit --amend -m "新的提交信息"`（未推送时） |
| **误删分支恢复** | `git reflog` 查看操作历史，`git checkout -b <分支名> <commit-id>` 恢复 |

### 14.2 远程同步问题

| 问题 | 解决方案 |
|------|----------|
| **远程分支不存在** | `git fetch origin` 刷新远程分支列表 |
| **远程仓库地址变更** | `git remote set-url origin <新地址>` |
| **Fork仓库同步上游** | 参见「6.3 Fork与PR协作」章节 |
| **想查看某次远程提交** | `git log origin/main --oneline` 或 `git log origin/dev` |

### 14.3 特殊场景处理

| 场景 | 操作命令 |
|------|----------|
| **暂时保存当前工作** | `git stash` / `git stash push -m "描述"` |
| **恢复保存的工作** | `git stash pop` |
| **查看已stash列表** | `git stash list` |
| **创建空白提交（打标签）** | `git commit --allow-empty -m "chore：版本里程碑"` |
| **暂存部分文件** | `git add -p` 交互式添加 |
| **查看两个版本的差异** | `git diff main...dev` 查看分支间差异 |
| **找出引入Bug的提交** | `git bisect start` → `git bisect bad` → `git bisect good <提交ID>` |

### 14.4 Windows特有注意事项

| 问题 | 解决方案 |
|------|----------|
| **换行符导致文件修改** | `git config --global core.autocrlf true` |
| **路径分隔符问题** | 使用正斜杠 `/` 或引号包裹路径 |
| **权限变更误提交** | `git config --global core.fileMode false` |
| **Git Bash中文显示乱码** | `git config --global core.quotepath false` |

### 14.5 GitHub操作问题

| 问题 | 解决方案 |
|------|----------|
| **PR无法自动合并** | 本地解决冲突后推送，或联系管理员 |
| **想撤销已合并的PR** | 在GitHub页面创建Revert PR |
| **Actions执行失败** | 查看日志排查错误，检查权限配置 |
| **想回退线上版本** | `git revert <合并提交>` 或 `git checkout v1.x.x` |

### 14.6 性能优化技巧

| 场景 | 优化命令 |
|------|----------|
| **Git操作慢** | `git config --global core.preloadindex true` |
| **克隆速度慢** | 使用镜像：`git clone https://github.com.cnpmjs.org/用户名/仓库.git` |
| **大仓库加速** | 使用浅克隆：`git clone --depth 1` |
| **历史记录清理** | `git gc --aggressive --prune=now` |

## 15. 附则

15.1 本规范根据项目迭代可适时调整，调整需通知所有协作成员并更新文档。

15.2 若违反规范导致代码混乱、版本冲突等问题，由相关责任人协调解决。

15.3 本规范自发布之日起执行，解释权归项目团队所有。

> （注：文档部分内容可能由 AI 生成）