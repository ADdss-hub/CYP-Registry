#!/usr/bin/env node

/**
 * 版本配置管理器 (Version Config Manager)
 * 负责项目自适应配置管理和版本规则配置
 * 
 * @author Universal Version Manager
 * @version v2.1.0
 */

const fs = require('fs');
const path = require('path');

/**
 * 版本配置管理器类
 * 提供项目自适应配置和版本规则管理
 */
class VersionConfig {
  constructor(projectRoot = process.cwd()) {
    this.projectRoot = path.resolve(projectRoot);
    this.configFile = path.join(this.projectRoot, '.version-config.json');
    this.config = this.loadConfig();
  }

  /**
   * 加载配置文件
   */
  loadConfig() {
    if (fs.existsSync(this.configFile)) {
      try {
        const config = JSON.parse(fs.readFileSync(this.configFile, 'utf8'));
        console.log(`✅ 配置文件已加载: ${this.configFile}`);
        return config;
      } catch (error) {
        console.warn('配置文件加载失败，使用默认配置:', error.message);
        return this.getDefaultConfig();
      }
    }
    
    console.log('未找到配置文件，将创建默认配置');
    return this.getDefaultConfig();
  }

  /**
   * 获取默认配置
   */
  getDefaultConfig() {
    const projectType = this.detectProjectType();
    
    return {
      schema: '2.1.0',
      project: {
        name: this.getProjectName(),
        type: projectType,
        root: this.projectRoot
      },
      versioning: {
        format: 'semver', // semver, date-based, etc.
        prefix: 'v',      // 版本前缀
        rules: {
          enforcePrefix: true,
          validateIncrement: true,
          checkDuplicate: true,
          maxHistoryEntries: 100
        }
      },
      changes: {
        types: {
          feat: { level: 'minor', description: '新功能' },
          fix: { level: 'patch', description: '修复问题' },
          docs: { level: 'patch', description: '文档更新' },
          style: { level: 'patch', description: '代码风格' },
          refactor: { level: 'patch', description: '代码重构' },
          perf: { level: 'patch', description: '性能优化' },
          test: { level: 'patch', description: '测试相关' },
          build: { level: 'patch', description: '构建系统' },
          ci: { level: 'patch', description: 'CI配置' },
          chore: { level: 'patch', description: '日常维护' },
          revert: { level: 'auto', description: '回滚提交' }
        }
      },
      scanning: {
        patterns: [
          '**/*.js',
          '**/*.ts', 
          '**/*.jsx',
          '**/*.tsx'
        ],
        ignorePatterns: [
          'node_modules/**',
          'dist/**',
          'build/**',
          '.git/**',
          '**/*.min.js'
        ],
        versionPatterns: [
          /version\s*[:=]\s*['"]([^'"]+)['"]/gi,
          /VERSION\s*[:=]\s*['"]([^'"]+)['"]/gi,
          /@version\s+([^\s]+)/gi,
          /const\s+version\s*=\s*['"]([^'"]+)['"]/gi,
          /export\s+const\s+version\s*=\s*['"]([^'"]+)['"]/gi
        ]
      },
      automation: {
        autoIncrement: true,
        autoRecord: true,
        autoSync: true,
        autoGenerate: true
      }
    };
  }

  /**
   * 检测项目类型
   */
  detectProjectType() {
    const indicators = {
      'node': ['package.json', 'node_modules'],
      'npm': ['package.json'],
      'git': ['.git'],
      'docker': ['Dockerfile', 'docker-compose.yml'],
      'vue': ['vue.config.js', 'package.json'],
      'react': ['package.json', 'src/App.js'],
      'typescript': ['tsconfig.json'],
      'jest': ['jest.config.js', 'package.json']
    };

    const detectedTypes = [];
    
    for (const [type, files] of Object.entries(indicators)) {
      const hasFiles = files.some(file => 
        fs.existsSync(path.join(this.projectRoot, file))
      );
      if (hasFiles) {
        detectedTypes.push(type);
      }
    }

    return detectedTypes.length > 0 ? detectedTypes : ['generic'];
  }

  /**
   * 获取项目名称
   */
  getProjectName() {
    // 尝试从 package.json 获取
    const packagePath = path.join(this.projectRoot, 'package.json');
    if (fs.existsSync(packagePath)) {
      try {
        const packageData = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
        if (packageData.name) {
          return packageData.name;
        }
      } catch (error) {
        console.warn('读取 package.json 失败:', error.message);
      }
    }

    // 使用目录名
    return path.basename(this.projectRoot);
  }

  /**
   * 初始化配置
   */
  initConfig(options = {}) {
    console.log('🔧 初始化版本配置...');

    // 应用选项覆盖
    const config = { ...this.config };
    
    if (options.projectName) {
      config.project.name = options.projectName;
    }

    if (options.versioning) {
      config.versioning = { ...config.versioning, ...options.versioning };
    }

    if (options.changes) {
      config.changes.types = { ...config.changes.types, ...options.changes.types };
    }

    // 保存配置
    this.saveConfig(config);
    
    console.log('✅ 版本配置初始化完成');
    return config;
  }

  /**
   * 保存配置
   */
  saveConfig(config = null) {
    const configToSave = config || this.config;
    
    try {
      fs.writeFileSync(this.configFile, JSON.stringify(configToSave, null, 2));
      this.config = configToSave;
      console.log(`💾 配置已保存: ${this.configFile}`);
      return true;
    } catch (error) {
      console.error('保存配置失败:', error.message);
      return false;
    }
  }

  /**
   * 获取配置值
   */
  get(key, defaultValue = null) {
    const keys = key.split('.');
    let value = this.config;
    
    for (const k of keys) {
      if (value && typeof value === 'object' && k in value) {
        value = value[k];
      } else {
        return defaultValue;
      }
    }
    
    return value;
  }

  /**
   * 设置配置值
   */
  set(key, value) {
    const keys = key.split('.');
    let obj = this.config;
    
    for (let i = 0; i < keys.length - 1; i++) {
      const k = keys[i];
      if (!obj[k] || typeof obj[k] !== 'object') {
        obj[k] = {};
      }
      obj = obj[k];
    }
    
    obj[keys[keys.length - 1]] = value;
    return this.saveConfig();
  }

  /**
   * 获取完整配置
   */
  getConfig() {
    return { ...this.config };
  }

  /**
   * 验证配置
   */
  validateConfig(config = null) {
    const configToValidate = config || this.config;
    const errors = [];
    const warnings = [];

    // 验证必需字段
    if (!configToValidate.project || !configToValidate.project.name) {
      errors.push('缺少项目名称配置');
    }

    if (!configToValidate.versioning || !configToValidate.versioning.format) {
      errors.push('缺少版本格式化配置');
    }

    // 验证版本规则
    if (configToValidate.versioning && configToValidate.versioning.rules) {
      const rules = configToValidate.versioning.rules;
      
      if (rules.maxHistoryEntries && (rules.maxHistoryEntries < 1 || rules.maxHistoryEntries > 10000)) {
        warnings.push('历史记录数量限制应在 1-10000 之间');
      }
    }

    // 验证变更类型配置
    if (configToValidate.changes && configToValidate.changes.types) {
      const changeTypes = configToValidate.changes.types;
      const validLevels = ['major', 'minor', 'patch', 'auto'];
      
      for (const [type, config] of Object.entries(changeTypes)) {
        if (!validLevels.includes(config.level)) {
          warnings.push(`变更类型 ${type} 的级别配置无效: ${config.level}`);
        }
      }
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
      validatedConfig: configToValidate
    };
  }

  /**
   * 创建配置文件
   */
  createConfigFile(config = null) {
    const configToCreate = config || this.getDefaultConfig();
    
    try {
      fs.writeFileSync(this.configFile, JSON.stringify(configToCreate, null, 2));
      this.config = configToCreate;
      console.log(`📄 配置文件已创建: ${this.configFile}`);
      return true;
    } catch (error) {
      console.error('创建配置文件失败:', error.message);
      return false;
    }
  }

  /**
   * 重置为默认配置
   */
  resetToDefault() {
    const defaultConfig = this.getDefaultConfig();
    return this.saveConfig(defaultConfig);
  }

  /**
   * 导入配置
   */
  importConfig(configPath) {
    if (!fs.existsSync(configPath)) {
      throw new Error(`配置文件不存在: ${configPath}`);
    }

    try {
      const configData = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      const validation = this.validateConfig(configData);
      
      if (!validation.valid) {
        throw new Error(`配置文件验证失败: ${validation.errors.join(', ')}`);
      }

      this.config = validation.validatedConfig;
      this.saveConfig();
      
      console.log(`✅ 配置已从 ${configPath} 导入`);
      return true;
    } catch (error) {
      console.error('导入配置失败:', error.message);
      return false;
    }
  }

  /**
   * 导出现有配置
   */
  exportConfig(outputPath) {
    try {
      fs.writeFileSync(outputPath, JSON.stringify(this.config, null, 2));
      console.log(`📤 配置已导出到: ${outputPath}`);
      return true;
    } catch (error) {
      console.error('导出配置失败:', error.message);
      return false;
    }
  }

  /**
   * 获取项目类型特定配置
   */
  getProjectSpecificConfig() {
    const projectTypes = this.config.project.type;
    const specificConfig = {};

    if (Array.isArray(projectTypes)) {
      projectTypes.forEach(type => {
        switch (type) {
          case 'node':
          case 'npm':
            specificConfig.npm = {
              syncPackageJson: true,
              updateScripts: true
            };
            break;
          case 'typescript':
            specificConfig.typescript = {
              checkTsConfig: true,
              updateDeclarationFiles: false
            };
            break;
          case 'jest':
            specificConfig.jest = {
              updateTestVersion: true
            };
            break;
          case 'vue':
            specificConfig.vue = {
              syncVueConfig: true,
              updateComponentVersions: true
            };
            break;
          case 'react':
            specificConfig.react = {
              syncReactConfig: true,
              updateComponentVersions: true
            };
            break;
        }
      });
    }

    return specificConfig;
  }

  /**
   * 显示配置信息
   */
  showConfig() {
    console.log('\n🔧 版本配置信息:');
    console.log('=' .repeat(40));
    console.log(`项目名称: ${this.config.project.name}`);
    console.log(`项目类型: ${this.config.project.type.join(', ')}`);
    console.log(`版本格式: ${this.config.versioning.format}`);
    console.log(`版本前缀: ${this.config.versioning.prefix}`);
    console.log(`配置文件: ${this.configFile}`);

    console.log('\n📋 版本规则:');
    Object.entries(this.config.versioning.rules).forEach(([key, value]) => {
      console.log(`  ${key}: ${value}`);
    });

    console.log('\n🔄 变更类型:');
    Object.entries(this.config.changes.types).forEach(([type, config]) => {
      console.log(`  ${type}: ${config.level} - ${config.description}`);
    });

    console.log('\n🤖 自动化:');
    Object.entries(this.config.automation).forEach(([key, value]) => {
      console.log(`  ${key}: ${value}`);
    });
  }

  /**
   * 更新项目配置
   */
  updateProjectConfig(updates) {
    const currentConfig = this.getConfig();
    const updatedConfig = {
      ...currentConfig,
      ...updates,
      schema: currentConfig.schema // 保持schema版本
    };

    const validation = this.validateConfig(updatedConfig);
    if (!validation.valid) {
      throw new Error(`配置更新失败: ${validation.errors.join(', ')}`);
    }

    return this.saveConfig(updatedConfig);
  }

  /**
   * 合并外部配置
   */
  mergeExternalConfig(externalConfig, overwrite = false) {
    const currentConfig = this.getConfig();
    let mergedConfig;

    if (overwrite) {
      mergedConfig = { ...externalConfig, schema: currentConfig.schema };
    } else {
      mergedConfig = { ...currentConfig, ...externalConfig };
      // 深层合并
      mergedConfig.project = { ...currentConfig.project, ...externalConfig.project };
      mergedConfig.versioning = { ...currentConfig.versioning, ...externalConfig.versioning };
      mergedConfig.changes = { 
        ...currentConfig.changes, 
        types: { ...currentConfig.changes.types, ...externalConfig.changes?.types }
      };
    }

    const validation = this.validateConfig(mergedConfig);
    if (!validation.valid) {
      throw new Error(`配置合并失败: ${validation.errors.join(', ')}`);
    }

    return this.saveConfig(mergedConfig);
  }

  /**
   * 获取配置摘要
   */
  getConfigSummary() {
    return {
      projectName: this.config.project.name,
      projectTypes: this.config.project.type,
      versioningFormat: this.config.versioning.format,
      versionPrefix: this.config.versioning.prefix,
      totalChangeTypes: Object.keys(this.config.changes.types).length,
      automationEnabled: Object.values(this.config.automation).filter(Boolean).length,
      configFile: this.configFile,
      lastUpdated: fs.existsSync(this.configFile) ? fs.statSync(this.configFile).mtime : null
    };
  }
}

module.exports = VersionConfig;