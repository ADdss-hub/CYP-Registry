#!/usr/bin/env node

/**
 * 版本API (Version API)
 * 统一版本服务API，提供版本信息的查询、管理和同步功能
 * 
 * @author Universal Version Manager
 * @version v2.1.0
 */

const fs = require('fs');
const path = require('path');

/**
 * 版本API类
 * 提供统一的版本服务接口
 */
class VersionAPI {
  constructor(options = {}) {
    this.options = {
      projectRoot: options.projectRoot || process.cwd(),
      ...options
    };

    this.versionDir = path.join(this.options.projectRoot, '.version');
    this.versionFile = path.join(this.versionDir, 'version.json');
    this.historyFile = path.join(this.versionDir, 'changelog.json');
    this.recordFile = path.join(this.options.projectRoot, '.version-record.json');
  }

  /**
   * 获取当前版本信息
   */
  getCurrentVersion() {
    if (fs.existsSync(this.versionFile)) {
      try {
        return JSON.parse(fs.readFileSync(this.versionFile, 'utf8'));
      } catch (error) {
        console.warn('读取当前版本失败:', error.message);
        return null;
      }
    }
    return null;
  }

  /**
   * 获取版本历史
   */
  getVersionHistory(limit = null) {
    if (fs.existsSync(this.historyFile)) {
      try {
        const historyData = JSON.parse(fs.readFileSync(this.historyFile, 'utf8'));
        const history = historyData.history || [];
        
        if (limit && typeof limit === 'number') {
          return history.slice(0, limit);
        }
        
        return history;
      } catch (error) {
        console.warn('读取版本历史失败:', error.message);
        return [];
      }
    }
    return [];
  }

  /**
   * 获取版本记录信息
   */
  getVersionRecords() {
    if (fs.existsSync(this.recordFile)) {
      try {
        return JSON.parse(fs.readFileSync(this.recordFile, 'utf8'));
      } catch (error) {
        console.warn('读取版本记录失败:', error.message);
        return null;
      }
    }
    return null;
  }

  /**
   * 生成版本模块
   */
  generateVersionModule() {
    const currentVersion = this.getCurrentVersion();
    const history = this.getVersionHistory(10); // 最近10条历史
    const records = this.getVersionRecords();

    const moduleContent = `/**
 * 统一版本模块 (Universal Version Module)
 * 自动生成的版本信息汇总
 * 生成时间: ${new Date().toISOString()}
 */

const versionInfo = {
  // 当前版本信息
  current: ${currentVersion ? JSON.stringify(currentVersion, null, 2) : 'null'},
  
  // 版本历史
  history: ${JSON.stringify(history, null, 2)},
  
  // 版本记录统计
  records: ${records ? JSON.stringify(records, null, 2) : 'null'},
  
  // 版本服务API
  api: {
    getCurrentVersion() {
      return ${currentVersion ? `"${currentVersion.version}"` : 'null'};
    },
    
    getProjectName() {
      return ${currentVersion ? `"${currentVersion.project}"` : 'null'};
    },
    
    getVersionTimestamp() {
      return ${currentVersion ? `"${currentVersion.timestamp}"` : 'null'};
    },
    
    getVersionAuthor() {
      return ${currentVersion ? `"${currentVersion.author}"` : 'null'};
    },
    
    getVersionChanges() {
      return ${currentVersion ? JSON.stringify(currentVersion.changes, null, 2) : '[]'};
    },
    
    getVersionHistory() {
      return ${JSON.stringify(history, null, 2)};
    },
    
    getVersionSummary() {
      const current = this.getCurrentVersion();
      const history = this.getVersionHistory();
      
      return {
        current,
        totalHistory: history.length,
        latestUpdate: history[0]?.timestamp || null,
        project: this.getProjectName()
      };
    }
  }
};

module.exports = versionInfo;

// 快速访问函数
module.exports.getVersion = () => module.exports.api.getCurrentVersion();
module.exports.getProject = () => module.exports.api.getProjectName();
module.exports.getHistory = () => module.exports.api.getVersionHistory();
module.exports.getSummary = () => module.exports.api.getVersionSummary();

console.log('统一版本模块已加载:', module.exports.getSummary());
`;

    const modulePath = path.join(this.options.projectRoot, 'universal-version-module.js');
    fs.writeFileSync(modulePath, moduleContent);
    console.log(`📦 统一版本模块已生成: ${modulePath}`);
    
    return modulePath;
  }

  /**
   * 同步版本信息到指定文件
   */
  syncVersionToFiles(version) {
    const jsFiles = this.getAllJsFiles();
    const patterns = [
      /version\s*[:=]\s*['"][^'"]+['"]/gi,
      /VERSION\s*[:=]\s*['"][^'"]+['"]/gi,
      /@version\s+[^\s]+/gi
    ];

    let updatedCount = 0;
    
    jsFiles.forEach(filePath => {
      try {
        let content = fs.readFileSync(filePath, 'utf8');
        let updated = false;

        patterns.forEach(pattern => {
          if (pattern.test(content)) {
            content = content.replace(pattern, `version: "${version}"`);
            updated = true;
          }
        });

        if (updated) {
          fs.writeFileSync(filePath, content);
          console.log(`已同步: ${path.relative(this.options.projectRoot, filePath)}`);
          updatedCount++;
        }
      } catch (error) {
        console.warn(`同步文件失败 ${filePath}:`, error.message);
      }
    });

    return updatedCount;
  }

  /**
   * 获取所有JS文件
   */
  getAllJsFiles() {
    const files = [];
    const walkDir = (dir) => {
      if (!fs.existsSync(dir)) return;
      
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

    walkDir(this.options.projectRoot);
    return files;
  }

  /**
   * 创建版本信息JSON
   */
  createVersionJson() {
    const currentVersion = this.getCurrentVersion();
    const history = this.getVersionHistory();
    const records = this.getVersionRecords();

    const versionJson = {
      schema: '2.1.0',
      generated: new Date().toISOString(),
      project: currentVersion?.project || 'unknown',
      version: currentVersion?.version || null,
      timestamp: currentVersion?.timestamp || null,
      author: currentVersion?.author || null,
      changes: currentVersion?.changes || [],
      history: {
        total: history.length,
        recent: history.slice(0, 5)
      },
      records: records ? {
        totalFiles: records.totalFiles || 0,
        uniqueVersions: records.uniqueVersions || 0,
        versions: records.versions || []
      } : null
    };

    const jsonPath = path.join(this.options.projectRoot, 'version.json');
    fs.writeFileSync(jsonPath, JSON.stringify(versionJson, null, 2));
    console.log(`📄 版本信息JSON已创建: ${jsonPath}`);
    
    return jsonPath;
  }

  /**
   * 验证版本一致性
   */
  validateVersionConsistency() {
    const currentVersion = this.getCurrentVersion();
    const records = this.getVersionRecords();
    const packageJson = this.getPackageJsonVersion();

    const inconsistencies = [];

    // 检查 package.json 版本一致性
    if (packageJson && currentVersion) {
      if (packageJson !== currentVersion.version) {
        inconsistencies.push({
          type: 'package.json',
          expected: currentVersion.version,
          actual: packageJson,
          message: `package.json 版本与当前版本不一致`
        });
      }
    }

    // 检查版本记录中的最新版本
    if (records && currentVersion && records.versions && records.versions.length > 0) {
      const latestInRecords = records.versions[0];
      if (latestInRecords !== currentVersion.version) {
        inconsistencies.push({
          type: 'version-records',
          expected: currentVersion.version,
          actual: latestInRecords,
          message: `版本记录中的最新版本与当前版本不一致`
        });
      }
    }

    return {
      consistent: inconsistencies.length === 0,
      inconsistencies
    };
  }

  /**
   * 获取 package.json 版本
   */
  getPackageJsonVersion() {
    const packagePath = path.join(this.options.projectRoot, 'package.json');
    if (fs.existsSync(packagePath)) {
      try {
        const packageData = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
        return packageData.version || null;
      } catch (error) {
        console.warn('读取 package.json 失败:', error.message);
        return null;
      }
    }
    return null;
  }

  /**
   * 更新 package.json 版本
   */
  updatePackageJsonVersion(version) {
    const packagePath = path.join(this.options.projectRoot, 'package.json');
    if (fs.existsSync(packagePath)) {
      try {
        const packageData = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
        packageData.version = version;
        fs.writeFileSync(packagePath, JSON.stringify(packageData, null, 2));
        console.log(`✅ package.json 版本已更新: ${version}`);
        return true;
      } catch (error) {
        console.warn('更新 package.json 失败:', error.message);
        return false;
      }
    }
    return false;
  }

  /**
   * 执行API命令
   */
  async executeCommand(command, ...args) {
    switch (command) {
      case 'current':
        const current = this.getCurrentVersion();
        console.log('当前版本信息:');
        console.log(JSON.stringify(current, null, 2));
        return current;
        
      case 'history':
        const history = this.getVersionHistory();
        console.log('版本历史:');
        history.forEach((entry, index) => {
          console.log(`${index + 1}. ${entry.version} - ${entry.timestamp}`);
        });
        return history;
        
      case 'generate':
        const modulePath = this.generateVersionModule();
        console.log(`版本模块已生成: ${modulePath}`);
        return modulePath;
        
      case 'json':
        const jsonPath = this.createVersionJson();
        console.log(`版本JSON已创建: ${jsonPath}`);
        return jsonPath;
        
      case 'sync':
        const currentVersion = this.getCurrentVersion();
        if (currentVersion) {
          const count = this.syncVersionToFiles(currentVersion.version);
          console.log(`已同步 ${count} 个文件的版本信息`);
          return count;
        } else {
          console.log('未找到当前版本');
          return false;
        }
        
      case 'validate':
        const validation = this.validateVersionConsistency();
        if (validation.consistent) {
          console.log('✅ 版本信息一致');
        } else {
          console.log('❌ 发现版本不一致:');
          validation.inconsistencies.forEach(issue => {
            console.log(`  - ${issue.message} (期望: ${issue.expected}, 实际: ${issue.actual})`);
          });
        }
        return validation;
        
      case 'records':
        const records = this.getVersionRecords();
        console.log('版本记录信息:');
        console.log(JSON.stringify(records, null, 2));
        return records;
        
      default:
        console.log(`未知API命令: ${command}`);
        return false;
    }
  }

  /**
   * 获取版本统计信息
   */
  getVersionStatistics() {
    const currentVersion = this.getCurrentVersion();
    const history = this.getVersionHistory();
    const records = this.getVersionRecords();

    // 统计变更类型
    const changeTypeStats = {};
    history.forEach(entry => {
      if (entry.changes) {
        entry.changes.forEach(change => {
          changeTypeStats[change.type] = (changeTypeStats[change.type] || 0) + 1;
        });
      }
    });

    // 统计作者贡献
    const authorStats = {};
    history.forEach(entry => {
      authorStats[entry.author] = (authorStats[entry.author] || 0) + 1;
    });

    return {
      current: currentVersion?.version || null,
      totalVersions: history.length,
      changeTypes: changeTypeStats,
      authors: authorStats,
      records: records ? {
        totalFiles: records.totalFiles || 0,
        uniqueVersions: records.uniqueVersions || 0
      } : null,
      firstVersion: history.length > 0 ? history[history.length - 1].version : null,
      lastUpdate: history.length > 0 ? history[0].timestamp : null
    };
  }

  /**
   * 生成版本报告
   */
  generateVersionReport() {
    const stats = this.getVersionStatistics();
    const validation = this.validateVersionConsistency();

    console.log('\n📊 版本管理报告:');
    console.log('=' .repeat(50));
    console.log(`当前版本: ${stats.current || '未设置'}`);
    console.log(`总版本数: ${stats.totalVersions}`);
    console.log(`版本文件: ${stats.records?.totalFiles || 0} 个`);
    console.log(`唯一版本: ${stats.records?.uniqueVersions || 0} 个`);

    if (stats.totalVersions > 0) {
      console.log(`\n📜 变更类型统计:`);
      Object.entries(stats.changeTypes).forEach(([type, count]) => {
        console.log(`  ${type}: ${count} 次`);
      });

      console.log(`\n👥 作者贡献统计:`);
      Object.entries(stats.authors).forEach(([author, count]) => {
        console.log(`  ${author}: ${count} 次更新`);
      });
    }

    console.log(`\n🔍 版本一致性: ${validation.consistent ? '✅ 一致' : '❌ 不一致'}`);
    if (!validation.consistent) {
      validation.inconsistencies.forEach(issue => {
        console.log(`  - ${issue.message}`);
      });
    }

    return { stats, validation };
  }
}

module.exports = VersionAPI;