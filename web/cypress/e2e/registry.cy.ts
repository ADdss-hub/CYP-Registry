/// <reference types="cypress" />

describe('镜像推送/拉取流程测试', () => {
  const testProject = {
    name: `test-project-${Date.now()}`,
    description: '用于Cypress测试的项目',
    isPublic: false,
  }

  const testImage = {
    name: `test-image-${Date.now()}`,
    tag: 'latest',
    fullName: '', // 将在测试中填充
  }

  // 测试用户凭据
  const testUser = {
    username: Cypress.env('TEST_USERNAME') || 'testuser',
    password: Cypress.env('TEST_PASSWORD') || 'TestPassword123',
  }

  beforeEach(() => {
    // 确保登录状态
    cy.clearLocalStorage()
    cy.clearCookies()

    // 如果有保存的 token，直接设置
    if (testUser.username && testUser.password) {
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((response) => {
        if (response.status === 200 && response.body.code === 20000) {
          cy.window().then((win) => {
            win.localStorage.setItem('token', response.body.data.accessToken)
            win.localStorage.setItem('refreshToken', response.body.data.refreshToken)
            win.localStorage.setItem('user', JSON.stringify(response.body.data.user))
          })
        }
      })
    }
  })

  after(() => {
    // 清理测试数据
    cy.request({
      method: 'POST',
      url: `${Cypress.env('apiUrl')}/auth/login`,
      body: testUser,
    }).then((loginResponse) => {
      if (loginResponse.status === 200) {
        const token = loginResponse.body.data.accessToken

        // 删除测试项目
        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          if (projectsResponse.body.code === 20000) {
            const testProjectData = projectsResponse.body.data.list.find(
              (p: any) => p.name === testProject.name
            )
            if (testProjectData) {
              cy.request({
                method: 'DELETE',
                url: `${Cypress.env('apiUrl')}/projects/${testProjectData.id}`,
                headers: { Authorization: `Bearer ${token}` },
              })
            }
          }
        })
      }
    })
  })

  describe('项目管理测试', () => {
    it('应该能够创建新项目', () => {
      cy.visit('/projects')
      cy.contains('项目管理').should('exist')

      // 点击创建项目按钮
      cy.get('[data-testid="create-project-button"]').click()

      // 验证对话框出现
      cy.contains('创建项目').should('exist')

      // 填写项目信息
      cy.get('[data-testid="project-name-input"]').type(testProject.name)
      cy.get('[data-testid="project-description-input"]').type(testProject.description)

      // 提交表单
      cy.get('[data-testid="submit-button"]').click()

      // 验证项目创建成功
      cy.contains(testProject.name, { timeout: 10000 }).should('exist')
    })

    it('应该显示项目详情页面', () => {
      // 先创建项目（如果不存在）
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        // 获取项目列表
        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          const project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (project) {
            // 访问项目详情页
            cy.visit(`/projects/${project.id}`)

            // 验证页面加载
            cy.contains(testProject.name).should('exist')
            cy.contains('镜像版本').should('exist')
            cy.contains('成员管理').should('exist')
            cy.contains('项目设置').should('exist')
          }
        })
      })
    })

    it('应该能够编辑项目设置', () => {
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          const project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (project) {
            cy.visit(`/projects/${project.id}`)

            // 点击项目设置标签
            cy.contains('项目设置').click()

            // 修改项目描述
            const newDescription = '更新的描述信息'
            cy.get('[data-testid="project-description-input"]')
              .clear()
              .type(newDescription)

            // 保存设置
            cy.get('[data-testid="save-settings-button"]').click()

            // 验证更新成功
            cy.contains('保存成功').should('exist')
          }
        })
      })
    })
  })

  describe('镜像管理测试', () => {
    beforeEach(() => {
      // 确保项目存在
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        // 获取或创建测试项目
        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          let project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (!project) {
            // 创建项目
            cy.request({
              method: 'POST',
              url: `${Cypress.env('apiUrl')}/projects`,
              headers: { Authorization: `Bearer ${token}` },
              body: testProject,
            }).then((createResponse) => {
              project = createResponse.body.data
              testImage.fullName = `${testUser.username}/${testProject.name}/${testImage.name}:${testImage.tag}`
            })
          } else {
            testImage.fullName = `${testUser.username}/${testProject.name}/${testImage.name}:${testImage.tag}`
          }
        })
      })
    })

    it('应该显示镜像版本标签页', () => {
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          const project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (project) {
            cy.visit(`/projects/${project.id}`)
            cy.contains('镜像版本').click()
            cy.contains('暂无镜像').should('exist')
          }
        })
      })
    })

    it('应该能够拉取镜像（模拟）', () => {
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          const project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (project) {
            cy.visit(`/projects/${project.id}`)
            cy.contains('镜像版本').click()

            // 点击拉取镜像按钮
            cy.get('[data-testid="pull-image-button"]').click()

            // 验证对话框出现
            cy.contains('拉取镜像').should('exist')

            // 输入镜像名称
            cy.get('[data-testid="image-name-input"]').type('alpine:latest')

            // 确认拉取
            cy.get('[data-testid="confirm-pull-button"]').click()

            // 验证操作成功
            cy.contains('拉取指令已生成').should('exist')
          }
        })
      })
    })

    it('应该显示镜像列表', () => {
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          const project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (project) {
            cy.visit(`/projects/${project.id}`)
            cy.contains('镜像版本').click()

            // 验证表格存在
            cy.get('[data-testid="image-table"]').should('exist')
          }
        })
      })
    })
  })

  describe('成员管理测试', () => {
    it('应该显示成员管理标签页', () => {
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          const project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (project) {
            cy.visit(`/projects/${project.id}`)
            cy.contains('成员管理').click()

            // 验证成员管理页面元素
            cy.contains('成员列表').should('exist')
            cy.get('[data-testid="add-member-button"]').should('exist')
          }
        })
      })
    })

    it('应该能够添加项目成员', () => {
      cy.request({
        method: 'POST',
        url: `${Cypress.env('apiUrl')}/auth/login`,
        body: testUser,
      }).then((loginResponse) => {
        const token = loginResponse.body.data.accessToken

        cy.request({
          method: 'GET',
          url: `${Cypress.env('apiUrl')}/projects`,
          headers: { Authorization: `Bearer ${token}` },
        }).then((projectsResponse) => {
          const project = projectsResponse.body.data.list.find(
            (p: any) => p.name === testProject.name
          )

          if (project) {
            cy.visit(`/projects/${project.id}`)
            cy.contains('成员管理').click()

            // 点击添加成员
            cy.get('[data-testid="add-member-button"]').click()

            // 验证对话框出现
            cy.contains('添加成员').should('exist')

            // 输入成员信息
            cy.get('[data-testid="member-username-input"]').type('newmember')

            // 选择角色
            cy.get('[data-testid="member-role-select"]').click()
            cy.contains('开发者').click()

            // 确认添加
            cy.get('[data-testid="confirm-add-button"]').click()

            // 验证成员添加成功
            cy.contains('newmember').should('exist')
          }
        })
      })
    })
  })

  describe('漏洞扫描集成测试', () => {
    it('应该能够启动漏洞扫描', () => {
      cy.visit('/scans')

      // 验证扫描页面加载
      cy.contains('漏洞扫描').should('exist')
      cy.contains('立即扫描').should('exist')
      cy.contains('扫描历史').should('exist')

      // 输入镜像名称
      cy.get('[data-testid="image-name-input"]').type('alpine:latest')

      // 选择扫描策略
      cy.get('[data-testid="policy-select"]').click()
      cy.contains('全量扫描').click()

      // 点击扫描按钮
      cy.get('[data-testid="start-scan-button"]').click()

      // 验证扫描任务创建成功
      cy.contains('扫描任务已创建', { timeout: 5000 }).should('exist')
    })

    it('应该显示扫描历史', () => {
      cy.visit('/scans')
      cy.contains('扫描历史').click()

      // 验证历史表格存在
      cy.get('[data-testid="scan-history-table"]').should('exist')
    })
  })

  describe('Webhook集成测试', () => {
    const testWebhook = {
      name: `test-webhook-${Date.now()}`,
      url: 'https://example.com/webhook',
      events: ['image_push', 'scan_completed'],
    }

    it('应该能够创建Webhook', () => {
      cy.visit('/webhooks')

      // 验证Webhook页面加载
      cy.contains('Webhook管理').should('exist')

      // 点击创建Webhook
      cy.get('[data-testid="create-webhook-button"]').click()

      // 验证对话框出现
      cy.contains('创建Webhook').should('exist')

      // 填写Webhook信息
      cy.get('[data-testid="webhook-name-input"]').type(testWebhook.name)
      cy.get('[data-testid="webhook-url-input"]').type(testWebhook.url)

      // 选择事件类型
      cy.get('[data-testid="image-push-event"]').check()
      cy.get('[data-testid="scan-completed-event"]').check()

      // 提交表单
      cy.get('[data-testid="submit-button"]').click()

      // 验证Webhook创建成功
      cy.contains(testWebhook.name, { timeout: 10000 }).should('exist')
    })

    it('应该能够测试Webhook', () => {
      cy.visit('/webhooks')

      // 找到测试Webhook
      cy.contains(testWebhook.name).parent().parent().as('webhookRow')

      // 点击测试按钮
      cy.get('@webhookRow').find('[data-testid="test-button"]').click()

      // 验证测试对话框出现
      cy.contains('测试Webhook').should('exist')

      // 确认测试
      cy.get('[data-testid="confirm-test-button"]').click()

      // 验证测试结果
      cy.contains('测试请求已发送', { timeout: 10000 }).should('exist')
    })
  })

  describe('系统设置测试', () => {
    it('应该能够访问系统设置', () => {
      cy.visit('/settings')

      // 验证设置页面加载
      cy.contains('系统设置').should('exist')

      // 验证各设置标签存在
      cy.contains('个人资料').should('exist')
      cy.contains('安全设置').should('exist')
      cy.contains('通知设置').should('exist')
      cy.contains('访问令牌').should('exist')
      cy.contains('外观设置').should('exist')
    })

    it('应该能够创建访问令牌', () => {
      const tokenName = `test-token-${Date.now()}`

      cy.visit('/settings')
      cy.contains('访问令牌').click()

      // 点击创建令牌
      cy.get('[data-testid="create-token-button"]').click()

      // 验证对话框出现
      cy.contains('创建访问令牌').should('exist')

      // 填写令牌信息
      cy.get('[data-testid="token-name-input"]').type(tokenName)

      // 选择权限范围
      cy.get('[data-testid="scope-read"]').check()
      cy.get('[data-testid="scope-write"]').check()

      // 设置过期时间
      cy.get('[data-testid="expiry-select"]').click()
      cy.contains('30天').click()

      // 创建令牌
      cy.get('[data-testid="create-button"]').click()

      // 验证令牌创建成功
      cy.contains(tokenName).should('exist')
      cy.contains('复制').should('exist')
    })

    it('应该能够切换主题', () => {
      cy.visit('/settings')
      cy.contains('外观设置').click()

      // 验证主题选项存在
      cy.contains('浅色模式').should('exist')
      cy.contains('深色模式').should('exist')
      cy.contains('跟随系统').should('exist')

      // 切换到深色模式
      cy.get('[data-testid="dark-theme-option"]').click()

      // 验证主题切换成功
      cy.get('html').should('have.class', 'dark')
    })
  })

  describe('响应式设计测试', () => {
    it('应该在移动端正确显示导航菜单', () => {
      // 设置移动端视口
      cy.viewport('iphone-x')

      cy.visit('/dashboard')

      // 验证汉堡菜单按钮存在
      cy.get('[data-testid="sidebar-toggle"]').should('exist')

      // 点击展开菜单
      cy.get('[data-testid="sidebar-toggle"]').click()

      // 验证菜单项可见
      cy.contains('仪表盘').should('be.visible')
      cy.contains('项目管理').should('be.visible')
    })

    it('应该在平板端正确显示布局', () => {
      // 设置平板视口
      cy.viewport('ipad-2')

      cy.visit('/dashboard')

      // 验证侧边栏可见
      cy.get('[data-testid="sidebar"]').should('be.visible')

      // 验证主内容区存在
      cy.get('[data-testid="main-content"]').should('exist')
    })
  })
})

