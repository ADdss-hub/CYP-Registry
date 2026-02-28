// 通用类型定义

// 用户相关
export interface User {
  id: string;
  username: string;
  email: string;
  nickname: string;
  avatar: string;
  bio: string;
  is_active: boolean;
  is_admin: boolean;
  created_at: string;
  last_login_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

// 项目相关
export interface Project {
  id: string;
  name: string;
  description: string;
  ownerId: string;
  isPublic: boolean;
  storageUsed: number;
  storageQuota: number;
  imageCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateProjectRequest {
  name: string;
  description: string;
  isPublic: boolean;
  storageQuota?: number;
}

export interface UpdateProjectRequest {
  name?: string;
  description?: string;
  isPublic?: boolean;
  storageQuota?: number;
}

// 镜像相关
export interface Image {
  name: string;
  tags: ImageTag[];
  manifest: Manifest;
  size: number;
  pullCount: number;
  lastPushTime: string;
  digest: string;
}

export interface ImageTag {
  name: string;
  digest: string;
  size: number;
  lastPushTime: string;
  pushedBy: string;
}

export interface Manifest {
  schemaVersion: number;
  mediaType: string;
  config: LayerInfo;
  layers: LayerInfo[];
}

export interface LayerInfo {
  mediaType: string;
  digest: string;
  size: number;
}

// 扫描相关
export interface ScanTask {
  taskId: string;
  projectId: string;
  imageName: string;
  digest: string;
  reference: string;
  policyId: string;
  triggerType: string;
  status: ScanStatus;
  priority: number;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  error?: string;
}

export type ScanStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "blocked";

export interface Vulnerability {
  id: string;
  severity: SeverityLevel;
  title: string;
  description: string;
  fixedVersion: string;
  packageName: string;
  packageVersion: string;
  type: VulnerabilityType;
  references: string[];
  cvssScore: number;
}

export type SeverityLevel = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW" | "UNKNOWN";

export type VulnerabilityType = "os-package" | "library" | "application";

export interface VulnerabilityReport {
  reportId: string;
  projectId: string;
  imageName: string;
  digest: string;
  scanStatus: ScanStatus;
  scanTime: string;
  scannerVersion: string;
  osInfo?: OSInfo;
  summary: VulnerabilitySummary;
  vulnerabilities: Vulnerability[];
  blockedReasons?: string[];
}

export interface OSInfo {
  name: string;
  version: string;
}

export interface VulnerabilitySummary {
  criticalCount: number;
  highCount: number;
  mediumCount: number;
  lowCount: number;
  unknownCount: number;
  totalCount: number;
}

export interface ScanPolicy {
  policyId: string;
  name: string;
  description: string;
  blockOnCritical: boolean;
  blockOnHigh: boolean;
  blockOnMedium: boolean;
  criticalSeverityThreshold: number;
  highSeverityThreshold: number;
  excludedCveIds: string[];
  maxScanTimeout: number;
  ignoreUnfixed: boolean;
}

// Webhook相关
export interface Webhook {
  webhookId: string;
  projectId: string;
  name: string;
  description: string;
  url: string;
  secret: string;
  events: WebhookEventType[];
  isActive: boolean;
  headers: Record<string, string>;
  retryPolicy: RetryPolicy;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  lastTriggeredAt?: string;
  successCount?: number;
  failedCount?: number;
}

export type WebhookEventType =
  | "push"
  | "pull"
  | "delete"
  | "scan"
  | "scan_fail"
  | "policy";

export interface RetryPolicy {
  maxRetries: number;
  retryDelay: number;
  backoffMultiplier: number;
  maxDelay: number;
}

export interface WebhookEvent {
  eventId: string;
  webhookId: string;
  eventType: WebhookEventType;
  projectId: string;
  repository: string;
  tag?: string;
  digest?: string;
  userId?: string;
  username?: string;
  payload: Record<string, unknown>;
  timestamp: string;
  status: WebhookEventStatus;
  attempts: number;
  lastAttemptAt?: string;
  nextRetryAt?: string;
  error?: string;
}

export type WebhookEventStatus = "pending" | "sent" | "failed" | "retrying";

export interface WebhookDelivery {
  deliveryId: string;
  eventId: string;
  webhookId: string;
  requestUrl: string;
  requestMethod: string;
  requestHeaders: Record<string, string>;
  responseStatus: number;
  responseHeaders: Record<string, string>;
  responseBody: string;
  duration: number;
  deliveredAt: string;
  error?: string;
}

// API响应
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

// 分页参数
export interface PaginationParams {
  page: number;
  pageSize: number;
}

// 错误处理
export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

// 访问令牌
export interface AccessToken {
  id: string;
  name: string;
  token: string;
  createdAt: string;
  lastUsedAt?: string;
  expiresAt?: string;
  scopes?: string[];
}

// 通知设置
export interface NotificationSettings {
  email_enabled: boolean;
  scan_completed: boolean;
  security_alerts: boolean;
  webhook_notifications: boolean;
  digest: "realtime" | "daily" | "weekly";
  notification_email: string;
}
