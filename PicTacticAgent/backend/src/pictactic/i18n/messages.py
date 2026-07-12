"""All translatable backend messages."""

MESSAGES: dict[str, dict[str, str]] = {
    "en": {
        # API responses - generation
        "task.created": "Task created, generation starting...",
        "task.not_found": "Task not found: {task_id}",
        "task.cancelled": "Task cancelled",
        "task.not_running": "Task was not running",

        # Status messages
        "status.pending": "Task queued, waiting to start...",
        "status.enhancing": "Analyzing templates and enhancing prompt...",
        "status.generating": "Generating images (Round {round_num})...",
        "status.reviewing": "Evaluating images (Best score: {best:.2f})...",
        "status.completed": "Completed! Best score: {best:.2f}",
        "status.failed": "Generation failed: {error}",
        "status.failed_generic": "Generation failed",
        "status.cancelled": "Task cancelled",
        "status.unknown": "Unknown status",

        # Health
        "health.status": "healthy",
        "health.name": "PicTacticAgent API",

        # WebSocket
        "ws.task_not_found": "Task not found",

        # Provider errors
        "provider.gemini.server_error": "Gemini API server error (500). This may be temporary, please retry later",
        "provider.gemini.unavailable": "Gemini API service temporarily unavailable (503), please retry later",
        "provider.gemini.auth_failed": "Gemini API authentication failed (401), please check API Key configuration",
        "provider.gemini.rate_limited": "Gemini API rate limit exceeded (429), please retry later",
        "provider.gemini.client_error": "Gemini API request error ({status_code})",
        "provider.gemini.generic_error": "Gemini API error ({status_code})",
        "provider.gemini.no_images": "No images returned from Gemini API",
        "provider.gemini.network_error": "Network error calling Gemini API after {retries} retries: {error}",
        "provider.gemini.image_load_error": "Failed to load reference image: {error}",
        "provider.gemini.generation_error": "Image generation failed: {error}",

        # Gemini Web provider errors
        "provider.gemini_web.no_cookie": "Gemini Web cookie (__Secure-1PSID) is not configured. Set GEMINI_WEB_SECURE_1PSID in .env",

        # Evaluator
        "evaluator.parse_failed": "Evaluation response parsing failed: {error}",
        "evaluator.no_json": "No JSON found in response",
        "evaluator.enhanced_failed": "Enhanced evaluation failed: {error}",
        "evaluator.not_specified": "Not specified",

        # Evaluator - image labels
        "evaluator.image_to_review": "Image to review:",
        "evaluator.reference_image": "User's original photo (for comparison):",

        # Video status messages
        "video.status.pending": "Video task queued, waiting to start...",
        "video.status.enhancing": "Enhancing video prompt...",
        "video.status.generating": "Generating videos (Round {round_num})...",
        "video.status.polling": "Waiting for video generation... {progress}%",
        "video.status.completed": "Completed! {total} video(s) generated",
        "video.status.failed": "Video generation failed: {error}",
        "video.status.failed_generic": "Video generation failed",
        "video.status.cancelled": "Video task cancelled",

        # Defect labels
        "defect.chinese_garbled": "Chinese text garbled",
        "defect.identity_mismatch": "Identity unrecognizable",
        "defect.face_distortion": "Face distortion",

        # Auth
        "auth.email_not_whitelisted": "This email is not authorized to access the service. Please contact the administrator.",
    },
    "zh": {
        # API responses - generation
        "task.created": "任务已创建，正在开始生成...",
        "task.not_found": "任务未找到: {task_id}",
        "task.cancelled": "任务已取消",
        "task.not_running": "任务未在运行",

        # Status messages
        "status.pending": "任务排队中，等待开始...",
        "status.enhancing": "正在分析模板并优化提示词...",
        "status.generating": "正在生成图片（第 {round_num} 轮）...",
        "status.reviewing": "正在评估图片（最佳分数: {best:.2f}）...",
        "status.completed": "完成！最佳分数: {best:.2f}",
        "status.failed": "生成失败: {error}",
        "status.failed_generic": "生成失败",
        "status.cancelled": "任务已取消",
        "status.unknown": "未知状态",

        # Health
        "health.status": "healthy",
        "health.name": "PicTacticAgent API",

        # WebSocket
        "ws.task_not_found": "任务未找到",

        # Provider errors
        "provider.gemini.server_error": "Gemini API 服务器错误 (500)。这可能是临时性问题，请稍后重试",
        "provider.gemini.unavailable": "Gemini API 服务暂时不可用 (503)，请稍后重试",
        "provider.gemini.auth_failed": "Gemini API 认证失败 (401)，请检查 API Key 配置",
        "provider.gemini.rate_limited": "Gemini API 请求频率超限 (429)，请稍后重试",
        "provider.gemini.client_error": "Gemini API 请求错误 ({status_code})",
        "provider.gemini.generic_error": "Gemini API 错误 ({status_code})",
        "provider.gemini.no_images": "Gemini API 未返回图片",
        "provider.gemini.network_error": "调用 Gemini API 网络错误，重试 {retries} 次后失败: {error}",
        "provider.gemini.image_load_error": "加载参考图片失败: {error}",
        "provider.gemini.generation_error": "图片生成失败: {error}",

        # Gemini Web provider errors
        "provider.gemini_web.no_cookie": "Gemini Web Cookie (__Secure-1PSID) 未配置，请在 .env 中设置 GEMINI_WEB_SECURE_1PSID",

        # Evaluator
        "evaluator.parse_failed": "评估响应解析失败: {error}",
        "evaluator.no_json": "响应中未找到 JSON",
        "evaluator.enhanced_failed": "增强评估失败: {error}",
        "evaluator.not_specified": "未指定",

        # Evaluator - image labels
        "evaluator.image_to_review": "待评审的图片：",
        "evaluator.reference_image": "用户原始人像（用于对比）：",

        # Video status messages
        "video.status.pending": "视频任务排队中，等待开始...",
        "video.status.enhancing": "正在优化视频提示词...",
        "video.status.generating": "正在生成视频（第 {round_num} 轮）...",
        "video.status.polling": "等待视频生成中... {progress}%",
        "video.status.completed": "完成！共生成 {total} 个视频",
        "video.status.failed": "视频生成失败: {error}",
        "video.status.failed_generic": "视频生成失败",
        "video.status.cancelled": "视频任务已取消",

        # Defect labels
        "defect.chinese_garbled": "中文文字乱码",
        "defect.identity_mismatch": "人像无法识别",
        "defect.face_distortion": "人脸扭曲变形",

        # Auth
        "auth.email_not_whitelisted": "该邮箱未被授权使用此服务，请联系管理员。",
    },
}
