// Cypress 支持文件
// ***********************************************************
// 此文件在每个测试文件之前运行
// ***********************************************************

import './commands'

// 导航到应用前清除 localStorage
beforeEach(() => {
  cy.clearLocalStorage()
  cy.clearCookies()
})

// 记录测试失败
afterEach(function () {
  if (this.currentTest.state === 'failed') {
    const testTitle = this.currentTest.title.replace(/\s+/g, '-')
    cy.screenshot(`failed-${testTitle}`)
  }
})

