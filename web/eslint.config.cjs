/**
 * ESLint v10+ 扁平配置（flat config）
 * 使用 @vue/eslint-config-typescript 提供的官方配置。
 */
const { default: createConfig } = require('@vue/eslint-config-typescript');

const baseConfig = createConfig();

module.exports = [
  ...baseConfig,
  {
    ignores: ['dist/**'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-empty-object-type': 'off',
    },
  },
];

