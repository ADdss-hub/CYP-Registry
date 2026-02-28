import { defineConfig } from 'cypress'

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:4173',
    viewportWidth: 1280,
    viewportHeight: 720,
    video: true,
    videoCompression: 32,
    screenshotOnRunFailure: true,
    // CI 环境中清理旧资源，本地开发时保留以便调试
    trashAssetsBeforeRuns: process.env.CI === 'true',
    defaultCommandTimeout: 10000,
    requestTimeout: 10000,
    responseTimeout: 30000,
    pageLoadTimeout: 30000,
    execTimeout: 60000,
    taskTimeout: 60000,
    retries: {
      runMode: 1,
      openMode: 0,
    },
    // CI 环境中不保留测试在内存中（节省内存），本地开发时保留（提升性能）
    numTestsKeptInMemory: process.env.CI === 'true' ? 0 : 50,
    supportFile: 'cypress/support/e2e.ts',
    specPattern: 'cypress/e2e/**/*.cy.{ts,js}',
    fixturesFolder: 'cypress/fixtures',
    videosFolder: 'cypress/videos',
    screenshotsFolder: 'cypress/screenshots',
    downloadsFolder: 'cypress/downloads',
    env: {
      apiUrl: process.env.CYPRESS_apiUrl || 'http://localhost:8080/api/v1',
      TEST_USERNAME: process.env.CYPRESS_TEST_USERNAME || 'testuser',
      TEST_PASSWORD: process.env.CYPRESS_TEST_PASSWORD || 'TestPassword123',
    },
    setupNodeEvents(on, config) {
      // 可以在这里添加自定义任务
      on('task', {
        log(message) {
          console.log(message)
          return null
        },
      })
    },
  },
  component: {
    devServer: {
      framework: 'vue',
      bundler: 'vite',
    },
  },
})

