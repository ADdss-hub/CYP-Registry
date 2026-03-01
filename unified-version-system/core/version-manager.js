#!/usr/bin/env node

/**
 * 统一版本管理系统 (Core Version Manager)
 * 整合了版本规范、检查、记录和自动管理的完整解决方案
 * 
 * 功能特性：
 * - 语义化版本管理 (SemVer 2.0.0 with v-prefix)
 * - 强制数字限制 (MAJOR: 0-99, MINOR: 0-50, PATCH: 0-30)
 * - 变更类型规范和验证
 * - 项目自适应配置
 * - 版本记录自动查找和同步
 * - 统一版本服务API
 * 
 * @author CYP (nasDSSCYP@outlook.com)
 * @version v2.1.0
 */

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

// 核心模块导入
const VersionChecker = require('./version-checker.js');
const VersionUpdater = require('./version-updater.js');
const VersionAPI = require('./version-api.js');
const VersionConfig = require('./version-config.js');
const VersionRecordManager = require('../version-record-simple.js');

/**
 * 统一版本管理器
 * 整合所有版本管理功能的核心类
 */
class UniversalVersionManager {
  constructor(options = {}) {
    this.projectRoot = path.resolve(options.projectRoot || process.cwd());
    this.options = {
      ...options,
      projectRoot: this.projectRoot
    };

    // 初始化核心组件
    this.config = new VersionConfig(this.projectRoot);
    this.api = new VersionAPI(this.options);
    this.checker = new VersionChecker({
      ...this.options,
      projectName: this.config.get('project.name'),
      config: this.config.getConfig()
    });
    this.updater = new VersionUpdater({
      ...this.options,
      projectName: this.config.get('project.name'),
      config: this.config.getConfig()
    });
    this.recordManager = new VersionRecordManager(this.projectRoot);

    // 统一版本管理状态
    this.versionState = {
      current: null,
      history: [],
      records: null
    };

    this.loadVersionState();
  }

  /**
   * 加载版本状态
   */
  loadVersionState() {
    try {
      // 加载当前版本
      const versionFile = path.join(this.projectRoot, '.version', 'version.json');
      if (fs.existsSync(versionFile)) {
        const data = JSON.parse(fs.readFileSync(versionFile, 'utf8'));
        this.versionState.current = data;
      }

      // 加载版本历史
      const historyFile = path.join(this.projectRoot, '.version', 'changelog.json');
      if (fs.existsSync(historyFile)) {
        const data = JSON.parse(fs.readFileSync(historyFile, 'utf8'));
        this.versionState.history = data.history || [];
      }

      // 加载版本记录
      const recordInfo = this.recordManager.getVersionSummary();
      this.versionState.records = recordInfo;

    } catch (error) {
      console.warn('加载版本状态失败:', error.message);
    }
  }

  /**
   * 初始化版本系统
   */
  init(version: "v1.1.0", options = {}) {
    console.log('🚀 初始化统一版本管理系统...');

    // 1. 初始化配置
    this.config.initConfig(options);
    
    // 2. 初始化版本
    this.updater.initializeVersion(version, options);
    
    // 3. 初始化版本记录
    this.recordManager.autoUpdate();
    
    // 4. 生成版本模块
    this.api.generateVersionModule();
    
    // 5. 更新package.json
    this.updatePackageJson(version);

    console.log('✅ 版本系统初始化完成!');
    this.showStatus();
  }

  /**
   * 验证并更新版本
   */
  async validateAndUpdate(version, options = {}) {
    console.log(`🔄 验证并更新版本到 ${version}...`);

    try {
      // 1. 验证版本格式和规则
      const changes = options.changes || [];
      const report = this.checker.checkVersion(version, changes);
      this.checker.printReport(report);

      if (!report.valid) {
        console.error('❌ 版本验证失败，无法更新版本');
        return false;
      }

      // 2. 更新版本
      this.updater.updateVersion(version, options);

      // 3. 更新版本记录
      this.recordManager.autoUpdate();

      // 4. 生成统一版本模块
      this.api.generateVersionModule();

      // 5. 导出版本记录
      this.recordManager.exportVersionModule();

      // 6. 更新其他文件的版本信息
      this.updateVersionInFiles(version);

      // 7. 更新package.json
      this.updatePackageJson(version);

      // 8. 重新加载状态
      this.loadVersionState();

      console.log('✅ 版本更新完成!');
      this.showStatus();
      return true;

    } catch (error) {
      console.error('❌ 版本更新失败:', error.message);
      return false;
    }
  }

  /**
   * 自动递增版本
   */
  autoIncrement(level = 'patch', options = {}) {
    if (!this.versionState.current) {
      console.error('❌ 未找到当前版本，请先初始化版本系统');
      return null;
    }

    const currentVersion = this.versionState.current.version;
    const newVersion = this.updater.incrementVersion(currentVersion, level);

    if (newVersion) {
      console.log(`📈 自动递增版本: ${currentVersion} -> ${newVersion}`);
      
      // 自动设置变更类型
      const changeType = this.getChangeTypeByLevel(level);
      options.changes = options.changes || [{
        type: changeType,
        description: `自动递增 ${level} 版本`
      }];

      return this.validateAndUpdate(newVersion, options);
    }

    return false;
  }

  /**
   * 根据递增级别获取变更类型
   */
  getChangeTypeByLevel(level) {
    const typeMap = {
      'major': 'feat',     // 新功能
      'minor': 'feat',     // 新功能
      'patch': 'fix'       // 修复
    };
    return typeMap[level] || 'chore';
  }

  /**
   * 扫描和同步所有JS文件
   */
  syncAllFiles() {
    console.log('🔄 同步所有文件版本信息...');
    
    const currentVersion = this.getCurrentVersion();
    if (currentVersion) {
      this.updateVersionInFiles(currentVersion);
      console.log('✅ 文件版本同步完成');
    } else {
      console.error('❌ 未找到当前版本');
    }
  }

  /**
   * 获取当前版本信息
   */
  getCurrentVersion() {
    return this.versionState.current;
  }

  /**
   * 显示版本状态
   */
  showStatus() {
    console.log('\n📊 版本状态:');
    console.log(`   当前版本: ${this.versionState.current?.version || '未设置'}`);
    console.log(`   项目名称: ${this.config.get('project.name')}`);
    console.log(`   版本文件: ${this.versionState.records?.totalFiles || 0} 个`);
    console.log(`   唯一版本: ${this.versionState.records?.uniqueVersions || 0} 个`);
    
    if (this.versionState.history.length > 0) {
      console.log(`   历史记录: ${this.versionState.history.length} 条`);
    }
  }

  /**
   * 更新package.json中的版本
   */
  updatePackageJson(version) {
    const packagePath = path.join(this.projectRoot, 'package.json');
    if (fs.existsSync(packagePath)) {
      try {
        const packageData = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
        packageData.version = version;
        fs.writeFileSync(packagePath, JSON.stringify(packageData, null, 2));
      } catch (error) {
        console.warn('更新package.json失败:', error.message);
      }
    }
  }

  /**
   * 更新文件中版本信息
   */
  updateVersionInFiles(version) {
    const patterns = [
      /version\s*[:=]\s*['"][^'"]+['"]/gi,
      /VERSION\s*[:=]\s*['"][^'"]+['"]/gi
    ];

    // 扫描并更新JS文件
    const files = this.getAllJsFiles();
    
    files.forEach(filePath => {
      try {
        let content = fs.readFileSync(filePath, 'utf8');
        let updated = false;

        patterns.forEach(pattern => {
          if (pattern.test(content)) {
            content = content.replace(pattern, `version: "v1.1.0"`);
            updated = true;
          }
        });

        if (updated) {
          fs.writeFileSync(filePath, content);
          console.log(`已更新: ${path.relative(this.projectRoot, filePath)}`);
        }
      } catch (error) {
        console.warn(`更新文件失败 ${filePath}:`, error.message);
      }
    });
  }

  /**
   * 获取所有JS文件
   */
  getAllJsFiles() {
    const files = [];
    const walkDir = (dir) => {
      const items = fs.readdirSync(dir);
      items.forEach(item => {
        const fullPath = path.join(dir, item);
        const stat = fs.statSync(fullPath);
        
        if (stat.isDirectory() && !item.startsWith('.') && item !== 'node_modules') {
          walkDir(fullPath);
        } else if (stat.isFile() && /\.(js|ts|jsx|tsx)$/.test(item)) {
          files.push(fullPath);
        }
      });
    };

    walkDir(this.projectRoot);
    return files;
  }

  /**
   * 统一API接口
   */
  async executeCommand(command, ...args) {
    switch (command) {
      case 'init':
        return this.init(...args);
      case 'check':
        return this.checkVersion(...args);
      case 'update':
        return this.validateAndUpdate(...args);
      case 'increment':
        return this.autoIncrement(...args);
      case 'sync':
        return this.syncAllFiles();
      case 'status':
        return this.showStatus();
      case 'record':
        return this.recordManager.autoUpdate();
      case 'api':
        return this.api.executeCommand(...args);
      default:
        console.log(`未知命令: ${command}`);
        return false;
    }
  }

  /**
   * 版本检查
   */
  checkVersion(version, changes = []) {
    const report = this.checker.checkVersion(version, changes);
    this.checker.printReport(report);
    return report.valid;
  }

  /**
   * 静态帮助方法
   */
  static showHelp() {
    console.log(`
🔧 统一版本管理系统 v2.1.0
============================

用法: node version-manager.js <命令> [参数] [选项]

核心命令:
  init <version>          初始化版本系统 (默认: 0.1.0)
  check <version>         检查版本格式和规则
  update <version>        验证并更新版本
  increment <level>       自动递增版本 (major/minor/patch)
  sync                    同步所有文件版本信息
  status                  显示当前版本状态
  record                  更新版本记录

版本记录命令:
  record auto             自动更新版本记录
  record info             显示版本记录信息
  record export           导出版本记录模块

API命令:
  api current             获取当前版本信息
  api history             获取版本历史
  api generate            生成版本模块

变更类型规范:
  feat       新功能        → MINOR 版本递增
  fix        修复问题      → PATCH 版本递增
  docs       文档更新      → PATCH 版本递增
  style      代码风格      → PATCH 版本递增
  refactor   代码重构      → PATCH 版本递增
  perf       性能优化      → PATCH 版本递增
  test       测试相关      → PATCH 版本递增
  build      构建系统      → PATCH 版本递增
  ci         CI配置        → PATCH 版本递增
  chore      日常维护      → PATCH 版本递增
  revert     回滚提交      → 根据被回滚的提交决定

选项:
  --author <name>         设置作者名称
  --message <msg>         设置变更描述
  --changes <json>        设置变更列表 (JSON格式)
  --config <file>         指定配置文件路径
  --project-root <path>   指定项目根目录

示例:
  node version-manager.js init v1.0.0
  node version-manager.js check v1.2.0 --changes '[{"type":"feat","description":"新增用户功能"}]'
  node version-manager.js increment minor --message "新增核心功能"
  node version-manager.js update v1.1.0 --author "开发者" --changes '[{"type":"fix","description":"修复bug"}]'
  node version-manager.js record auto
  node version-manager.js api current
    `);
  }
}

// 主程序入口
async function main() {
  const args = process.argv.slice(2);
  
  if (args.length === 0) {
    UniversalVersionManager.showHelp();
    return;
  }

  const command = args[0];
  
  // 全局选项解析
  const globalOptions = {};
  const nonGlobalArgs = [];
  
  for (let i = 1; i < args.length; i++) {
    if (args[i] === '--config' && args[i + 1]) {
      globalOptions.configPath = args[i + 1];
      i++;
    } else if (args[i] === '--project-root' && args[i + 1]) {
      globalOptions.projectRoot = args[i + 1];
      i++;
    } else {
      nonGlobalArgs.push(args[i]);
    }
  }

  // 解析命令特定选项
  const commandOptions = {};
  for (let i = 1; i < nonGlobalArgs.length; i++) {
    if (nonGlobalArgs[i] === '--author' && nonGlobalArgs[i + 1]) {
      commandOptions.author = nonGlobalArgs[i + 1];
      i++;
    } else if (nonGlobalArgs[i] === '--message' && nonGlobalArgs[i + 1]) {
      commandOptions.changes = [{ type: 'chore', description: nonGlobalArgs[i + 1] }];
      i++;
    } else if (nonGlobalArgs[i] === '--changes' && nonGlobalArgs[i + 1]) {
      try {
        commandOptions.changes = JSON.parse(nonGlobalArgs[i + 1]);
      } catch (error) {
        console.error('变更列表解析失败:', error.message);
        process.exit(1);
      }
      i++;
    }
  }

  try {
    const manager = new UniversalVersionManager(globalOptions);

    switch (command) {
      case 'help':
        UniversalVersionManager.showHelp();
        break;
        
      case 'init':
        const initVersion = nonGlobalArgs[0] || '0.1.0';
        manager.init(initVersion, commandOptions);
        break;
        
      case 'check':
        if (nonGlobalArgs.length < 1) {
          console.log('请指定要检查的版本号');
          process.exit(1);
        }
        manager.checkVersion(nonGlobalArgs[0], commandOptions.changes || []);
        break;
        
      case 'update':
        if (nonGlobalArgs.length < 1) {
          console.log('请指定要更新的版本号');
          process.exit(1);
        }
        const success = await manager.validateAndUpdate(nonGlobalArgs[0], commandOptions);
        process.exit(success ? 0 : 1);
        break;
        
      case 'increment':
        const level = nonGlobalArgs[0] || 'patch';
        const incremented = await manager.autoIncrement(level, commandOptions);
        process.exit(incremented ? 0 : 1);
        break;
        
      case 'sync':
        manager.syncAllFiles();
        break;
        
      case 'status':
        manager.showStatus();
        break;
        
      case 'record':
        if (nonGlobalArgs[0] === 'auto') {
          await manager.recordManager.autoUpdate();
        } else if (nonGlobalArgs[0] === 'info') {
          const info = manager.recordManager.getVersionSummary();
          console.log('版本记录信息:');
          console.log(JSON.stringify(info, null, 2));
        } else if (nonGlobalArgs[0] === 'export') {
          const outputPath = nonGlobalArgs[1] || 'version-record-module.js';
          manager.recordManager.exportVersionModule(outputPath);
        } else {
          await manager.recordManager.autoUpdate();
        }
        break;
        
      case 'api':
        await manager.api.executeCommand(...nonGlobalArgs);
        break;
        
      default:
        console.log(`未知命令: ${command}\n`);
        UniversalVersionManager.showHelp();
        process.exit(1);
    }
    
  } catch (error) {
    console.error('执行命令时发生错误:', error.message);
    console.error('详细错误信息:', error.stack);
    process.exit(1);
  }
}

// 直接运行脚本时的入口
if (require.main === module) {
  main().catch(error => {
    console.error('程序执行失败:', error);
    process.exit(1);
  });
}

module.exports = UniversalVersionManager;