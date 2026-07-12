"""
邮件服务 - SMTP 邮件发送

提供:
- 邮箱验证邮件
- 密码重置邮件
- 欢迎邮件
"""

import asyncio
import logging
import smtplib
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from typing import Optional

from ..core.config import settings

logger = logging.getLogger(__name__)


class EmailService:
    """邮件服务"""

    def __init__(self):
        self.host = settings.smtp_host
        self.port = settings.smtp_port
        self.user = settings.smtp_user
        self.password = settings.smtp_password
        self.from_email = settings.smtp_from_email or settings.smtp_user
        self.from_name = settings.smtp_from_name
        self.use_tls = settings.smtp_use_tls
        self.frontend_url = settings.frontend_url

    def is_configured(self) -> bool:
        """检查是否已配置"""
        return bool(self.host and self.user and self.password)

    def _create_message(
        self,
        to_email: str,
        subject: str,
        html_content: str,
        text_content: Optional[str] = None,
    ) -> MIMEMultipart:
        """创建邮件消息"""
        msg = MIMEMultipart("alternative")
        msg["Subject"] = subject
        msg["From"] = f"{self.from_name} <{self.from_email}>"
        msg["To"] = to_email

        # 纯文本版本
        if text_content:
            msg.attach(MIMEText(text_content, "plain", "utf-8"))

        # HTML 版本
        msg.attach(MIMEText(html_content, "html", "utf-8"))

        return msg

    def _send_sync(
        self, to_email: str, subject: str, html_content: str, text_content: str = None
    ) -> bool:
        """同步发送邮件"""
        if not self.is_configured():
            logger.warning("SMTP not configured, email not sent")
            return False

        try:
            msg = self._create_message(to_email, subject, html_content, text_content)

            with smtplib.SMTP(self.host, self.port) as server:
                if self.use_tls:
                    server.starttls()
                server.login(self.user, self.password)
                server.send_message(msg)

            logger.info(f"Email sent to {to_email}: {subject}")
            return True

        except Exception as e:
            logger.error(f"Failed to send email to {to_email}: {e}")
            return False

    async def send(
        self, to_email: str, subject: str, html_content: str, text_content: str = None
    ) -> bool:
        """异步发送邮件"""
        return await asyncio.to_thread(
            self._send_sync, to_email, subject, html_content, text_content
        )

    async def send_verification_email(
        self, to_email: str, token: str, display_name: str = None
    ) -> bool:
        """发送邮箱验证邮件"""
        verification_url = f"{self.frontend_url}/auth/verify-email?token={token}"
        name = display_name or to_email.split("@")[0]

        subject = "验证您的邮箱 - PicTactic Agent"

        html_content = f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0; font-size: 24px;">PicTactic Agent</h1>
    </div>

    <div style="background: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <h2 style="color: #333; margin-top: 0;">您好，{name}！</h2>

        <p>感谢您注册 PicTactic Agent。请点击下方按钮验证您的邮箱地址：</p>

        <div style="text-align: center; margin: 30px 0;">
            <a href="{verification_url}"
               style="display: inline-block; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; font-size: 16px;">
                验证邮箱
            </a>
        </div>

        <p style="color: #666; font-size: 14px;">如果按钮无法点击，请复制以下链接到浏览器：</p>
        <p style="background: #f5f5f5; padding: 12px; border-radius: 6px; word-break: break-all; font-size: 12px; color: #666;">
            {verification_url}
        </p>

        <p style="color: #999; font-size: 12px; margin-top: 30px;">
            此链接 24 小时内有效。如果您没有注册账号，请忽略此邮件。
        </p>
    </div>

    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>PicTactic Agent - AI 图像生成助手</p>
    </div>
</body>
</html>
"""

        text_content = f"""
您好，{name}！

感谢您注册 PicTactic Agent。请点击以下链接验证您的邮箱地址：

{verification_url}

此链接 24 小时内有效。如果您没有注册账号，请忽略此邮件。

---
PicTactic Agent
"""

        return await self.send(to_email, subject, html_content, text_content)

    async def send_password_reset_email(
        self, to_email: str, token: str, display_name: str = None
    ) -> bool:
        """发送密码重置邮件"""
        reset_url = f"{self.frontend_url}/auth/reset-password?token={token}"
        name = display_name or to_email.split("@")[0]

        subject = "重置您的密码 - PicTactic Agent"

        html_content = f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0; font-size: 24px;">PicTactic Agent</h1>
    </div>

    <div style="background: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <h2 style="color: #333; margin-top: 0;">您好，{name}！</h2>

        <p>我们收到了重置您账号密码的请求。请点击下方按钮设置新密码：</p>

        <div style="text-align: center; margin: 30px 0;">
            <a href="{reset_url}"
               style="display: inline-block; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; font-size: 16px;">
                重置密码
            </a>
        </div>

        <p style="color: #666; font-size: 14px;">如果按钮无法点击，请复制以下链接到浏览器：</p>
        <p style="background: #f5f5f5; padding: 12px; border-radius: 6px; word-break: break-all; font-size: 12px; color: #666;">
            {reset_url}
        </p>

        <div style="background: #fff3cd; border: 1px solid #ffc107; border-radius: 6px; padding: 12px; margin-top: 20px;">
            <p style="color: #856404; font-size: 13px; margin: 0;">
                <strong>安全提示：</strong>如果您没有请求重置密码，请忽略此邮件。您的账号密码不会被更改。
            </p>
        </div>

        <p style="color: #999; font-size: 12px; margin-top: 30px;">
            此链接 1 小时内有效。
        </p>
    </div>

    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>PicTactic Agent - AI 图像生成助手</p>
    </div>
</body>
</html>
"""

        text_content = f"""
您好，{name}！

我们收到了重置您账号密码的请求。请点击以下链接设置新密码：

{reset_url}

此链接 1 小时内有效。

安全提示：如果您没有请求重置密码，请忽略此邮件。您的账号密码不会被更改。

---
PicTactic Agent
"""

        return await self.send(to_email, subject, html_content, text_content)

    async def send_welcome_email(
        self, to_email: str, display_name: str = None
    ) -> bool:
        """发送欢迎邮件"""
        name = display_name or to_email.split("@")[0]
        login_url = f"{self.frontend_url}/auth/login"

        subject = "欢迎加入 PicTactic Agent！"

        html_content = f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0; font-size: 24px;">欢迎加入！</h1>
    </div>

    <div style="background: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <h2 style="color: #333; margin-top: 0;">您好，{name}！</h2>

        <p>感谢您加入 PicTactic Agent！您的账号已成功激活。</p>

        <p>现在您可以：</p>
        <ul style="color: #555;">
            <li>使用 AI 生成高质量封面图</li>
            <li>通过多轮迭代优化生成效果</li>
            <li>支持模板图片参考生成</li>
        </ul>

        <div style="text-align: center; margin: 30px 0;">
            <a href="{login_url}"
               style="display: inline-block; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; font-size: 16px;">
                开始使用
            </a>
        </div>

        <p style="color: #666; font-size: 14px;">如有任何问题，欢迎随时联系我们。</p>
    </div>

    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>PicTactic Agent - AI 图像生成助手</p>
    </div>
</body>
</html>
"""

        text_content = f"""
您好，{name}！

感谢您加入 PicTactic Agent！您的账号已成功激活。

现在您可以：
- 使用 AI 生成高质量封面图
- 通过多轮迭代优化生成效果
- 支持模板图片参考生成

立即开始使用：{login_url}

如有任何问题，欢迎随时联系我们。

---
PicTactic Agent
"""

        return await self.send(to_email, subject, html_content, text_content)


# 单例
email_service = EmailService()
