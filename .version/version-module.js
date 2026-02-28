/**
 * 自动生成的版本模块
 * 版本: v1.0.0
 * 生成时间: 2026-02-27T16:34:55.017Z
 */

const version = 'v1.0.0';
const versionData = {
  "version": "v1.0.0",
  "timestamp": "2026-02-27T16:34:55.016Z",
  "author": "Unknown",
  "project": "CYP-Registry",
  "changes": [
    {
      "type": "chore",
      "description": "发布 1.0.0 版本"
    }
  ],
  "metadata": {
    "previousVersion": "v0.1.0",
    "updated": true,
    "schema": "2.1.0"
  }
};

module.exports = {
  version,
  versionData,
  
  // 版本信息
  getVersion() {
    return version;
  },
  
  getVersionData() {
    return versionData;
  },
  
  // 版本比较
  compare(otherVersion) {
    const VersionChecker = require('./version-checker.js');
    const checker = new VersionChecker();
    
    const current = checker.parseVersion(version);
    const other = checker.parseVersion(otherVersion);
    
    if (!current || !other) return null;
    
    if (current.major !== other.major) {
      return current.major > other.major ? 1 : -1;
    }
    if (current.minor !== other.minor) {
      return current.minor > other.minor ? 1 : -1;
    }
    if (current.patch !== other.patch) {
      return current.patch > other.patch ? 1 : -1;
    }
    
    return 0;
  },
  
  // 版本格式化
  format(format = 'full') {
    switch (format) {
      case 'short':
        return version;
      case 'major':
        return `v${versionData.version.major}`;
      case 'majorMinor':
        return `v${versionData.version.major}.${versionData.version.minor}`;
      case 'full':
      default:
        return `${versionData.project} v${version} (${versionData.timestamp})`;
    }
  }
};

console.log('版本模块已加载:', module.exports.format('full'));
