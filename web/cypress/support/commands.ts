/* eslint-disable @typescript-eslint/no-namespace */
// ***********************************************************
// 自定义命令
// ***********************************************************

/// <reference types="cypress" />

// 登录命令
Cypress.Commands.add('login', (username: string, password: string) => {
  cy.session([username, password], () => {
    cy.visit('/login')
    cy.get('[data-testid="username-input"]').clear().type(username)
    cy.get('[data-testid="password-input"]').clear().type(password)
    cy.get('[data-testid="login-button"]').click()
    // 等待登录成功
    cy.url().should('not.include', '/login')
    cy.localStorage('token').should('exist')
  })
})

// 注册命令
Cypress.Commands.add(
  'register',
  (username: string, email: string, password: string, nickname?: string) => {
    cy.visit('/register')
    cy.get('[data-testid="username-input"]').clear().type(username)
    cy.get('[data-testid="email-input"]').clear().type(email)
    cy.get('[data-testid="password-input"]').clear().type(password)
    cy.get('[data-testid="confirm-password-input"]').clear().type(password)
    if (nickname) {
      cy.get('[data-testid="nickname-input"]').clear().type(nickname)
    }
    cy.get('[data-testid="register-button"]').click()
    // 等待注册成功并重定向到登录页
    cy.url().should('include', '/login')
  }
)

// API 登录命令
Cypress.Commands.add('apiLogin', (username: string, password: string) => {
  cy.request({
    method: 'POST',
    url: `${Cypress.env('apiUrl')}/auth/login`,
    body: {
      username,
      password,
    },
  }).then((response) => {
    expect(response.status).to.eq(200)
    expect(response.body.code).to.eq(20000)
    expect(response.body.data).to.have.property('accessToken')
    // 保存 token 到 localStorage
    cy.window().then((win) => {
      win.localStorage.setItem('token', response.body.data.accessToken)
      win.localStorage.setItem('refreshToken', response.body.data.refreshToken)
      win.localStorage.setItem('user', JSON.stringify(response.body.data.user))
    })
  })
})

// 登出命令
Cypress.Commands.add('logout', () => {
  cy.window().then((win) => {
    win.localStorage.removeItem('token')
    win.localStorage.removeItem('refreshToken')
    win.localStorage.removeItem('user')
  })
  cy.visit('/login')
})

// 创建项目命令
Cypress.Commands.add(
  'createProject',
  (name: string, description?: string, isPublic = false) => {
    cy.visit('/projects')
    cy.get('[data-testid="create-project-button"]').click()
    cy.get('[data-testid="project-name-input"]').clear().type(name)
    if (description) {
      cy.get('[data-testid="project-description-input"]').clear().type(description)
    }
    if (isPublic) {
      cy.get('[data-testid="public-checkbox"]').check()
    }
    cy.get('[data-testid="submit-button"]').click()
    // 验证项目创建成功
    cy.contains(name).should('exist')
  }
)

// 等待 API 请求完成
Cypress.Commands.add('waitForApi', (method: string, urlPattern: string) => {
  cy.intercept(method, urlPattern).as('apiRequest')
  cy.wait('@apiRequest', { timeout: 10000 })
})

// 检查元素存在
Cypress.Commands.add('shouldExist', { prevSubject: 'optional' }, (subject, selector) => {
  if (subject) {
    cy.wrap(subject).should('exist')
  } else {
    cy.get(selector).should('exist')
  }
})

// 检查元素不存在
Cypress.Commands.add('shouldNotExist', { prevSubject: 'optional' }, (subject, selector) => {
  if (subject) {
    cy.wrap(subject).should('not.exist')
  } else {
    cy.get(selector).should('not.exist')
  }
})

// 断言 toast 消息
Cypress.Commands.assertToast = (message: string, type = 'success') => {
  cy.get(`.el-message--${type}`).should('contain', message)
}

declare global {
  namespace Cypress {
    interface Chainable {
      login(username: string, password: string): Chainable<void>
      register(
        username: string,
        email: string,
        password: string,
        nickname?: string
      ): Chainable<void>
      apiLogin(username: string, password: string): Chainable<void>
      logout(): Chainable<void>
      createProject(name: string, description?: string, isPublic?: boolean): Chainable<void>
      waitForApi(method: string, urlPattern: string): Chainable<void>
      shouldExist(selector: string): Chainable<void>
      shouldNotExist(selector: string): Chainable<void>
    }
  }
}

export {}

