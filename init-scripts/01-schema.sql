-- ============================================
-- CYP-Registry 数据库初始化脚本
-- 遵循《全平台通用数据库个人管理规范》
-- ============================================

-- 创建数据库
CREATE DATABASE registry_db WITH OWNER registry;

\c registry_db

-- ============================================
-- 1. 创建扩展
-- ============================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- gen_random_uuid() 来自 pgcrypto；缺失会导致默认 UUID 生成失败
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================
-- 2. 创建枚举类型
-- ============================================
-- 用户状态
DO $$ BEGIN
    CREATE TYPE user_status AS ENUM ('active', 'locked', 'banned');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 扫描状态
DO $$ BEGIN
    CREATE TYPE scan_status AS ENUM ('pending', 'scanning', 'success', 'failed');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 事件严重程度
DO $$ BEGIN
    CREATE TYPE event_severity AS ENUM ('low', 'medium', 'high', 'critical');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- ============================================
-- 3. 创建表结构
-- ============================================

-- 用户表
CREATE TABLE IF NOT EXISTS registry_users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username            VARCHAR(64) NOT NULL UNIQUE,
    email               VARCHAR(255) NOT NULL UNIQUE,
    password            VARCHAR(255) NOT NULL,
    nickname            VARCHAR(128),
    avatar              VARCHAR(512),
    bio                 TEXT,
    is_active           BOOLEAN DEFAULT TRUE,
    is_admin            BOOLEAN DEFAULT FALSE,
    first_login         BOOLEAN DEFAULT FALSE,
    last_login_at       TIMESTAMP,
    last_login_ip       VARCHAR(45),
    login_count         INTEGER DEFAULT 0,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- 项目表
CREATE TABLE IF NOT EXISTS registry_projects (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(128) NOT NULL UNIQUE,
    description         TEXT,
    owner_id            UUID NOT NULL REFERENCES registry_users(id),
    is_public           BOOLEAN DEFAULT FALSE,
    storage_used        BIGINT DEFAULT 0,
    storage_quota       BIGINT DEFAULT 10737418240, -- 10GB
    image_count         INTEGER DEFAULT 0,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- 角色表
CREATE TABLE IF NOT EXISTS registry_roles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(64) NOT NULL UNIQUE,
    display_name        VARCHAR(128),
    description         TEXT,
    is_system           BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- 权限表
CREATE TABLE IF NOT EXISTS registry_permissions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(128) NOT NULL UNIQUE,
    name                VARCHAR(128) NOT NULL,
    description         TEXT,
    resource            VARCHAR(64),
    action              VARCHAR(32),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- 项目成员表
CREATE TABLE IF NOT EXISTS registry_project_members (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES registry_projects(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES registry_users(id) ON DELETE CASCADE,
    role_id             UUID NOT NULL REFERENCES registry_roles(id),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP,
    UNIQUE(project_id, user_id)
);

-- 角色权限表
CREATE TABLE IF NOT EXISTS registry_role_permissions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id             UUID NOT NULL REFERENCES registry_roles(id) ON DELETE CASCADE,
    permission_id       UUID NOT NULL REFERENCES registry_permissions(id) ON DELETE CASCADE,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP,
    UNIQUE(role_id, permission_id)
);

-- 个人访问令牌表
CREATE TABLE IF NOT EXISTS registry_pat_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES registry_users(id) ON DELETE CASCADE,
    token_hash          VARCHAR(256) NOT NULL,
    name                VARCHAR(128) NOT NULL,
    scopes              TEXT, -- JSON数组
    expires_at          TIMESTAMP NOT NULL,
    last_used_at        TIMESTAMP,
    revoked_at          TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- 个人访问令牌归档表（用于自动删除前的逻辑备份）
CREATE TABLE IF NOT EXISTS registry_pat_tokens_archive (
    id                  UUID PRIMARY KEY,
    user_id             UUID NOT NULL,
    token_hash          VARCHAR(256) NOT NULL,
    name                VARCHAR(128) NOT NULL,
    scopes              TEXT,
    expires_at          TIMESTAMP NOT NULL,
    last_used_at        TIMESTAMP,
    revoked_at          TIMESTAMP,
    created_at          TIMESTAMP,
    updated_at          TIMESTAMP,
    deleted_at          TIMESTAMP,
    archived_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- 归档时间
);

CREATE INDEX IF NOT EXISTS idx_pat_tokens_archive_user ON registry_pat_tokens_archive(user_id);

-- 刷新令牌表
CREATE TABLE IF NOT EXISTS registry_refresh_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES registry_users(id) ON DELETE CASCADE,
    token               VARCHAR(256) NOT NULL UNIQUE,
    expires_at          TIMESTAMP NOT NULL,
    revoked_at          TIMESTAMP,
    user_agent          VARCHAR(512),
    ip                  VARCHAR(45),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- 镜像表
CREATE TABLE IF NOT EXISTS registry_images (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES registry_projects(id) ON DELETE CASCADE,
    name                VARCHAR(256) NOT NULL,
    description         TEXT,
    tags_count          INTEGER DEFAULT 0,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP,
    UNIQUE(project_id, name)
);

-- 镜像标签表
CREATE TABLE IF NOT EXISTS registry_image_tags (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id            UUID NOT NULL REFERENCES registry_images(id) ON DELETE CASCADE,
    name                VARCHAR(128) NOT NULL,
    digest              VARCHAR(256),
    manifest            TEXT,
    size                BIGINT DEFAULT 0,
    last_pull_at        TIMESTAMP,
    pull_count          INTEGER DEFAULT 0,
    scan_status         scan_status DEFAULT 'pending',
    scanned_at          TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP,
    UNIQUE(image_id, name)
);

-- 漏洞扫描结果表
CREATE TABLE IF NOT EXISTS registry_scan_results (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tag_id              UUID NOT NULL REFERENCES registry_image_tags(id) ON DELETE CASCADE,
    scanner             VARCHAR(64) DEFAULT 'trivy',
    severity            event_severity NOT NULL,
    vulnerability_id    VARCHAR(128),
    package_name        VARCHAR(256),
    installed_version   VARCHAR(128),
    fixed_version       VARCHAR(128),
    title               TEXT,
    description         TEXT,
    cvss_score          DECIMAL(3,2),
    cvss_vector         VARCHAR(256),
    status              VARCHAR(32) DEFAULT 'open',
    resolved_at         TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 审计日志表
CREATE TABLE IF NOT EXISTS registry_audit_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID REFERENCES registry_users(id),
    action              VARCHAR(64) NOT NULL,
    resource            VARCHAR(128),
    resource_id         UUID,
    ip                  VARCHAR(45),
    user_agent          VARCHAR(512),
    details             TEXT,
    status              VARCHAR(32),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- 审计日志归档表（用于自动删除前的逻辑备份）
CREATE TABLE IF NOT EXISTS registry_audit_logs_archive (
    id                  UUID PRIMARY KEY,
    user_id             UUID,
    action              VARCHAR(64) NOT NULL,
    resource            VARCHAR(128),
    resource_id         UUID,
    ip                  VARCHAR(45),
    user_agent          VARCHAR(512),
    details             TEXT,
    status              VARCHAR(32),
    created_at          TIMESTAMP,
    updated_at          TIMESTAMP,
    deleted_at          TIMESTAMP,
    archived_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- 归档时间
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_user ON registry_audit_logs_archive(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_created ON registry_audit_logs_archive(created_at);

-- 安全事件表
CREATE TABLE IF NOT EXISTS registry_security_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type          VARCHAR(64) NOT NULL,
    severity            event_severity NOT NULL,
    user_id             UUID REFERENCES registry_users(id),
    ip                  VARCHAR(45),
    details             TEXT,
    resolved_at         TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- Webhook配置表
CREATE TABLE IF NOT EXISTS registry_webhooks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID REFERENCES registry_projects(id) ON DELETE CASCADE,
    name                VARCHAR(128) NOT NULL,
    url                 VARCHAR(512) NOT NULL,
    secret              VARCHAR(256),
    events              TEXT, -- JSON数组
    enabled             BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- Webhook发送记录表
CREATE TABLE IF NOT EXISTS registry_webhook_deliveries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id          UUID NOT NULL REFERENCES registry_webhooks(id) ON DELETE CASCADE,
    event               VARCHAR(64) NOT NULL,
    request_payload     TEXT,
    response_status     INTEGER,
    response_body       TEXT,
    attempt_count       INTEGER DEFAULT 1,
    next_retry_at       TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 维护任务执行日志表（记录自动清理/自动删除等任务执行情况）
CREATE TABLE IF NOT EXISTS registry_maintenance_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_name            VARCHAR(128) NOT NULL,                      -- 任务名称，如 cleanup_old_audit_logs
    run_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- 执行时间
    affected_rows       BIGINT   NOT NULL DEFAULT 0,                -- 本次影响的行数
    status              VARCHAR(32) NOT NULL DEFAULT 'success',     -- 执行状态：success / failed
    message             TEXT                                        -- 附加信息或错误信息
);

CREATE INDEX IF NOT EXISTS idx_maintenance_logs_job_name ON registry_maintenance_logs(job_name);
CREATE INDEX IF NOT EXISTS idx_maintenance_logs_run_at ON registry_maintenance_logs(run_at);

-- ============================================
-- 4. 创建索引
-- ============================================

CREATE INDEX IF NOT EXISTS idx_users_username ON registry_users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON registry_users(email);
CREATE INDEX IF NOT EXISTS idx_projects_owner ON registry_projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_projects_is_public ON registry_projects(is_public) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_project_members_project ON registry_project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user ON registry_project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_pat_tokens_user ON registry_pat_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON registry_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_images_project ON registry_images(project_id);
CREATE INDEX IF NOT EXISTS idx_image_tags_image ON registry_image_tags(image_id);
CREATE INDEX IF NOT EXISTS idx_image_tags_digest ON registry_image_tags(digest);
CREATE INDEX IF NOT EXISTS idx_scan_results_tag ON registry_scan_results(tag_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON registry_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON registry_audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_security_events_type ON registry_security_events(event_type);
CREATE INDEX IF NOT EXISTS idx_security_events_severity ON registry_security_events(severity);
CREATE INDEX IF NOT EXISTS idx_webhooks_project ON registry_webhooks(project_id);

-- 全文搜索索引
CREATE INDEX IF NOT EXISTS idx_users_username_gin ON registry_users USING GIN (username gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_projects_name_gin ON registry_projects USING GIN (name gin_trgm_ops);

-- ============================================
-- 5. 插入初始数据
-- ============================================

-- 默认角色、权限、角色权限数据统一由 RBAC 服务在应用启动时初始化，
-- 这里不再插入初始数据，以避免与应用逻辑重复或不一致。

-- 赋权：确保业务账号 registry 拥有对 public schema 下对象的完全访问权限
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO registry;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO registry;

-- ============================================
-- 6. 创建视图
-- ============================================

-- 项目详细信息视图
CREATE OR REPLACE VIEW v_project_details AS
SELECT
    p.id,
    p.name,
    p.description,
    p.is_public,
    p.storage_used,
    p.storage_quota,
    p.image_count,
    p.created_at,
    u.username as owner_username,
    COUNT(DISTINCT pm.id) as member_count
FROM registry_projects p
JOIN registry_users u ON p.owner_id = u.id
LEFT JOIN registry_project_members pm ON p.id = pm.project_id AND pm.deleted_at IS NULL
WHERE p.deleted_at IS NULL
GROUP BY p.id, u.username;

-- ============================================
-- 7. 创建函数
-- ============================================

-- 自动更新updated_at的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 应用触发器
DO $$
DECLARE
    t text;
BEGIN
    -- 抑制 NOTICE 消息，避免生产环境日志噪音
    SET client_min_messages = WARNING;
    
    FOR t IN
        SELECT table_name FROM information_schema.columns
        WHERE column_name = 'updated_at'
        AND table_schema = 'public'
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS update_%I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER update_%I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()', t, t);
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 清理旧审计日志（90天前）
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs()
RETURNS void AS $$
DECLARE
    v_affected BIGINT := 0;
BEGIN
    -- 1. 将即将删除的数据归档到审计日志归档表，满足“自动删除前备份”规范
    INSERT INTO registry_audit_logs_archive (
        id, user_id, action, resource, resource_id, ip, user_agent,
        details, status, created_at, updated_at, deleted_at
    )
    SELECT
        id, user_id, action, resource, resource_id, ip, user_agent,
        details, status, created_at, updated_at, deleted_at
    FROM registry_audit_logs
    WHERE created_at < NOW() - INTERVAL '90 days';

    -- 2. 实际删除主表中的旧数据
    DELETE FROM registry_audit_logs WHERE created_at < NOW() - INTERVAL '90 days';
    GET DIAGNOSTICS v_affected = ROW_COUNT;

    INSERT INTO registry_maintenance_logs (job_name, affected_rows, status, message)
    VALUES ('cleanup_old_audit_logs', v_affected, 'success', NULL);
EXCEPTION
    WHEN OTHERS THEN
        INSERT INTO registry_maintenance_logs (job_name, affected_rows, status, message)
        VALUES ('cleanup_old_audit_logs', 0, 'failed', SQLERRM);
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 8. 创建定时任务（使用pg_cron或外部调度器）
-- ============================================

-- 示例：创建清理过期PAT的函数
CREATE OR REPLACE FUNCTION cleanup_expired_pat()
RETURNS void AS $$
DECLARE
    v_affected BIGINT := 0;
BEGIN
    -- 1. 将即将删除的 PAT 记录归档到归档表，满足“自动删除前备份”规范
    INSERT INTO registry_pat_tokens_archive (
        id, user_id, token_hash, name, scopes,
        expires_at, last_used_at, revoked_at,
        created_at, updated_at, deleted_at
    )
    SELECT
        id, user_id, token_hash, name, scopes,
        expires_at, last_used_at, revoked_at,
        created_at, updated_at, deleted_at
    FROM registry_pat_tokens
    WHERE expires_at < NOW() OR revoked_at IS NOT NULL;

    -- 2. 实际删除主表中的过期或已撤销 PAT
    DELETE FROM registry_pat_tokens WHERE expires_at < NOW() OR revoked_at IS NOT NULL;
    GET DIAGNOSTICS v_affected = ROW_COUNT;

    INSERT INTO registry_maintenance_logs (job_name, affected_rows, status, message)
    VALUES ('cleanup_expired_pat', v_affected, 'success', NULL);
EXCEPTION
    WHEN OTHERS THEN
        INSERT INTO registry_maintenance_logs (job_name, affected_rows, status, message)
        VALUES ('cleanup_expired_pat', 0, 'failed', SQLERRM);
END;
$$ LANGUAGE plpgsql;

COMMENT ON DATABASE registry_db IS 'CYP-Registry 容器镜像仓库数据库';

