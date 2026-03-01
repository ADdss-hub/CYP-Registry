<script setup lang="ts">
import { ref } from "vue";
import CypButton from "@/components/common/CypButton.vue";
import CypCard from "@/components/common/CypCard.vue";

interface EndpointParam {
  name: string;
  type: string;
  required: boolean;
  desc: string;
}

interface Endpoint {
  method: string;
  path: string;
  description: string;
  params?: EndpointParam[];
  auth?: boolean;
  response: any;
}

const activeEndpoint = ref<string | null>(null);

const apiSections: Array<{
  name: string;
  endpoints: Endpoint[];
}> = [
  {
    name: "认证",
    endpoints: [
      {
        method: "POST",
        path: "/api/v1/auth/login",
        description: "用户登录",
        params: [
          { name: "username", type: "string", required: true, desc: "用户名" },
          { name: "password", type: "string", required: true, desc: "密码" },
        ],
        response: {
          code: 20000,
          data: {
            user: { id: "uuid", username: "string" },
            accessToken: "string",
            refreshToken: "string",
            expiresIn: 3600,
          },
        },
      },
      {
        method: "POST",
        path: "/api/v1/auth/refresh",
        description: "刷新Token",
        params: [
          {
            name: "refreshToken",
            type: "string",
            required: true,
            desc: "刷新令牌",
          },
        ],
        response: {
          code: 20000,
          data: {
            accessToken: "string",
            refreshToken: "string",
            expiresIn: 3600,
          },
        },
      },
      {
        method: "POST",
        path: "/api/v1/auth/logout",
        description: "用户登出（后端仅记录，前端需删除本地Token）",
        auth: true,
        response: { code: 20000, message: "logout ok" },
      },
    ],
  },
  {
    name: "用户管理",
    endpoints: [
      {
        method: "GET",
        path: "/api/v1/users/me",
        description: "获取当前用户信息",
        auth: true,
        response: {
          code: 20000,
          data: {
            id: "uuid",
            username: "string",
            email: "string",
            nickname: "string",
            avatar: "string",
            bio: "string",
            isAdmin: false,
            createdAt: "2026-01-31T10:00:00Z",
          },
        },
      },
      {
        method: "PUT",
        path: "/api/v1/users/me",
        description: "更新当前用户信息",
        auth: true,
        params: [
          { name: "nickname", type: "string", required: false, desc: "昵称" },
          { name: "avatar", type: "string", required: false, desc: "头像URL" },
          { name: "bio", type: "string", required: false, desc: "个人简介" },
        ],
        response: { code: 20000, message: "更新成功" },
      },
      {
        method: "PUT",
        path: "/api/v1/users/me/password",
        description: "修改密码",
        auth: true,
        params: [
          {
            name: "oldPassword",
            type: "string",
            required: true,
            desc: "旧密码",
          },
          {
            name: "newPassword",
            type: "string",
            required: true,
            desc: "新密码",
          },
        ],
        response: { code: 20000, message: "密码修改成功" },
      },
    ],
  },
  {
    name: "访问令牌 (PAT)",
    endpoints: [
      {
        method: "POST",
        path: "/api/v1/users/me/pat",
        description: "创建 Personal Access Token（PAT）",
        auth: true,
        params: [
          { name: "name", type: "string", required: true, desc: "令牌名称" },
          {
            name: "scopes",
            type: "array",
            required: true,
            desc: "权限范围数组，可选值：read（读取）、write（写入）、delete（删除）、admin（管理）。选择什么权限就是什么权限。",
          },
          {
            name: "expire_in",
            type: "number",
            required: false,
            desc: "有效期（秒），0表示使用默认过期时间，-1表示永不过期",
          },
        ],
        response: {
          code: 20000,
          data: {
            id: "uuid",
            name: "string",
            scopes: ["read", "write"],
            expires_at: "2026-01-01T10:00:00Z",
            created_at: "2026-01-01T10:00:00Z",
            token: "pat_v1_实际一次性返回的明文令牌",
            token_type: "pat",
          },
        },
      },
      {
        method: "GET",
        path: "/api/v1/users/me/pat",
        description: "列出当前用户的所有 PAT",
        auth: true,
        response: {
          code: 20000,
          data: [
            {
              id: "uuid",
              name: "string",
              scopes: ["read", "write"],
              expires_at: "2026-01-01T10:00:00Z",
              created_at: "2026-01-01T10:00:00Z",
              last_used_at: "2026-01-15T10:00:00Z",
            },
          ],
        },
      },
      {
        method: "DELETE",
        path: "/api/v1/users/me/pat/{id}",
        description: "撤销指定 PAT",
        auth: true,
        response: { code: 20000, message: "PAT已删除" },
      },
    ],
  },
  {
    name: "管理员功能",
    endpoints: [
      {
        method: "GET",
        path: "/api/v1/admin/logs",
        description: "获取系统审计日志列表（需要管理员权限和admin scope）",
        auth: true,
        params: [
          {
            name: "page",
            type: "number",
            required: false,
            desc: "页码，从1开始，默认1",
          },
          {
            name: "page_size",
            type: "number",
            required: false,
            desc: "每页数量，默认20，最大100",
          },
          {
            name: "user_id",
            type: "string",
            required: false,
            desc: "用户ID筛选（UUID格式）",
          },
          {
            name: "action",
            type: "string",
            required: false,
            desc: "操作类型筛选",
          },
          {
            name: "resource",
            type: "string",
            required: false,
            desc: "资源类型筛选",
          },
          {
            name: "start_time",
            type: "string",
            required: false,
            desc: "开始时间（RFC3339格式，如：2026-01-01T00:00:00Z）",
          },
          {
            name: "end_time",
            type: "string",
            required: false,
            desc: "结束时间（RFC3339格式）",
          },
          {
            name: "keyword",
            type: "string",
            required: false,
            desc: "关键词搜索（在操作详情中搜索）",
          },
        ],
        response: {
          code: 20000,
          data: {
            logs: [
              {
                id: "uuid",
                user_id: "uuid",
                action: "login",
                resource: "user",
                resource_id: "uuid",
                ip: "192.168.1.1",
                user_agent: "Mozilla/5.0...",
                details: "用户登录",
                status: "success",
                created_at: "2026-01-01T10:00:00Z",
              },
            ],
            total: 100,
            page: 1,
            page_size: 20,
            total_page: 5,
          },
        },
      },
    ],
  },
  {
    name: "项目管理",
    endpoints: [
      {
        method: "GET",
        path: "/api/v1/projects",
        description: "获取当前用户可见的项目列表",
        auth: true,
        params: [
          {
            name: "page",
            type: "number",
            required: false,
            desc: "页码，默认 1",
          },
          {
            name: "page_size",
            type: "number",
            required: false,
            desc: "每页数量，默认 20，最大 100",
          },
        ],
        response: {
          code: 20000,
          data: {
            projects: [{ id: "uuid", name: "string", description: "string" }],
            total: 10,
            page: 1,
            page_size: 20,
          },
        },
      },
      {
        method: "POST",
        path: "/api/v1/projects",
        description: "创建项目",
        auth: true,
        params: [
          {
            name: "name",
            type: "string",
            required: true,
            desc: '项目名称（即 Registry 仓库名，可包含命名空间，例如 "team1/app-service"；建议仅使用字母、数字、-、_、/ 和 .）',
          },
          {
            name: "description",
            type: "string",
            required: false,
            desc: "描述",
          },
        ],
        response: {
          code: 20000,
          data: {
            project: {
              id: "uuid",
              name: "string",
              description: "string",
            },
          },
        },
      },
      {
        method: "GET",
        path: "/api/v1/projects/{id}",
        description: "获取项目详情",
        auth: true,
        response: {
          code: 20000,
          data: {
            project: {
              id: "uuid",
              name: "string",
              description: "string",
              isPublic: false,
              storageUsed: 1024,
              storageQuota: 10737418240,
              imageCount: 5,
              createdAt: "2026-01-01T10:00:00Z",
            },
          },
        },
      },
      {
        method: "PUT",
        path: "/api/v1/projects/{id}",
        description: "更新项目信息（仅项目所有者可更新）",
        auth: true,
        params: [
          { name: "name", type: "string", required: false, desc: "项目名称" },
          {
            name: "description",
            type: "string",
            required: false,
            desc: "描述",
          },
          {
            name: "isPublic",
            type: "boolean",
            required: false,
            desc: "是否公开",
          },
        ],
        response: {
          code: 20000,
          data: { message: "project updated successfully" },
        },
      },
      {
        method: "DELETE",
        path: "/api/v1/projects/{id}",
        description: "删除项目（仅项目所有者可删除）",
        auth: true,
        response: {
          code: 20000,
          data: { message: "project deleted successfully" },
        },
      },
      {
        method: "PUT",
        path: "/api/v1/projects/{id}/quota",
        description: "更新项目存储配额（仅项目所有者可操作）",
        auth: true,
        params: [
          {
            name: "quota",
            type: "number",
            required: true,
            desc: "新的配额，单位：字节",
          },
        ],
        response: {
          code: 20000,
          data: { message: "quota updated" },
        },
      },
    ],
  },
  // 原“漏洞扫描”接口文档已移除
  {
    name: "Webhook",
    endpoints: [
      {
        method: "GET",
        path: "/api/v1/webhooks",
        description: "获取指定项目的 Webhook 列表",
        auth: true,
        params: [
          { name: "projectId", type: "string", required: true, desc: "项目ID" },
        ],
        response: {
          code: 20000,
          data: [
            {
              id: "uuid",
              name: "string",
              url: "https://example.com",
              events: ["push"],
            },
          ],
        },
      },
      {
        method: "POST",
        path: "/api/v1/webhooks",
        description: "创建 Webhook",
        auth: true,
        params: [
          { name: "projectId", type: "string", required: true, desc: "项目ID" },
          { name: "name", type: "string", required: true, desc: "名称" },
          { name: "url", type: "string", required: true, desc: "回调URL" },
          {
            name: "events",
            type: "array",
            required: true,
            desc: "事件类型列表",
          },
          {
            name: "secret",
            type: "string",
            required: false,
            desc: "签名密钥（可选）",
          },
        ],
        response: {
          code: 201,
          data: { id: "uuid", name: "string", url: "https://example.com" },
        },
      },
      {
        method: "PUT",
        path: "/api/v1/webhooks/{webhookId}",
        description: "更新 Webhook",
        auth: true,
        params: [
          { name: "name", type: "string", required: false, desc: "名称" },
          { name: "url", type: "string", required: false, desc: "回调URL" },
          {
            name: "events",
            type: "array",
            required: false,
            desc: "事件类型列表",
          },
          { name: "secret", type: "string", required: false, desc: "签名密钥" },
        ],
        response: { code: 200, message: "Webhook updated successfully" },
      },
      {
        method: "POST",
        path: "/api/v1/webhooks/{webhookId}/test",
        description: "测试 Webhook",
        auth: true,
        response: {
          code: 200,
          data: { responseStatus: 200, duration: 150 },
        },
      },
      {
        method: "DELETE",
        path: "/api/v1/webhooks/{webhookId}",
        description: "删除 Webhook",
        auth: true,
        response: { code: 204, message: "Webhook deleted" },
      },
      {
        method: "GET",
        path: "/api/v1/webhooks/statistics",
        description: "获取 Webhook 统计信息",
        auth: true,
        response: {
          code: 200,
          data: {
            total: 10,
            active: 8,
            totalTriggers: 123,
          },
        },
      },
    ],
  },
];

const methodColors: Record<string, string> = {
  GET: "#22c55e",
  POST: "#6366f1",
  PUT: "#f59e0b",
  DELETE: "#ef4444",
  PATCH: "#8b5cf6",
};

function toggleEndpoint(sectionName: string) {
  if (activeEndpoint.value === sectionName) {
    activeEndpoint.value = null;
  } else {
    activeEndpoint.value = sectionName;
  }
}

function openSwaggerUI() {
  window.open("/swagger/index.html", "_blank");
}
</script>

<template>
  <div class="api-docs-page">
    <div class="page-header">
      <div class="header-content">
        <h2>API 文档</h2>
        <p>CYP-Registry RESTful API 接口文档</p>
      </div>
      <div class="header-actions">
        <CypButton type="primary" @click="openSwaggerUI">
          <svg
            viewBox="0 0 24 24"
            width="16"
            height="16"
            style="margin-right: 6px"
          >
            <path
              fill="currentColor"
              d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zM6 20V4h7v5h5v11H6z"
            />
          </svg>
          打开 Swagger UI
        </CypButton>
      </div>
    </div>

    <div class="docs-content">
      <div class="docs-info">
        <CypCard>
          <template #header>
            <h3>快速开始</h3>
          </template>
          <div class="info-section">
            <h4>认证方式</h4>
            <p>所有API接口（除认证相关）都需要在请求头中携带Access Token：</p>
            <pre class="code-block">
Authorization: Bearer &lt;your-access-token&gt;</pre
            >
            <p style="margin-top: 8px; font-size: 12px; color: #64748b">
              支持两种Token类型：
            </p>
            <ul style="margin: 8px 0 0 20px; font-size: 12px; color: #64748b">
              <li>
                <strong>JWT Token</strong>：通过登录接口获取，继承用户所有权限
              </li>
              <li>
                <strong>PAT Token</strong>：Personal Access
                Token，可自定义权限范围
              </li>
            </ul>
          </div>
          <div class="info-section">
            <h4>PAT权限说明</h4>
            <p style="font-size: 12px; color: #64748b; margin-bottom: 8px">
              PAT支持以下权限范围（scopes）：
            </p>
            <table class="error-table" style="font-size: 12px">
              <thead>
                <tr>
                  <th>权限</th>
                  <th>说明</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td><code>read</code></td>
                  <td>读取权限：拉取镜像、查看项目信息</td>
                </tr>
                <tr>
                  <td><code>write</code></td>
                  <td>写入权限：推送镜像、创建/更新项目（包含read权限）</td>
                </tr>
                <tr>
                  <td><code>delete</code></td>
                  <td>删除权限：删除镜像、删除项目（包含write和read权限）</td>
                </tr>
                <tr>
                  <td><code>admin</code></td>
                  <td>
                    管理权限：访问管理员功能、查看日志、用户管理（包含所有权限）
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="info-section">
            <h4>基础URL</h4>
            <pre class="code-block">http(s)://&lt;your-host&gt;/api/v1</pre>
          </div>
          <div class="info-section">
            <h4>响应格式</h4>
            <pre class="code-block">
{
  "code": 20000,
  "message": "success",
  "data": {}
}</pre
            >
          </div>
          <div class="info-section">
            <h4>错误码说明</h4>
            <table class="error-table">
              <thead>
                <tr>
                  <th>错误码</th>
                  <th>说明</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>20000</td>
                  <td>请求成功</td>
                </tr>
                <tr>
                  <td>10001</td>
                  <td>参数错误</td>
                </tr>
                <tr>
                  <td>30001</td>
                  <td>未登录</td>
                </tr>
                <tr>
                  <td>30009</td>
                  <td>用户名或密码错误</td>
                </tr>
                <tr>
                  <td>30010</td>
                  <td>Token无效</td>
                </tr>
                <tr>
                  <td>30011</td>
                  <td>Token过期</td>
                </tr>
                <tr>
                  <td>40001</td>
                  <td>资源不存在</td>
                </tr>
                <tr>
                  <td>30003</td>
                  <td>禁止访问</td>
                </tr>
                <tr>
                  <td>30004</td>
                  <td>权限不足（通用）</td>
                </tr>
                <tr>
                  <td>30014</td>
                  <td>PAT缺少读取权限（需要选择'读取'权限）</td>
                </tr>
                <tr>
                  <td>30015</td>
                  <td>PAT缺少写入权限（需要选择'写入'权限）</td>
                </tr>
                <tr>
                  <td>30016</td>
                  <td>PAT缺少删除权限（需要选择'删除'权限）</td>
                </tr>
                <tr>
                  <td>30017</td>
                  <td>PAT缺少管理员权限（需要选择'管理'权限）</td>
                </tr>
                <tr>
                  <td>30018</td>
                  <td>PAT缺少权限信息</td>
                </tr>
                <tr>
                  <td>30019</td>
                  <td>PAT权限信息格式错误</td>
                </tr>
                <tr>
                  <td>40001</td>
                  <td>资源不存在</td>
                </tr>
                <tr>
                  <td>40002</td>
                  <td>权限不足</td>
                </tr>
              </tbody>
            </table>
          </div>
        </CypCard>
      </div>

      <div class="docs-endpoints">
        <CypCard v-for="section in apiSections" :key="section.name">
          <template #header>
            <div class="section-header" @click="toggleEndpoint(section.name)">
              <h3>{{ section.name }}</h3>
              <svg
                viewBox="0 0 24 24"
                width="20"
                height="20"
                :style="{
                  transform:
                    activeEndpoint === section.name ? 'rotate(180deg)' : '',
                }"
              >
                <path fill="currentColor" d="M7 10l5 5 5-5z" />
              </svg>
            </div>
          </template>

          <div v-show="activeEndpoint === section.name" class="endpoints-list">
            <div
              v-for="(endpoint, index) in section.endpoints"
              :key="index"
              class="endpoint-item"
            >
              <div class="endpoint-header">
                <span
                  class="method"
                  :style="{ background: methodColors[endpoint.method] }"
                >
                  {{ endpoint.method }}
                </span>
                <code class="path">{{ endpoint.path }}</code>
                <span class="description">{{ endpoint.description }}</span>
                <span v-if="endpoint.auth" class="auth-badge">需要认证</span>
              </div>

              <div class="endpoint-details">
                <div
                  v-if="endpoint.params && endpoint.params.length > 0"
                  class="params-section"
                >
                  <h5>请求参数</h5>
                  <table class="params-table">
                    <thead>
                      <tr>
                        <th>参数名</th>
                        <th>类型</th>
                        <th>必填</th>
                        <th>说明</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="param in endpoint.params" :key="param.name">
                        <td>
                          <code>{{ param.name }}</code>
                        </td>
                        <td>{{ param.type }}</td>
                        <td>
                          <span
                            :class="['required', param.required ? 'yes' : 'no']"
                          >
                            {{ param.required ? "是" : "否" }}
                          </span>
                        </td>
                        <td>{{ param.desc }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>

                <div class="response-section">
                  <h5>响应示例</h5>
                  <pre class="response-block">{{
                    JSON.stringify(endpoint.response, null, 2)
                  }}</pre>
                </div>
              </div>
            </div>
          </div>
        </CypCard>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.api-docs-page {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;

  h2 {
    font-size: 28px;
    font-weight: 700;
    line-height: 1.3;
    color: var(--text-primary, #1e293b);
    margin: 0 0 4px;
  }

  p {
    font-size: 14px;
    color: #64748b;
    margin: 0;
  }
}

.docs-content {
  display: grid;
  grid-template-columns: 360px 1fr;
  gap: 24px;
}

.docs-info {
  position: sticky;
  top: 88px;
  align-self: start;
}

.info-section {
  margin-bottom: 24px;

  &:last-child {
    margin-bottom: 0;
  }

  h4 {
    font-size: 14px;
    font-weight: 600;
    color: #1e293b;
    margin: 0 0 8px;
  }

  p {
    font-size: 13px;
    color: #64748b;
    margin: 0 0 8px;
  }
}

.code-block {
  background: #1e293b;
  color: #e2e8f0;
  padding: 12px;
  border-radius: 8px;
  font-size: 13px;
  overflow-x: auto;
  margin: 0;
}

.error-table,
.params-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;

  th,
  td {
    padding: 8px 12px;
    text-align: left;
    border-bottom: 1px solid #e2e8f0;
  }

  th {
    background: #f8fafc;
    font-weight: 500;
    color: #64748b;
  }

  td {
    color: #374151;

    code {
      background: #f1f5f9;
      padding: 2px 6px;
      border-radius: 4px;
      font-size: 12px;
    }
  }
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  cursor: pointer;

  h3 {
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
  }

  svg {
    color: #64748b;
    transition: transform 0.2s ease;
  }
}

.endpoints-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 16px;
}

.endpoint-item {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  overflow: hidden;
}

.endpoint-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #f8fafc;
  cursor: pointer;

  .method {
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    color: white;
    min-width: 50px;
    text-align: center;
  }

  .path {
    font-size: 13px;
    color: #1e293b;
  }

  .description {
    font-size: 13px;
    color: #64748b;
    flex: 1;
  }

  .auth-badge {
    padding: 2px 8px;
    background: #fef3c7;
    color: #d97706;
    border-radius: 4px;
    font-size: 11px;
  }
}

.endpoint-details {
  padding: 16px;
  display: none;

  .endpoint-item:not(.collapsed) & {
    display: block;
  }
}

.endpoint-item:hover .endpoint-details {
  display: block;
}

.params-section,
.response-section {
  margin-bottom: 16px;

  &:last-child {
    margin-bottom: 0;
  }

  h5 {
    font-size: 13px;
    font-weight: 600;
    color: #374151;
    margin: 0 0 12px;
  }
}

.response-block {
  background: #1e293b;
  color: #e2e8f0;
  padding: 12px;
  border-radius: 8px;
  font-size: 12px;
  overflow-x: auto;
  margin: 0;
  max-height: 300px;
}

.required {
  &.yes {
    color: #ef4444;
  }
  &.no {
    color: #22c55e;
  }
}

@media (max-width: 1024px) {
  .docs-content {
    grid-template-columns: 1fr;
  }

  .docs-info {
    position: static;
  }
}

@media (max-width: 768px) {
  .endpoint-header {
    flex-wrap: wrap;
  }

  .path {
    width: 100%;
    margin-top: 8px;
  }
}
</style>
