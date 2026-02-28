#!/usr/bin/env node

/**
 * 简洁版版本记录管理系统
 * 专注于版本记录的自动查找和同步功能
 */

const fs = require('fs');
const path = require('path');
const VersionChecker = require('./core/version-checker.js');

class SimpleVersionRecordManager {
  constructor(projectRoot = process.cwd()) {
    this.projectRoot = path.resolve(projectRoot);
    this.recordFile = path.join(this.projectRoot, '.version-record.json');
    
    // 初始化版本检查器
    this.versionChecker = new VersionChecker({
      projectName: path.basename(this.projectRoot)
    });
    
    // 项目中的文件列表
    this.projectFiles = [];
    // 版本记录映射表
    this.fileVersionMap = new Map();
    
    // 版本锁定配置
    this.versionLocking = {
      enabled: false,
      lockedFiles: new Set(),
      lockedVersions: new Map(),
      lockFile: path.join(this.projectRoot, '.version-lock.json'),
      autoLock: false,
      lockPatterns: []
    };
    
    // 项目自适应配置
    this.projectConfig = {
      type: null,
      framework: null,
      buildSystem: null,
      packageManager: null,
      hasDocker: false,
      hasGit: false,
      structure: {},
      customPatterns: []
    };
    
    // 系统文件排除模式
    this.systemExclusionPatterns = [
      // 排除自身系统文件
      /unified-version-system/,
      /\.version-record\.json$/,
      /version-record-simple\.js$/,
      /version-manager\.js$/,
      /\.version-config\.json$/,
      /universal-version-manager/,
      
      // 排除node_modules中的版本信息
      /node_modules/,
      
      // 排除构建输出
      /dist\//,
      /build\//,
      /out\//,
      /target\//,
      
      // 排除缓存文件
      /\.cache\//,
      /\.tmp\//,
      /\.temp\//
    ];
    
    this.loadRecordFile();
    this.detectProjectType();
    this.loadVersionLocking();
  }

  /**
   * 检测项目类型和配置
   */
  detectProjectType() {
    console.log('开始检测项目类型...');
    
    // 检测Git仓库
    this.projectConfig.hasGit = fs.existsSync(path.join(this.projectRoot, '.git'));
    
    // 检测Docker
    this.projectConfig.hasDocker = fs.existsSync(path.join(this.projectRoot, 'Dockerfile')) ||
                                   fs.existsSync(path.join(this.projectRoot, 'docker-compose.yml'));
    
    // 检测包管理器
    this.detectPackageManager();
    
    // 检测框架
    this.detectFramework();
    
    // 检测构建系统
    this.detectBuildSystem();
    
    // 分析项目结构
    this.analyzeProjectStructure();
    
    console.log('项目检测完成:', this.projectConfig);
  }
  
  /**
   * 检测包管理器类型
   */
  detectPackageManager() {
    const packageFiles = {
      'npm': ['package.json', 'package-lock.json'],
      'yarn': ['yarn.lock'],
      'pnpm': ['pnpm-lock.yaml'],
      'maven': ['pom.xml'],
      'gradle': ['build.gradle', 'build.gradle.kts'],
      'pip': ['requirements.txt', 'setup.py', 'Pipfile'],
      'cargo': ['Cargo.toml'],
      'go': ['go.mod'],
      'composer': ['composer.json'],
      'nuget': ['*.csproj'],
      'Bundler': ['Gemfile']
    };
    
    for (const [manager, files] of Object.entries(packageFiles)) {
      for (const file of files) {
        if (file.includes('*')) {
          // 处理通配符
          const pattern = file.replace('*', '.*');
          const regex = new RegExp(pattern);
          const rootFiles = fs.readdirSync(this.projectRoot).filter(f => regex.test(f));
          if (rootFiles.length > 0) {
            this.projectConfig.packageManager = manager;
            return;
          }
        } else if (fs.existsSync(path.join(this.projectRoot, file))) {
          this.projectConfig.packageManager = manager;
          return;
        }
      }
    }
  }
  
  /**
   * 检测框架类型
   */
  detectFramework() {
    const frameworks = {
      'React': {
        files: ['package.json'],
        check: (content) => content.includes('"react"') || content.includes('"create-react-app"')
      },
      'Vue': {
        files: ['package.json', 'vue.config.js'],
        check: (content) => content.includes('"vue"') || content.includes('@vue/cli')
      },
      'Angular': {
        files: ['package.json', 'angular.json'],
        check: (content) => content.includes('"@angular/core"') || content.includes('@angular/cli')
      },
      'Next.js': {
        files: ['package.json', 'next.config.js'],
        check: (content) => content.includes('"next"')
      },
      'Nuxt.js': {
        files: ['package.json', 'nuxt.config.js'],
        check: (content) => content.includes('"nuxt"')
      },
      'Express': {
        files: ['package.json'],
        check: (content) => content.includes('"express"')
      },
      'Spring Boot': {
        files: ['pom.xml', 'build.gradle'],
        check: (content) => content.includes('spring-boot') || content.includes('springframework')
      },
      'Django': {
        files: ['requirements.txt', 'manage.py'],
        check: (content, files) => fs.existsSync(path.join(this.projectRoot, 'manage.py'))
      },
      'Flask': {
        files: ['app.py', 'requirements.txt'],
        check: (content) => content.includes('flask') || content.includes('from flask')
      },
      'FastAPI': {
        files: ['main.py', 'requirements.txt'],
        check: (content) => content.includes('fastapi')
      }
    };
    
    for (const [framework, config] of Object.entries(frameworks)) {
      let found = false;
      for (const file of config.files) {
        const filePath = path.join(this.projectRoot, file);
        if (fs.existsSync(filePath)) {
          try {
            const content = fs.readFileSync(filePath, 'utf8');
            if (config.check(content, config.files)) {
              this.projectConfig.framework = framework;
              found = true;
              break;
            }
          } catch (error) {
            // 忽略读取错误
          }
        }
      }
      if (found) break;
    }
  }
  
  /**
   * 检测构建系统
   */
  detectBuildSystem() {
    const buildSystems = {
      'Webpack': ['webpack.config.js', 'webpack.config.ts'],
      'Vite': ['vite.config.js', 'vite.config.ts'],
      'Rollup': ['rollup.config.js'],
      'Parcel': ['.parcelrc'],
      'Gulp': ['gulpfile.js', 'gulpfile.ts'],
      'Grunt': ['Gruntfile.js'],
      'Make': ['Makefile'],
      'CMake': ['CMakeLists.txt'],
      'Maven': ['pom.xml'],
      'Gradle': ['build.gradle', 'build.gradle.kts'],
      'MSBuild': ['*.csproj'],
      'Xcode': ['*.xcodeproj'],
      'Bazel': ['BUILD', 'BUILD.bazel']
    };
    
    for (const [buildSystem, files] of Object.entries(buildSystems)) {
      for (const file of files) {
        if (file.includes('*')) {
          // 处理通配符
          const pattern = file.replace('*', '.*');
          const regex = new RegExp(pattern);
          const rootFiles = fs.readdirSync(this.projectRoot).filter(f => regex.test(f));
          if (rootFiles.length > 0) {
            this.projectConfig.buildSystem = buildSystem;
            return;
          }
        } else if (fs.existsSync(path.join(this.projectRoot, file))) {
          this.projectConfig.buildSystem = buildSystem;
          return;
        }
      }
    }
  }
  
  /**
   * 分析项目结构
   */
  analyzeProjectStructure() {
    const structure = {
      hasSrc: fs.existsSync(path.join(this.projectRoot, 'src')),
      hasApp: fs.existsSync(path.join(this.projectRoot, 'app')),
      hasLib: fs.existsSync(path.join(this.projectRoot, 'lib')),
      hasTest: fs.existsSync(path.join(this.projectRoot, 'test')) || 
               fs.existsSync(path.join(this.projectRoot, 'tests')) ||
               fs.existsSync(path.join(this.projectRoot, '__tests__')),
      hasDocs: fs.existsSync(path.join(this.projectRoot, 'docs')) ||
               fs.existsSync(path.join(this.projectRoot, 'doc')),
      hasConfig: fs.existsSync(path.join(this.projectRoot, 'config')),
      hasPublic: fs.existsSync(path.join(this.projectRoot, 'public')),
      hasAssets: fs.existsSync(path.join(this.projectRoot, 'assets')),
      depth: this.calculateDirectoryDepth(this.projectRoot)
    };
    
    this.projectConfig.structure = structure;
  }
  
  /**
   * 计算目录深度
   */
  calculateDirectoryDepth(dir, currentDepth = 0, maxDepth = 5) {
    if (currentDepth >= maxDepth) return currentDepth;
    
    try {
      const items = fs.readdirSync(dir);
      let maxChildDepth = currentDepth;
      
      for (const item of items) {
        if (item.startsWith('.') || item === 'node_modules') continue;
        
        const fullPath = path.join(dir, item);
        try {
          const stat = fs.statSync(fullPath);
          if (stat.isDirectory()) {
            const childDepth = this.calculateDirectoryDepth(fullPath, currentDepth + 1, maxDepth);
            maxChildDepth = Math.max(maxChildDepth, childDepth);
          }
        } catch (error) {
          // 忽略无法访问的目录
        }
      }
      
      return maxChildDepth;
    } catch (error) {
      return currentDepth;
    }
  }

  /**
   * 加载版本记录文件
   */
  loadRecordFile() {
    if (fs.existsSync(this.recordFile)) {
      try {
        const data = fs.readFileSync(this.recordFile, 'utf8');
        const record = JSON.parse(data);
        this.fileVersionMap = new Map(Object.entries(record.files || {}));
        console.log(`已加载版本记录: ${Object.keys(record.files || {}).length} 个文件`);
      } catch (error) {
        console.warn('版本记录文件加载失败:', error.message);
        this.fileVersionMap = new Map();
      }
    } else {
      console.log('未找到版本记录文件，将创建新的记录');
      this.fileVersionMap = new Map();
    }
  }

  /**
   * 保存版本记录文件
   */
  saveRecordFile() {
    const record = {
      lastUpdate: new Date().toISOString(),
      files: Object.fromEntries(this.fileVersionMap)
    };
    
    try {
      fs.writeFileSync(this.recordFile, JSON.stringify(record, null, 2), 'utf8');
      console.log('版本记录文件已保存');
      return true;
    } catch (error) {
      console.error('保存版本记录文件失败:', error.message);
      return false;
    }
  }

  /**
   * 递归扫描项目文件
   */
  async scanDirectory(dir, patterns, files) {
    const items = fs.readdirSync(dir);
    
    for (const item of items) {
      const fullPath = path.join(dir, item);
      const relativePath = path.relative(this.projectRoot, fullPath);
      
      // 检查系统版本信息排除模式
      if (this.shouldExcludeSystemVersion(relativePath)) {
        continue;
      }
      
      // 跳过系统目录和常见的不需要扫描的目录
      const skipDirs = [
        '.git', '.svn', '.hg',
        'node_modules', 'bower_components', 'vendor',
        'dist', 'build', 'target', 'out', 'output',
        '.next', '.nuxt', '.cache', '.temp', '.tmp',
        'coverage', '.nyc_output', 'test-results',
        'logs', '.logs', 'log', '.idea', '.vscode',
        '.DS_Store', 'Thumbs.db'
      ];
      
      // 检查是否为需要跳过的目录
      if (skipDirs.includes(item) || item.startsWith('.') && !item.startsWith('.version')) {
        continue;
      }
      
      try {
        const stat = fs.statSync(fullPath);
        
        if (stat.isDirectory()) {
          // 递归扫描子目录，但限制深度以避免无限递归
          if (relativePath.split(path.sep).length < 10) { // 最多10层深度
            await this.scanDirectory(fullPath, patterns, files);
          }
        } else if (stat.isFile() && this.isSourceFile(item)) {
          files.push(fullPath);
        }
      } catch (error) {
        // 忽略无法访问的文件或目录
        console.warn(`无法访问文件 ${fullPath}:`, error.message);
      }
    }
  }
  
  /**
   * 检查是否应该排除系统版本信息
   */
  shouldExcludeSystemVersion(filePath) {
    // 检查是否匹配系统排除模式
    for (const pattern of this.systemExclusionPatterns) {
      if (pattern.test(filePath)) {
        return true;
      }
    }
    
    // 额外检查：排除自身系统相关文件
    const systemKeywords = [
      'unified-version-system',
      'version-record-simple',
      'version-manager',
      'universal-version-manager',
      '.version-config',
      '.version-record'
    ];
    
    return systemKeywords.some(keyword => filePath.includes(keyword));
  }
  
  /**
   * 加载版本锁定配置
   */
  loadVersionLocking() {
    if (fs.existsSync(this.versionLocking.lockFile)) {
      try {
        const data = fs.readFileSync(this.versionLocking.lockFile, 'utf8');
        const lockConfig = JSON.parse(data);
        
        this.versionLocking.enabled = lockConfig.enabled || false;
        this.versionLocking.autoLock = lockConfig.autoLock || false;
        this.versionLocking.lockPatterns = lockConfig.lockPatterns || [];
        
        if (lockConfig.lockedVersions) {
          this.versionLocking.lockedVersions = new Map(Object.entries(lockConfig.lockedVersions));
        }
        
        console.log(`已加载版本锁定配置: ${this.versionLocking.lockedVersions.size} 个文件已锁定`);
      } catch (error) {
        console.warn('版本锁定文件加载失败:', error.message);
      }
    }
  }
  
  /**
   * 保存版本锁定配置
   */
  saveVersionLocking() {
    const lockConfig = {
      enabled: this.versionLocking.enabled,
      autoLock: this.versionLocking.autoLock,
      lockPatterns: this.versionLocking.lockPatterns,
      lockedVersions: Object.fromEntries(this.versionLocking.lockedVersions),
      lastUpdate: new Date().toISOString()
    };
    
    try {
      fs.writeFileSync(this.versionLocking.lockFile, JSON.stringify(lockConfig, null, 2), 'utf8');
      console.log('版本锁定配置已保存');
      return true;
    } catch (error) {
      console.error('保存版本锁定配置失败:', error.message);
      return false;
    }
  }
  
  /**
   * 启用版本锁定
   */
  enableVersionLocking() {
    this.versionLocking.enabled = true;
    console.log('版本锁定已启用');
    this.saveVersionLocking();
  }
  
  /**
   * 禁用版本锁定
   */
  disableVersionLocking() {
    this.versionLocking.enabled = false;
    console.log('版本锁定已禁用');
    this.saveVersionLocking();
  }
  
  /**
   * 锁定指定文件的版本
   */
  lockFileVersion(filePath, version) {
    const relativePath = path.relative(this.projectRoot, filePath);
    this.versionLocking.lockedVersions.set(relativePath, version);
    console.log(`已锁定文件版本: ${relativePath} -> ${version}`);
    this.saveVersionLocking();
  }
  
  /**
   * 解锁指定文件的版本
   */
  unlockFileVersion(filePath) {
    const relativePath = path.relative(this.projectRoot, filePath);
    this.versionLocking.lockedVersions.delete(relativePath);
    console.log(`已解锁文件版本: ${relativePath}`);
    this.saveVersionLocking();
  }
  
  /**
   * 检查文件版本是否已锁定
   */
  isFileVersionLocked(filePath) {
    const relativePath = path.relative(this.projectRoot, filePath);
    return this.versionLocking.lockedVersions.has(relativePath);
  }
  
  /**
   * 获取文件锁定的版本
   */
  getLockedVersion(filePath) {
    const relativePath = path.relative(this.projectRoot, filePath);
    return this.versionLocking.lockedVersions.get(relativePath);
  }
  
  /**
   * 添加锁定模式
   */
  addLockPattern(pattern) {
    if (!this.versionLocking.lockPatterns.includes(pattern)) {
      this.versionLocking.lockPatterns.push(pattern);
      console.log(`已添加锁定模式: ${pattern}`);
      this.saveVersionLocking();
    }
  }
  
  /**
   * 移除锁定模式
   */
  removeLockPattern(pattern) {
    const index = this.versionLocking.lockPatterns.indexOf(pattern);
    if (index > -1) {
      this.versionLocking.lockPatterns.splice(index, 1);
      console.log(`已移除锁定模式: ${pattern}`);
      this.saveVersionLocking();
    }
  }
  
  /**
   * 检查文件是否匹配锁定模式
   */
  matchesLockPattern(filePath) {
    const relativePath = path.relative(this.projectRoot, filePath);
    
    for (const pattern of this.versionLocking.lockPatterns) {
      try {
        const regex = new RegExp(pattern);
        if (regex.test(relativePath)) {
          return true;
        }
      } catch (error) {
        console.warn(`无效的锁定模式: ${pattern}`);
      }
    }
    
    return false;
  }
  
  /**
   * 自动锁定匹配模式的文件
   */
  autoLockMatchingFiles() {
    if (!this.versionLocking.autoLock) {
      console.log('自动锁定未启用');
      return;
    }
    
    let lockedCount = 0;
    
    for (const [filePath, version] of this.fileVersionMap.entries()) {
      if (this.matchesLockPattern(filePath) && !this.isFileVersionLocked(filePath)) {
        this.lockFileVersion(filePath, version);
        lockedCount++;
      }
    }
    
    console.log(`自动锁定了 ${lockedCount} 个文件`);
  }
  
  /**
   * 获取锁定状态摘要
   */
  getLockingSummary() {
    return {
      enabled: this.versionLocking.enabled,
      autoLock: this.versionLocking.autoLock,
      lockedFilesCount: this.versionLocking.lockedVersions.size,
      lockPatternsCount: this.versionLocking.lockPatterns.length,
      lockedFiles: Array.from(this.versionLocking.lockedVersions.keys()),
      lockPatterns: [...this.versionLocking.lockPatterns]
    };
  }

  /**
   * 判断是否为源文件
   */
  isSourceFile(filename) {
    const supportedExtensions = [
      // 编程语言
      '.js', '.ts', '.jsx', '.tsx', '.py', '.java', '.cs', '.cpp', '.cc', '.cxx', '.c', '.h', '.hpp',
      '.go', '.rs', '.php', '.rb', '.swift', '.kt', '.scala', '.dart', '.r', '.jl', '.m', '.pl',
      
      // Web技术
      '.html', '.htm', '.css', '.scss', '.sass', '.less', '.vue', '.svelte',
      
      // 配置文件
      '.json', '.yaml', '.yml', '.xml', '.ini', '.toml', '.env', '.properties', '.conf', '.cfg',
      
      // 文档格式
      '.md', '.markdown', '.txt', '.rst', '.adoc', '.tex', '.org',
      
      // 脚本文件
      '.sh', '.bash', '.zsh', '.fish', '.ps1', '.bat', '.cmd', '.sql',
      
      // 容器和部署文件
      'Dockerfile', 'docker-compose.yml', 'docker-compose.yaml', '.dockerfile',
      
      // 其他项目文件
      'Makefile', 'makefile', 'Gemfile', 'Pipfile', 'Cargo.toml', 'package.json'
    ];
    
    // 检查是否为支持的扩展名
    const ext = path.extname(filename).toLowerCase();
    const baseName = path.basename(filename);
    
    // 检查扩展名
    if (supportedExtensions.includes(ext)) {
      // 排除压缩和构建文件
      return !/\.(min|bundle|dist|build)\.(js|css)$/.test(filename);
    }
    
    // 检查特殊文件名
    const specialFiles = [
      'Dockerfile', 'docker-compose.yml', 'docker-compose.yaml', 'docker-compose.json',
      'Makefile', 'makefile', 'Gemfile', 'Pipfile', 'Cargo.toml', 'package.json'
    ];
    
    return specialFiles.includes(baseName);
  }

  /**
   * 扫描项目中所有源文件
   */
  async scanProjectFiles() {
    console.log('开始扫描项目文件...');
    
    const files = [];
    await this.scanDirectory(this.projectRoot, [], files);
    
    this.projectFiles = files;
    console.log(`扫描完成，共找到 ${files.length} 个源文件`);
    return files;
  }
  
  /**
   * 智能扫描项目（整合所有新功能）
   */
  async smartScanProject() {
    console.log('=== 开始智能版本扫描 ===');
    
    // 1. 项目自适应检测
    console.log('\n1. 执行项目自适应检测...');
    console.log('项目配置:', this.projectConfig);
    
    // 2. 扫描项目文件（包含系统版本信息排除）
    console.log('\n2. 扫描项目文件...');
    await this.scanProjectFiles();
    
    // 3. 分析文件版本信息
    console.log('\n3. 分析文件版本信息...');
    await this.analyzeFileVersions();
    
    // 4. 应用版本锁定
    console.log('\n4. 应用版本锁定策略...');
    if (this.versionLocking.enabled) {
      await this.applyVersionLocking();
    }
    
    // 5. 生成综合报告
    console.log('\n5. 生成扫描报告...');
    const report = this.generateSmartReport();
    
    console.log('\n=== 智能扫描完成 ===');
    return report;
  }
  
  /**
   * 应用版本锁定策略
   */
  async applyVersionLocking() {
    if (!this.versionLocking.enabled) {
      return;
    }
    
    console.log('应用版本锁定策略...');
    
    // 检查锁定冲突
    let conflicts = 0;
    for (const [filePath, currentVersion] of this.fileVersionMap.entries()) {
      if (this.isFileVersionLocked(filePath)) {
        const lockedVersion = this.getLockedVersion(filePath);
        if (lockedVersion !== currentVersion) {
          console.warn(`版本冲突: ${filePath} - 锁定版本: ${lockedVersion}, 当前版本: ${currentVersion}`);
          conflicts++;
        }
      }
    }
    
    if (conflicts > 0) {
      console.warn(`发现 ${conflicts} 个版本冲突，请检查锁定配置`);
    }
    
    // 自动锁定匹配模式的文件
    this.autoLockMatchingFiles();
    
    console.log('版本锁定策略应用完成');
  }
  
  /**
   * 生成智能扫描报告
   */
  generateSmartReport() {
    const summary = this.getVersionSummary();
    const lockingSummary = this.getLockingSummary();
    
    return {
      scanTime: new Date().toISOString(),
      projectInfo: {
        type: this.projectConfig.type,
        framework: this.projectConfig.framework,
        buildSystem: this.projectConfig.buildSystem,
        packageManager: this.projectConfig.packageManager,
        hasDocker: this.projectConfig.hasDocker,
        hasGit: this.projectConfig.hasGit,
        structure: this.projectConfig.structure
      },
      versionInfo: summary,
      lockingInfo: lockingSummary,
      systemExclusion: {
        enabled: true,
        excludedPatterns: this.systemExclusionPatterns.length
      },
      recommendations: this.generateRecommendations()
    };
  }
  
  /**
   * 生成智能建议
   */
  generateRecommendations() {
    const recommendations = [];
    
    // 基于项目类型的建议
    if (this.projectConfig.framework) {
      recommendations.push({
        type: 'framework',
        message: `检测到 ${this.projectConfig.framework} 项目，建议使用相应的版本管理模式`
      });
    }
    
    // 基于锁定状态的建议
    if (this.versionLocking.enabled && this.versionLocking.lockedVersions.size === 0) {
      recommendations.push({
        type: 'locking',
        message: '版本锁定已启用但没有锁定任何文件，建议添加锁定模式'
      });
    }
    
    // 基于文件数量的建议
    if (this.projectFiles.length > 1000) {
      recommendations.push({
        type: 'performance',
        message: '项目文件较多，建议配置排除模式以提高扫描性能'
      });
    }
    
    // 基于版本一致性的建议
    const versions = Array.from(this.fileVersionMap.values());
    const uniqueVersions = [...new Set(versions)];
    if (uniqueVersions.length > 5) {
      recommendations.push({
        type: 'consistency',
        message: `发现 ${uniqueVersions.length} 个不同版本，建议统一版本管理`
      });
    }
    
    return recommendations;
  }

  /**
   * 从文件内容中提取版本信息
   */
  extractVersionFromContent(content, filename) {
    const fileExt = path.extname(filename).toLowerCase();
    const baseName = path.basename(filename);
    
    // 编程语言文件
    if (fileExt.match(/\.(js|ts|jsx|tsx|py|java|cs|cpp|cc|cxx|c|h|hpp|go|rs|php|rb|swift|kt|scala|dart|r|jl|m|pl)$/)) {
      return this.extractVersionFromCode(content, filename);
    }
    
    // 配置文件
    if (fileExt.match(/\.(json|yaml|yml|xml|ini|toml|env|properties|conf|cfg)$/)) {
      return this.extractVersionFromConfig(content, filename);
    }
    
    // Web技术文件
    if (fileExt.match(/\.(html|htm|css|scss|sass|less|vue|svelte)$/)) {
      return this.extractVersionFromWeb(content, filename);
    }
    
    // 文档文件
    if (fileExt.match(/\.(md|markdown|txt|rst|adoc|tex|org)$/)) {
      return this.extractVersionFromDocument(content, filename);
    }
    
    // 脚本文件
    if (fileExt.match(/\.(sh|bash|zsh|fish|ps1|bat|cmd|sql)$/)) {
      return this.extractVersionFromScript(content, filename);
    }
    
    // 特殊文件
    if (['Dockerfile', 'docker-compose.yml', 'docker-compose.yaml', 'docker-compose.json'].includes(baseName)) {
      return this.extractVersionFromDocker(content, filename);
    }
    
    if (['Makefile', 'makefile', 'Gemfile', 'Pipfile', 'Cargo.toml', 'package.json'].includes(baseName)) {
      return this.extractVersionFromConfig(content, filename);
    }
    
    return null;
  }

  /**
   * 从代码文件中提取版本信息
   */
  extractVersionFromCode(content, filename) {
    const patterns = [
      /version\s*[:=]\s*['"]([^'"]+)['"]/gi,
      /VERSION\s*[:=]\s*['"]([^'"]+)['"]/gi,
      /@version\s+([^\s]+)/gi,
      /const\s+version\s*=\s*['"]([^'"]+)['"]/gi,
      /export\s+const\s+version\s*=\s*['"]([^'"]+)['"]/gi,
      /export\s+default\s+{[^}]*version\s*:\s*['"]([^'"]+)['"]/gi,
      /@author\s+([^\n]+)/gi
    ];

    for (const pattern of patterns) {
      const match = pattern.exec(content);
      if (match && match[1]) {
        const version = match[1];
        if (this.isValidVersion(version)) {
          return {
            version,
            pattern: pattern.source,
            matchText: match[0],
            type: 'code'
          };
        }
      }
    }

    return null;
  }

  /**
   * 从文档文件中提取版本信息
   */
  extractVersionFromDocument(content, filename) {
    const patterns = [
      // 版本号在标题中
      /^#.*\b(v\d+\.\d+\.\d+)\b.*$/gmi,
      // 版本信息在表格中
      /\|.*\b(v\d+\.\d+\.\d+)\b.*\|/gmi,
      // 版本号在文本中
      /\bversion[:\s]+(v\d+\.\d+\.\d+)\b/gi,
      // 版本历史记录中的版本号
      /## 版本历史.*?(v\d+\.\d+\.\d+)/gmi,
      // Markdown文档顶部的版本信息
      /^(?:---|\*\*\*)\s*[\r\n]+.*?version:\s*(v\d+\.\d+\.\d+)/gmi,
      // 文档末尾的版本信息
      /版本[:\s]+(v\d+\.\d+\.\d+)/gi,
      // 变更日志格式
      /###\s*(v\d+\.\d+\.\d+)/gmi,
      // 文档元数据中的版本
      /"version"\s*:\s*"(v\d+\.\d+\.\d+)"/gmi
    ];

    for (const pattern of patterns) {
      const match = pattern.exec(content);
      if (match && match[1]) {
        const version = match[1];
        if (this.isValidVersion(version)) {
          return {
            version,
            pattern: pattern.source,
            matchText: match[0],
            type: 'document'
          };
        }
      }
    }

    return null;
  }

  /**
   * 从配置文件中提取版本信息
   */
  extractVersionFromConfig(content, filename) {
    const patterns = [
      // JSON配置中的版本
      /"version"\s*:\s*"(v\d+\.\d+\.\d+)"/gmi,
      // XML配置中的版本
      /<version>(v\d+\.\d+\.\d+)<\/version>/gmi,
      // YAML配置中的版本
      /version:\s*(v\d+\.\d+\.\d+)/gmi,
      /version\s*:\s*(v\d+\.\d+\.\d+)/gmi,
      // INI配置中的版本
      /version=(v\d+\.\d+\.\d+)/gmi,
      // TOML配置中的版本
      /version\s*=\s*"(v\d+\.\d+\.\d+)"/gmi,
      // 环境变量中的版本
      /VERSION\s*=\s*(v\d+\.\d+\.\d+)/gmi,
      // Makefile中的版本
      /VERSION\s*:?=\s*(v\d+\.\d+\.\d+)/gmi,
      // Cargo.toml中的版本
      /version\s*=\s*"(v\d+\.\d+\.\d+)"/gmi
    ];

    for (const pattern of patterns) {
      const match = pattern.exec(content);
      if (match && match[1]) {
        const version = match[1];
        if (this.isValidVersion(version)) {
          return {
            version,
            pattern: pattern.source,
            matchText: match[0],
            type: 'config'
          };
        }
      }
    }

    return null;
  }

  /**
   * 从Web技术文件中提取版本信息
   */
  extractVersionFromWeb(content, filename) {
    const patterns = [
      // HTML中的版本信息
      /<meta\s+name=["']version["']\s+content=["'](v\d+\.\d+\.\d+)["']/gmi,
      /<meta\s+content=["'](v\d+\.\d+\.\d+)["']\s+name=["']version["']/gmi,
      // HTML注释中的版本
      /<!--\s*version[:\s]+(v\d+\.\d+\.\d+)\s*-->/gmi,
      // CSS中的版本信息
      /\/\*\s*version[:\s]+(v\d+\.\d+\.\d+)\s*\*\//gmi,
      // Vue/Svelte组件中的版本
      /version[:\s]+['"](v\d+\.\d+\.\d+)['"]/gmi,
      // Webpack或其他构建工具的版本标记
      /@version\s+(v\d+\.\d+\.\d+)/gmi
    ];

    for (const pattern of patterns) {
      const match = pattern.exec(content);
      if (match && match[1]) {
        const version = match[1];
        if (this.isValidVersion(version)) {
          return {
            version,
            pattern: pattern.source,
            matchText: match[0],
            type: 'web'
          };
        }
      }
    }

    return null;
  }

  /**
   * 从脚本文件中提取版本信息
   */
  extractVersionFromScript(content, filename) {
    const patterns = [
      // 脚本中的版本变量
      /VERSION\s*=\s*(v\d+\.\d+\.\d+)/gmi,
      /version\s*=\s*(v\d+\.\d+\.\d+)/gmi,
      // 注释中的版本信息
      /#\s*version[:\s]+(v\d+\.\d+\.\d+)/gmi,
      /\/\/\s*version[:\s]+(v\d+\.\d+\.\d+)/gmi,
      /\/\*\s*version[:\s]+(v\d+\.\d+\.\d+)\s*\*\//gmi,
      // SQL中的版本信息
      /--\s*version[:\s]+(v\d+\.\d+\.\d+)/gmi
    ];

    for (const pattern of patterns) {
      const match = pattern.exec(content);
      if (match && match[1]) {
        const version = match[1];
        if (this.isValidVersion(version)) {
          return {
            version,
            pattern: pattern.source,
            matchText: match[0],
            type: 'script'
          };
        }
      }
    }

    return null;
  }

  /**
   * 从Docker文件中提取版本信息
   */
  extractVersionFromDocker(content, filename) {
    const patterns = [
      // Docker标签中的版本
      /LABEL\s+version\s*=\s*["']?(v\d+\.\d+\.\d+)["']?/gmi,
      /LABEL\s+["']?version["']?\s*=\s*["']?(v\d+\.\d+\.\d+)["']?/gmi,
      // 注释中的版本信息
      /#\s*version[:\s]+(v\d+\.\d+\.\d+)/gmi,
      // FROM指令中的版本（基础镜像版本）
      /FROM\s+.*:(v\d+\.\d+\.\d+)/gmi,
      // ENV环境变量中的版本
      /ENV\s+.*VERSION\s*=\s*["']?(v\d+\.\d+\.\d+)["']?/gmi
    ];

    for (const pattern of patterns) {
      const match = pattern.exec(content);
      if (match && match[1]) {
        const version = match[1];
        if (this.isValidVersion(version)) {
          return {
            version,
            pattern: pattern.source,
            matchText: match[0],
            type: 'docker'
          };
        }
      }
    }

    return null;
  }

  /**
   * 验证版本号格式
   */
  isValidVersion(version) {
    return /^v\d+\.\d+\.\d+/.test(version);
  }

  /**
   * 比较两个版本号
   * 返回值：> 0 表示 v1 > v2，< 0 表示 v1 < v2，= 0 表示相等
   */
  compareVersions(v1, v2) {
    const parts1 = v1.match(/v(\d+)\.(\d+)\.(\d+)/);
    const parts2 = v2.match(/v(\d+)\.(\d+)\.(\d+)/);
    
    if (!parts1 || !parts2) return 0;
    
    const major1 = parseInt(parts1[1]);
    const minor1 = parseInt(parts1[2]);
    const patch1 = parseInt(parts1[3]);
    
    const major2 = parseInt(parts2[1]);
    const minor2 = parseInt(parts2[2]);
    const patch2 = parseInt(parts2[3]);
    
    if (major1 !== major2) return major1 - major2;
    if (minor1 !== minor2) return minor1 - minor2;
    return patch1 - patch2;
  }

  /**
   * 分析文件版本信息
   */
  async analyzeFileVersions() {
    console.log('开始分析文件版本信息...');
    
    let updatedCount = 0;
    
    for (const filePath of this.projectFiles) {
      try {
        const content = fs.readFileSync(filePath, 'utf8');
        const relativePath = path.relative(this.projectRoot, filePath);
        
        const versionInfo = this.extractVersionFromContent(content, filePath);
        
        if (versionInfo) {
          this.fileVersionMap.set(relativePath, versionInfo.version);
          console.log(`发现版本: ${relativePath} -> ${versionInfo.version}`);
        } else {
          console.log(`未发现版本: ${relativePath}`);
        }
      } catch (error) {
        console.warn(`分析文件失败 ${filePath}:`, error.message);
      }
    }
    
    console.log(`版本分析完成，共分析 ${this.projectFiles.length} 个文件`);
    return updatedCount;
  }

  /**
   * 获取项目版本信息摘要
   */
  getVersionSummary() {
    const versions = Array.from(this.fileVersionMap.values());
    const uniqueVersions = [...new Set(versions)];
    
    return {
      totalFiles: this.fileVersionMap.size,
      uniqueVersions: uniqueVersions.length,
      versions: uniqueVersions,
      filesWithVersions: this.fileVersionMap.size,
      lastUpdate: new Date().toISOString()
    };
  }

  /**
   * 检查版本记录文件的完整性和一致性
   */
  checkVersionRecord() {
    console.log('\n=== 版本记录检查报告 ===\n');
    
    const report = {
      valid: true,
      errors: [],
      warnings: [],
      info: [],
      checks: {}
    };

    // 1. 检查版本格式
    console.log('1. 检查版本格式...');
    const formatErrors = [];
    for (const [file, version] of this.fileVersionMap.entries()) {
      const formatCheck = this.versionChecker.checkVersionFormat(version);
      if (!formatCheck.valid) {
        formatErrors.push(`${file}: ${formatCheck.error}`);
        report.valid = false;
      }
    }
    report.checks.format = { errors: formatErrors };
    if (formatErrors.length > 0) {
      console.log(`   ❌ 发现 ${formatErrors.length} 个格式错误:`);
      formatErrors.forEach(err => console.log(`      - ${err}`));
      report.errors.push(...formatErrors);
    } else {
      console.log('   ✅ 所有版本格式正确');
      report.info.push('所有版本格式正确');
    }

    // 2. 检查版本号数字限制
    console.log('\n2. 检查版本号数字限制...');
    const limitErrors = [];
    const limitWarnings = [];
    for (const [file, version] of this.fileVersionMap.entries()) {
      const limitsCheck = this.versionChecker.checkVersionLimits(version);
      if (!limitsCheck.valid) {
        limitsCheck.errors.forEach(err => {
          limitErrors.push(`${file} (${version}): ${err}`);
        });
        report.valid = false;
      }
      if (limitsCheck.warnings.length > 0) {
        limitsCheck.warnings.forEach(warn => {
          limitWarnings.push(`${file} (${version}): ${warn}`);
        });
      }
    }
    report.checks.limits = { errors: limitErrors, warnings: limitWarnings };
    if (limitErrors.length > 0) {
      console.log(`   ❌ 发现 ${limitErrors.length} 个限制错误:`);
      limitErrors.forEach(err => console.log(`      - ${err}`));
      report.errors.push(...limitErrors);
    } else {
      console.log('   ✅ 所有版本号在限制范围内');
      report.info.push('所有版本号在限制范围内');
    }
    if (limitWarnings.length > 0) {
      console.log(`   ⚠️  发现 ${limitWarnings.length} 个警告:`);
      limitWarnings.forEach(warn => console.log(`      - ${warn}`));
      report.warnings.push(...limitWarnings);
    }

    // 3. 检查版本重复使用（多个文件使用相同版本号）
    console.log('\n3. 检查版本重复使用...');
    const versionUsage = new Map();
    for (const [file, version] of this.fileVersionMap.entries()) {
      if (!versionUsage.has(version)) {
        versionUsage.set(version, []);
      }
      versionUsage.get(version).push(file);
    }
    
    const duplicateVersions = [];
    for (const [version, files] of versionUsage.entries()) {
      if (files.length > 1) {
        duplicateVersions.push({ version, files, count: files.length });
      }
    }
    
    report.checks.duplicateUsage = { duplicates: duplicateVersions };
    if (duplicateVersions.length > 0) {
      console.log(`   ⚠️  发现 ${duplicateVersions.length} 个版本号被多个文件使用:`);
      duplicateVersions.forEach(dup => {
        console.log(`      - ${dup.version} (${dup.count} 个文件):`);
        dup.files.forEach(file => console.log(`          * ${file}`));
      });
      report.warnings.push(`${duplicateVersions.length} 个版本号被多个文件使用`);
    } else {
      console.log('   ✅ 每个版本号仅被一个文件使用');
      report.info.push('每个版本号仅被一个文件使用');
    }

    // 4. 检查版本一致性（是否有过多不同版本）
    console.log('\n4. 检查版本一致性...');
    const uniqueVersions = [...versionUsage.keys()];
    const totalFiles = this.fileVersionMap.size;
    const versionDiversity = uniqueVersions.length / totalFiles;
    
    report.checks.consistency = {
      totalFiles,
      uniqueVersions: uniqueVersions.length,
      diversity: versionDiversity
    };
    
    if (uniqueVersions.length > 10) {
      console.log(`   ⚠️  版本数量较多 (${uniqueVersions.length} 个不同版本)，建议统一版本管理`);
      report.warnings.push(`版本数量较多 (${uniqueVersions.length} 个)，建议统一管理`);
    } else if (versionDiversity > 0.8) {
      console.log(`   ⚠️  版本分散度较高 (${(versionDiversity * 100).toFixed(1)}%)，建议统一版本号`);
      report.warnings.push(`版本分散度较高 (${(versionDiversity * 100).toFixed(1)}%)`);
    } else {
      console.log(`   ✅ 版本一致性良好 (${uniqueVersions.length} 个不同版本)`);
      report.info.push(`版本一致性良好 (${uniqueVersions.length} 个不同版本)`);
    }

    // 5. 检查 CHANGELOG.md 内部版本号重复
    console.log('\n5. 检查 CHANGELOG.md 内部版本号重复...');
    const changelogPath = path.join(this.projectRoot, 'CHANGELOG.md');
    const changelogDuplicates = [];
    
    if (fs.existsSync(changelogPath)) {
      try {
        const content = fs.readFileSync(changelogPath, 'utf8');
        const versionPattern = /^## \[(v\d+\.\d+\.\d+)\]/gm;
        const foundVersions = new Map();
        let match;
        
        while ((match = versionPattern.exec(content)) !== null) {
          const version = match[1];
          const lineNumber = content.substring(0, match.index).split('\n').length;
          
          if (!foundVersions.has(version)) {
            foundVersions.set(version, []);
          }
          foundVersions.get(version).push(lineNumber);
        }
        
        for (const [version, lines] of foundVersions.entries()) {
          if (lines.length > 1) {
            changelogDuplicates.push({
              version,
              count: lines.length,
              lines
            });
          }
        }
        
        report.checks.changelogDuplicates = { duplicates: changelogDuplicates };
        
        if (changelogDuplicates.length > 0) {
          console.log(`   ❌ 发现 ${changelogDuplicates.length} 个重复的版本号:`);
          changelogDuplicates.forEach(dup => {
            console.log(`      - ${dup.version} 出现 ${dup.count} 次，位于第 ${dup.lines.join(', ')} 行`);
          });
          report.errors.push(`CHANGELOG.md 中有 ${changelogDuplicates.length} 个重复版本号`);
          report.valid = false;
        } else {
          console.log('   ✅ CHANGELOG.md 中没有重复的版本号');
          report.info.push('CHANGELOG.md 中没有重复的版本号');
        }
      } catch (error) {
        console.log(`   ⚠️  无法检查 CHANGELOG.md: ${error.message}`);
        report.warnings.push(`无法检查 CHANGELOG.md: ${error.message}`);
      }
    } else {
      console.log('   ℹ️  未找到 CHANGELOG.md 文件');
      report.info.push('未找到 CHANGELOG.md 文件');
    }

    // 6. 检查 CHANGELOG.md 版本顺序（应该从新到旧）
    console.log('\n6. 检查 CHANGELOG.md 版本顺序...');
    if (fs.existsSync(changelogPath)) {
      try {
        const content = fs.readFileSync(changelogPath, 'utf8');
        const versionPattern = /^## \[(v\d+\.\d+\.\d+)\]/gm;
        const versions = [];
        let match;
        
        while ((match = versionPattern.exec(content)) !== null) {
          const version = match[1];
          const lineNumber = content.substring(0, match.index).split('\n').length;
          versions.push({ version, line: lineNumber });
        }
        
        const orderErrors = [];
        for (let i = 0; i < versions.length - 1; i++) {
          const current = versions[i].version;
          const next = versions[i + 1].version;
          
          if (this.compareVersions(current, next) < 0) {
            orderErrors.push({
              current: { version: current, line: versions[i].line },
              next: { version: next, line: versions[i + 1].line }
            });
          }
        }
        
        report.checks.versionOrder = { errors: orderErrors };
        
        if (orderErrors.length > 0) {
          console.log(`   ❌ 发现 ${orderErrors.length} 处版本顺序错误（应该从新到旧）:`);
          orderErrors.forEach(err => {
            console.log(`      - 第 ${err.current.line} 行 ${err.current.version} 应该在第 ${err.next.line} 行 ${err.next.version} 之后`);
          });
          report.errors.push(`CHANGELOG.md 版本顺序错误 (${orderErrors.length} 处)`);
          report.valid = false;
        } else {
          console.log('   ✅ CHANGELOG.md 版本顺序正确（从新到旧）');
          report.info.push('CHANGELOG.md 版本顺序正确');
        }
      } catch (error) {
        console.log(`   ⚠️  无法检查版本顺序: ${error.message}`);
        report.warnings.push(`无法检查版本顺序: ${error.message}`);
      }
    }

    // 7. 检查版本号缺失（检查序列连续性）
    console.log('\n7. 检查版本号序列连续性...');
    if (fs.existsSync(changelogPath)) {
      try {
        const content = fs.readFileSync(changelogPath, 'utf8');
        const versionPattern = /^## \[(v\d+\.\d+\.\d+)\]/gm;
        const versions = [];
        let match;
        
        while ((match = versionPattern.exec(content)) !== null) {
          versions.push(match[1]);
        }
        
        // 按版本号分组
        const versionGroups = new Map();
        versions.forEach(v => {
          const parts = v.match(/v(\d+)\.(\d+)\.(\d+)/);
          const major = parseInt(parts[1]);
          const minor = parseInt(parts[2]);
          const patch = parseInt(parts[3]);
          const key = `${major}.${minor}`;
          
          if (!versionGroups.has(key)) {
            versionGroups.set(key, []);
          }
          versionGroups.get(key).push(patch);
        });
        
        const missingVersions = [];
        for (const [key, patches] of versionGroups.entries()) {
          patches.sort((a, b) => a - b);
          for (let i = 0; i < patches.length - 1; i++) {
            const current = patches[i];
            const next = patches[i + 1];
            if (next - current > 1) {
              for (let missing = current + 1; missing < next; missing++) {
                missingVersions.push(`v${key}.${missing}`);
              }
            }
          }
        }
        
        report.checks.missingVersions = { missing: missingVersions };
        
        if (missingVersions.length > 0) {
          console.log(`   ❌ 发现 ${missingVersions.length} 个缺失的版本号（版本号不能跳过）:`);
          missingVersions.forEach(v => {
            console.log(`      - ${v}`);
          });
          report.errors.push(`缺失 ${missingVersions.length} 个版本号（版本号必须连续）`);
          report.valid = false;
        } else {
          console.log('   ✅ 版本号序列连续，无缺失');
          report.info.push('版本号序列连续');
        }
      } catch (error) {
        console.log(`   ⚠️  无法检查版本缺失: ${error.message}`);
        report.warnings.push(`无法检查版本缺失: ${error.message}`);
      }
    }

    // 5. 生成摘要
    console.log('\n=== 检查摘要 ===');
    console.log(`总文件数: ${totalFiles}`);
    console.log(`不同版本数: ${uniqueVersions.length}`);
    console.log(`错误: ${report.errors.length}`);
    console.log(`警告: ${report.warnings.length}`);
    console.log(`状态: ${report.valid ? '✅ 通过' : '❌ 失败'}`);
    
    console.log('\n');
    
    return report;
  }

  /**
   * 导出版本记录模块
   */
  exportVersionModule(outputPath = 'version-record-module.js') {
    const summary = this.getVersionSummary();
    const moduleContent = `/**
 * 版本记录模块
 * 自动生成的版本信息汇总
 * 最后更新: ${summary.lastUpdate}
 */

const versionRecord = ${JSON.stringify(summary, null, 2)};

module.exports = versionRecord;

// 版本信息
console.log('版本记录模块已加载');
console.log('总文件数:', versionRecord.totalFiles);
console.log('唯一版本数:', versionRecord.uniqueVersions);
console.log('版本列表:', versionRecord.versions.join(', '));
`;

    try {
      const fullPath = path.resolve(outputPath);
      fs.writeFileSync(fullPath, moduleContent, 'utf8');
      console.log(`版本记录模块已导出到: ${fullPath}`);
      return true;
    } catch (error) {
      console.error('导出版本记录模块失败:', error.message);
      return false;
    }
  }

  /**
   * 自动更新版本记录
   */
  async autoUpdate() {
    console.log('开始自动更新版本记录...');
    
    // 1. 扫描文件
    await this.scanProjectFiles();
    
    // 2. 分析版本
    await this.analyzeFileVersions();
    
    // 3. 保存记录
    this.saveRecordFile();
    
    // 4. 显示摘要
    const summary = this.getVersionSummary();
    console.log('版本记录更新完成:');
    console.log(`  - 总文件数: ${summary.totalFiles}`);
    console.log(`  - 唯一版本数: ${summary.uniqueVersions}`);
    console.log(`  - 版本列表: ${summary.versions.join(', ')}`);
    
    return summary;
  }
}

// 主程序
async function main() {
  const args = process.argv.slice(2);
  const command = args[0] || 'help';
  
  const manager = new SimpleVersionRecordManager();
  
  switch (command) {
    case 'scan':
      await manager.scanProjectFiles();
      break;
      
    case 'analyze':
      await manager.analyzeFileVersions();
      break;
      
    case 'auto':
      await manager.autoUpdate();
      break;
      
    case 'info':
      const summary = manager.getVersionSummary();
      console.log('版本记录信息:');
      console.log(JSON.stringify(summary, null, 2));
      break;
      
    case 'check':
      manager.checkVersionRecord();
      break;
      
    case 'export':
      const outputPath = args[1] || 'version-record-module.js';
      manager.exportVersionModule(outputPath);
      break;
      
    case 'smart-scan':
      const report = await manager.smartScanProject();
      console.log('\n=== 智能扫描报告 ===');
      console.log(JSON.stringify(report, null, 2));
      break;
      
    case 'project-info':
      console.log('=== 项目自适应信息 ===');
      console.log('项目配置:', JSON.stringify(manager.projectConfig, null, 2));
      break;
      
    case 'lock':
      const fileToLock = args[1];
      const versionToLock = args[2];
      if (fileToLock && versionToLock) {
        manager.lockFileVersion(fileToLock, versionToLock);
      } else {
        console.log('用法: node version-record-simple.js lock <文件路径> <版本号>');
      }
      break;
      
    case 'unlock':
      const fileToUnlock = args[1];
      if (fileToUnlock) {
        manager.unlockFileVersion(fileToUnlock);
      } else {
        console.log('用法: node version-record-simple.js unlock <文件路径>');
      }
      break;
      
    case 'lock-info':
      const lockingSummary = manager.getLockingSummary();
      console.log('=== 版本锁定信息 ===');
      console.log(JSON.stringify(lockingSummary, null, 2));
      break;
      
    case 'lock-enable':
      manager.enableVersionLocking();
      break;
      
    case 'lock-disable':
      manager.disableVersionLocking();
      break;
      
    case 'add-pattern':
      const pattern = args[1];
      if (pattern) {
        manager.addLockPattern(pattern);
      } else {
        console.log('用法: node version-record-simple.js add-pattern <正则模式>');
      }
      break;
      
    case 'remove-pattern':
      const patternToRemove = args[1];
      if (patternToRemove) {
        manager.removeLockPattern(patternToRemove);
      } else {
        console.log('用法: node version-record-simple.js remove-pattern <正则模式>');
      }
      break;
      
    case 'auto-lock':
      manager.versionLocking.autoLock = true;
      manager.autoLockMatchingFiles();
      manager.saveVersionLocking();
      break;
      
    case 'help':
    default:
      console.log('通用版本记录管理系统 v2.0 (增强版)');
      console.log('');
      console.log('用法: node version-record-simple.js <命令> [参数]');
      console.log('');
      console.log('基础命令:');
      console.log('  scan         扫描项目中所有源文件');
      console.log('  analyze      分析文件中的版本信息');
      console.log('  auto         自动更新版本记录 (推荐)');
      console.log('  smart-scan   智能扫描 (整合所有新功能)');
      console.log('  info         显示版本记录摘要信息');
      console.log('  check        检查版本记录文件的完整性和一致性');
      console.log('  export       导出版本记录模块 [输出文件路径]');
      console.log('');
      console.log('项目自适应命令:');
      console.log('  project-info 显示项目自适应检测信息');
      console.log('');
      console.log('版本锁定命令:');
      console.log('  lock-enable      启用版本锁定');
      console.log('  lock-disable     禁用版本锁定');
      console.log('  lock             锁定指定文件版本 <文件路径> <版本号>');
      console.log('  unlock           解锁指定文件版本 <文件路径>');
      console.log('  lock-info        显示版本锁定状态信息');
      console.log('  add-pattern      添加锁定模式 <正则表达式>');
      console.log('  remove-pattern   移除锁定模式 <正则表达式>');
      console.log('  auto-lock        启用自动锁定匹配模式的文件');
      console.log('');
      console.log('  help         显示此帮助信息');
      break;
  }
}

// 如果直接运行此脚本
if (require.main === module) {
  main().catch(error => {
    console.error('执行过程中发生错误:', error);
    process.exit(1);
  });
}

module.exports = SimpleVersionRecordManager;