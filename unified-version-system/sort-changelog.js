#!/usr/bin/env node

/**
 * CHANGELOG.md 版本排序工具
 * 将 CHANGELOG.md 中的版本按照从新到旧的顺序重新排列
 */

const fs = require('fs');
const path = require('path');

class ChangelogSorter {
  constructor(changelogPath) {
    this.changelogPath = changelogPath;
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
   * 解析 CHANGELOG.md 文件
   */
  parseChangelog() {
    const content = fs.readFileSync(this.changelogPath, 'utf8');
    const lines = content.split('\n');
    
    // 找到第一个版本标题的位置
    let headerEndIndex = -1;
    for (let i = 0; i < lines.length; i++) {
      if (/^## \[v\d+\.\d+\.\d+\]/.test(lines[i])) {
        headerEndIndex = i;
        break;
      }
    }
    
    if (headerEndIndex === -1) {
      throw new Error('未找到版本标题');
    }
    
    // 提取文件头部（版本之前的内容）
    const header = lines.slice(0, headerEndIndex).join('\n');
    
    // 提取所有版本块
    const versionBlocks = [];
    let currentBlock = null;
    
    for (let i = headerEndIndex; i < lines.length; i++) {
      const line = lines[i];
      const versionMatch = line.match(/^## \[(v\d+\.\d+\.\d+)\]/);
      
      if (versionMatch) {
        // 保存上一个版本块
        if (currentBlock) {
          versionBlocks.push(currentBlock);
        }
        
        // 开始新的版本块
        currentBlock = {
          version: versionMatch[1],
          lines: [line]
        };
      } else if (currentBlock) {
        currentBlock.lines.push(line);
      }
    }
    
    // 保存最后一个版本块
    if (currentBlock) {
      versionBlocks.push(currentBlock);
    }
    
    return { header, versionBlocks };
  }

  /**
   * 排序版本块（从新到旧）
   */
  sortVersionBlocks(versionBlocks) {
    return versionBlocks.sort((a, b) => {
      // 降序排列（从新到旧）
      return this.compareVersions(b.version, a.version);
    });
  }

  /**
   * 重新生成 CHANGELOG.md
   */
  generateChangelog(header, sortedBlocks) {
    const parts = [header];
    
    sortedBlocks.forEach(block => {
      parts.push(block.lines.join('\n'));
    });
    
    return parts.join('\n');
  }

  /**
   * 执行排序
   */
  sort() {
    console.log('开始排序 CHANGELOG.md...');
    
    // 备份原文件
    const backupPath = this.changelogPath + '.backup';
    fs.copyFileSync(this.changelogPath, backupPath);
    console.log(`已备份原文件到: ${backupPath}`);
    
    // 解析文件
    const { header, versionBlocks } = this.parseChangelog();
    console.log(`找到 ${versionBlocks.length} 个版本块`);
    
    // 排序
    const sortedBlocks = this.sortVersionBlocks(versionBlocks);
    
    // 显示排序结果
    console.log('\n排序后的版本顺序（从新到旧）:');
    sortedBlocks.forEach((block, index) => {
      console.log(`  ${index + 1}. ${block.version}`);
    });
    
    // 生成新内容
    const newContent = this.generateChangelog(header, sortedBlocks);
    
    // 写入文件
    fs.writeFileSync(this.changelogPath, newContent, 'utf8');
    console.log(`\n✅ CHANGELOG.md 已重新排序`);
    console.log(`备份文件: ${backupPath}`);
  }
}

// 主程序
function main() {
  const changelogPath = path.join(process.cwd(), 'CHANGELOG.md');
  
  if (!fs.existsSync(changelogPath)) {
    console.error('❌ 未找到 CHANGELOG.md 文件');
    process.exit(1);
  }
  
  try {
    const sorter = new ChangelogSorter(changelogPath);
    sorter.sort();
  } catch (error) {
    console.error('❌ 排序失败:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = ChangelogSorter;
