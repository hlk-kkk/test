-- ============================================================================
-- V009: Add llm_config_name to agent_sessions
-- ============================================================================
-- 支持每会话独立模型选择，存储会话使用的 LLM 配置名称。
-- NULL 表示使用全局激活配置，不加外键约束以支持配置删除后优雅降级。
-- ============================================================================

ALTER TABLE agent_sessions
  ADD COLUMN llm_config_name VARCHAR(50) DEFAULT NULL
  COMMENT 'LLM config name used for this session';
