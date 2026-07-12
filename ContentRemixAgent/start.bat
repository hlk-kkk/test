@echo off
:: Copyright (c) 2026 relakkes@gmail.com
::
:: This file is part of MediaCrawlerPro-ContentRemixAgent project.
:: Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/start.bat
:: GitHub: https://github.com/NanmiCoder
:: Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
::
:: 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
:: 1. 不得用于任何商业用途。
:: 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
:: 3. 不得进行大规模爬取或对平台造成运营干扰。
:: 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
:: 5. 不得用于任何非法或不当的用途。
::
:: 详细许可条款请参阅项目根目录下的LICENSE文件。
:: 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

chcp 65001 >nul 2>&1
title Content Remix Agent

cd /d "%~dp0"

:: 检查依赖
where uv >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未找到 uv，请先安装: https://docs.astral.sh/uv/
    pause
    exit /b 1
)
where npm >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未找到 npm，请先安装 Node.js
    pause
    exit /b 1
)

echo ========================================
echo   Content Remix Agent 开发服务器
echo ========================================
echo   [!] 本项目仅供学习和研究目的使用
echo   Licensed under NON-COMMERCIAL LICENSE 1.1
echo ----------------------------------------
echo   后端: http://localhost:8001
echo   前端: http://localhost:5373
echo ========================================
echo.

:: 启动后端
echo [后端] 安装依赖并启动...
cd backend
start /b cmd /c "uv sync && uv run python scripts/run_api_server.py"
cd ..

:: 启动前端
echo [前端] 安装依赖并启动...
cd frontend
start /b cmd /c "npm install --silent && npm run dev"
cd ..

echo.
echo 按任意键停止所有服务...
pause >nul

:: 关闭服务
taskkill /f /im node.exe >nul 2>&1
taskkill /f /im python.exe >nul 2>&1
echo 所有服务已关闭
