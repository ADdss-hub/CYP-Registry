/**
 * 统一版本模块 (Universal Version Module)
 * 自动生成的版本信息汇总
 * 生成时间: 2026-03-01T12:32:43.850Z
 */

const versionInfo = {
  // 当前版本信息
  current: {
  "version": "v1.1.0",
  "timestamp": "2026-03-01T12:32:43.844Z",
  "author": "CYP",
  "project": "CYP-Registry",
  "changes": [
    {
      "type": "chore",
      "description": "发布 v1.1.0 版本"
    }
  ],
  "metadata": {
    "previousVersion": "v1.0.3",
    "updated": true,
    "schema": "2.1.0"
  }
},
  
  // 版本历史
  history: [
  {
    "version": "v1.1.0",
    "timestamp": "2026-03-01T12:32:43.844Z",
    "author": "CYP",
    "changes": [
      {
        "type": "chore",
        "description": "发布 v1.1.0 版本"
      }
    ],
    "type": "patch",
    "previousVersion": "v1.0.3",
    "metadata": {
      "previousVersion": "v1.0.3",
      "updated": true,
      "schema": "2.1.0"
    }
  },
  {
    "version": "v1.0.0",
    "timestamp": "2026-02-27T16:34:55.016Z",
    "author": "Unknown",
    "changes": [
      {
        "type": "chore",
        "description": "发布 1.0.0 版本"
      }
    ],
    "type": "patch",
    "previousVersion": "v0.1.0",
    "metadata": {
      "previousVersion": "v0.1.0",
      "updated": true,
      "schema": "2.1.0"
    }
  }
],
  
  // 版本记录统计
  records: {
  "lastUpdate": "2026-02-27T16:35:02.923Z",
  "files": {
    "docs\\api\\API.md": "v1.0.3",
    "src\\pkg\\version\\version.go": "v0.0.0-dev",
    "规范文件\\全平台Git与GitHub工作流管理规范.md": "v1.0.0",
    "规范文件\\全平台通用容器开发设计规范.md": "v1.0.0",
    "规范文件\\全平台通用开发任务设计规范.md": "v1.0.0",
    "规范文件\\全平台通用数据库个人管理规范.md": "v1.0.0",
    "规范文件\\全平台通用用户认证设计规范.md": "v1.0.0",
    "规范文件\\项目库、依赖及服务使用管理规范.md": "v1.0.0",
    "规范文件\\项目级规范-全局配置中心（.env）与派生规范.md": "v1.0.0",
    "项目设计文档\\系统开发设计文档.md": "v1.0.2",
    ".version\\version-module.js": "v1.0.0",
    ".version\\version.json": "v1.0.0",
    ".version\\changelog.json": "v1.0.0",
    "web\\src\\constants\\legal.ts": "v1.0.0"
  }
},
  
  // 版本服务API
  api: {
    getCurrentVersion() {
      return "v1.1.0";
    },
    
    getProjectName() {
      return "CYP-Registry";
    },
    
    getVersionTimestamp() {
      return "2026-03-01T12:32:43.844Z";
    },
    
    getVersionAuthor() {
      return "CYP";
    },
    
    getVersionChanges() {
      return [
  {
    "type": "chore",
    "description": "发布 v1.1.0 版本"
  }
];
    },
    
    getVersionHistory() {
      return [
  {
    "version": "v1.1.0",
    "timestamp": "2026-03-01T12:32:43.844Z",
    "author": "CYP",
    "changes": [
      {
        "type": "chore",
        "description": "发布 v1.1.0 版本"
      }
    ],
    "type": "patch",
    "previousVersion": "v1.0.3",
    "metadata": {
      "previousVersion": "v1.0.3",
      "updated": true,
      "schema": "2.1.0"
    }
  },
  {
    "version": "v1.0.0",
    "timestamp": "2026-02-27T16:34:55.016Z",
    "author": "Unknown",
    "changes": [
      {
        "type": "chore",
        "description": "发布 1.0.0 版本"
      }
    ],
    "type": "patch",
    "previousVersion": "v0.1.0",
    "metadata": {
      "previousVersion": "v0.1.0",
      "updated": true,
      "schema": "2.1.0"
    }
  }
];
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
