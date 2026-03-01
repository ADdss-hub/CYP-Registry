/**
 * 基于 Vue 3 + TypeScript 的基础 ESLint 配置，
 * 使用项目中已经安装的 eslint、eslint-plugin-vue 和 @vue/eslint-config-typescript。
 */
module.exports = {
  root: true,
  env: {
    browser: true,
    es2021: true,
    node: true,
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-recommended',
    '@vue/eslint-config-typescript',
  ],
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module',
  },
  // 不再关闭任何规则，完全交给 eslint:recommended + plugin:vue/vue3-recommended +
  // @vue/eslint-config-typescript 管理；若有需要，只做「风格选择」类调整，而不是关闭。
  rules: {},
};

