-- V010: Add Omni-Modal Support Field to LLM Configs
-- 标识模型是否为全模态（支持音频/视频/图片输入）

ALTER TABLE llm_configs
    ADD COLUMN support_omni TINYINT DEFAULT 0
    COMMENT '是否为全模态模型（支持音频输入）: 0=否, 1=是'
    AFTER support_multimodal;
