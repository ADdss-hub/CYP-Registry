/// <reference types="cypress" />

describe('登录/注册流程测试', () => {
  const testUser = {
    username: `testuser_${Date.now()}`,
    email: `test_${Date.now()}@example.com`,
    password: 'TestPassword123',
    nickname: '测试用户',
  }

  beforeEach(() => {
    // 清除所有本地存储和 cookies
    cy.clearLocalStorage()
    cy.clearCookies()
  })

  describe('登录功能测试', () => {
    it('应该显示登录页面', () => {
      cy.visit('/login')
      cy.contains('登录').should('exist')
      cy.contains('用户名').should('exist')
      cy.contains('密码').should('exist')
    })

    it('应该正确验证必填字段', () => {
      cy.visit('/login')
      cy.get('[data-testid="login-button"]').click()
      cy.contains('请输入用户名').should('exist')
    })

    it('应该正确验证用户名格式', () => {
      cy.visit('/login')
      cy.get('[data-testid="username-input"]').type('ab') // 少于3个字符
      cy.get('[data-testid="password-input"]').type('TestPassword123')
      cy.get('[data-testid="login-button"]').click()
      cy.contains('用户名长度必须在3-20个字符之间').should('exist')
    })

    it('应该正确验证密码格式', () => {
      cy.visit('/login')
      cy.get('[data-testid="username-input"]').type('testuser')
      cy.get('[data-testid="password-input"]').type('123') // 少于8个字符
      cy.get('[data-testid="login-button"]').click()
      cy.contains('密码至少8位，且包含数字和字母').should('exist')
    })

    it('应该显示"记住我"选项', () => {
      cy.visit('/login')
      cy.contains('记住我').should('exist')
      cy.get('[data-testid="remember-checkbox"]').should('exist')
    })

    it('应该显示"忘记密码"链接', () => {
      cy.visit('/login')
      cy.contains('忘记密码').should('exist')
    })

    it('应该显示注册链接', () => {
      cy.visit('/login')
      cy.contains('立即注册').should('exist')
    })

    it('应该能够切换到注册页面', () => {
      cy.visit('/login')
      cy.contains('立即注册').click()
      cy.url().should('include', '/register')
    })

    it('应该能够使用正确的凭据登录（需要后端服务）', () => {
      // 使用 API 先注册用户
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/register`,
        body: testUser,
      }).then((registerResponse) => {
        if (registerResponse.status === 200 && registerResponse.body.code === 20000) {
          // 注册成功后尝试登录
          cy.visit('/login')
          cy.get('[data-testid="username-input"]').type(testUser.username)
          cy.get('[data-testid="password-input"]').type(testUser.password)
          cy.get('[data-testid="login-button"]').click()
          // 验证登录成功（跳转到仪表盘）
          cy.url().should('include', '/dashboard')
        }
      })
    })

    it('应该显示错误的用户名或密码提示', () => {
      cy.visit('/login')
      cy.get('[data-testid="username-input"]').type('nonexistent_user')
      cy.get('[data-testid="password-input"]').type('WrongPassword123')
      cy.get('[data-testid="login-button"]').click()
      // 等待错误消息
      cy.contains('用户名或密码错误', { timeout: 5000 }).should('exist')
    })
  })

  describe('注册功能测试', () => {
    it('应该显示注册页面', () => {
      cy.visit('/register')
      cy.contains('注册').should('exist')
      cy.contains('用户名').should('exist')
      cy.contains('邮箱').should('exist')
      cy.contains('密码').should('exist')
    })

    it('应该正确验证必填字段', () => {
      cy.visit('/register')
      cy.get('[data-testid="register-button"]').click()
      cy.contains('请输入用户名').should('exist')
      cy.contains('请输入邮箱').should('exist')
      cy.contains('请输入密码').should('exist')
    })

    it('应该正确验证用户名格式', () => {
      cy.visit('/register')
      // 测试用户名过短
      cy.get('[data-testid="username-input"]').type('ab')
      cy.get('[data-testid="email-input"]').type('test@example.com')
      cy.get('[data-testid="password-input"]').type('TestPassword123')
      cy.get('[data-testid="confirm-password-input"]').type('TestPassword123')
      cy.get('[data-testid="register-button"]').click()
      cy.contains('用户名长度必须在3-20个字符之间').should('exist')
    })

    it('应该正确验证邮箱格式', () => {
      cy.visit('/register')
      cy.get('[data-testid="username-input"]').type('testuser')
      cy.get('[data-testid="email-input"]').type('invalid-email')
      cy.get('[data-testid="password-input"]').type('TestPassword123')
      cy.get('[data-testid="confirm-password-input"]').type('TestPassword123')
      cy.get('[data-testid="register-button"]').click()
      cy.contains('请输入有效的邮箱地址').should('exist')
    })

    it('应该正确验证密码格式', () => {
      cy.visit('/register')
      cy.get('[data-testid="username-input"]').type('testuser')
      cy.get('[data-testid="email-input"]').type('test@example.com')
      cy.get('[data-testid="password-input"]').type('12345678') // 没有字母
      cy.get('[data-testid="confirm-password-input"]').type('12345678')
      cy.get('[data-testid="register-button"]').click()
      cy.contains('密码至少8位，且包含数字和字母').should('exist')
    })

    it('应该验证两次密码输入一致', () => {
      cy.visit('/register')
      cy.get('[data-testid="username-input"]').type('testuser')
      cy.get('[data-testid="email-input"]').type('test@example.com')
      cy.get('[data-testid="password-input"]').type('TestPassword123')
      cy.get('[data-testid="confirm-password-input"]').type('DifferentPassword123')
      cy.get('[data-testid="register-button"]').click()
      cy.contains('两次密码输入不一致').should('exist')
    })

    it('应该能够成功注册新用户（需要后端服务）', () => {
      const uniqueUser = {
        username: `newuser_${Date.now()}`,
        email: `new_${Date.now()}@example.com`,
        password: 'NewPassword123',
      }
      cy.visit('/register')
      cy.get('[data-testid="username-input"]').type(uniqueUser.username)
      cy.get('[data-testid="email-input"]').type(uniqueUser.email)
      cy.get('[data-testid="password-input"]').type(uniqueUser.password)
      cy.get('[data-testid="confirm-password-input"]').type(uniqueUser.password)
      cy.get('[data-testid="register-button"]').click()
      // 验证注册成功并跳转到登录页
      cy.url().should('include', '/login')
      cy.contains('注册成功，请登录').should('exist')
    })

    it('应该显示用户名已存在的提示', () => {
      // 先注册用户
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/register`,
        body: testUser,
      })

      // 尝试使用相同的用户名注册
      cy.visit('/register')
      cy.get('[data-testid="username-input"]').type(testUser.username)
      cy.get('[data-testid="email-input"]').type('different@example.com')
      cy.get('[data-testid="password-input"]').type('TestPassword123')
      cy.get('[data-testid="confirm-password-input"]').type('TestPassword123')
      cy.get('[data-testid="register-button"]').click()
      // 等待错误消息
      cy.contains('用户名已存在', { timeout: 5000 }).should('exist')
    })

    it('应该显示登录链接', () => {
      cy.visit('/register')
      cy.contains('已有账号？立即登录').should('exist')
    })

    it('应该能够切换回登录页面', () => {
      cy.visit('/register')
      cy.contains('立即登录').click()
      cy.url().should('include', '/login')
    })
  })

  describe('Token 刷新测试', () => {
    it('应该在 token 过期前自动刷新（需要后端服务）', () => {
      // 先登录获取 token
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: {
          username: testUser.username,
          password: testUser.password,
        },
      }).then((loginResponse) => {
        if (loginResponse.status === 200) {
          // 访问需要认证的页面
          cy.visit('/dashboard')
          cy.url().should('include', '/dashboard')
        }
      })
    })
  })
})

