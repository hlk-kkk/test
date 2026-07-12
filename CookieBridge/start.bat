@echo off
chcp 65001 >nul 2>&1
setlocal

set "SCRIPT_DIR=%~dp0"
set "SERVER_DIR=%SCRIPT_DIR%server"

echo ==========================================
echo   CookieBridge Server 一键启动
echo ==========================================

:: 检查 uv 是否已安装
where uv >nul 2>&1
if %errorlevel% neq 0 (
    echo [1/3] 未检测到 uv，正在安装...
    powershell -ExecutionPolicy ByPass -NoProfile -Command "irm https://astral.sh/uv/install.ps1 | iex"
    :: 刷新当前会话的 PATH
    set "PATH=%USERPROFILE%\.local\bin;%PATH%"
    echo [1/3] uv 安装完成
) else (
    echo [1/3] uv 已安装
)

:: 安装依赖
echo [2/3] 安装项目依赖...
cd /d "%SERVER_DIR%"
uv sync

:: 启动服务
echo [3/3] 启动 CookieBridge Server...
if defined APP_PORT (
    echo   地址: http://localhost:%APP_PORT%
) else (
    echo   地址: http://localhost:8274
)
echo   按 Ctrl+C 停止服务
echo ==========================================
uv run app.py

endlocal
