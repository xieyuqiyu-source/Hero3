@echo off
chcp 65001 >nul 2>&1
title Hero3 Dev Environment

REM Hero3 一键启动脚本 (Windows) - 同时运行 Go 后端、玩家前端和 GM 后台

set ROOT_DIR=%~dp0

REM 加载 .env 文件
if exist "%ROOT_DIR%go\.env" (
    echo 加载 .env 配置...
    for /f "usebackq tokens=1,* delims==" %%A in ("%ROOT_DIR%go\.env") do (
        REM 跳过注释行和空行
        echo %%A | findstr /r "^#" >nul 2>&1 || (
            if not "%%A"=="" set "%%A=%%B"
        )
    )
)

echo.
echo ===== Hero3 开发环境启动 =====
echo.

REM SSH 隧道 (Windows 下用 ssh 命令，需要 OpenSSH 客户端)
if "%HERO3_DB_TUNNEL_ENABLED%"=="true" (
    echo 启动 SSH 数据库隧道...
    if not defined HERO3_DB_TUNNEL_LOCAL_PORT set HERO3_DB_TUNNEL_LOCAL_PORT=3307
    if not defined HERO3_DB_REMOTE_PORT set HERO3_DB_REMOTE_PORT=3306
    if not defined HERO3_DB_SSH_USER set HERO3_DB_SSH_USER=root
    start "DB-Tunnel" /min cmd /c "ssh -o StrictHostKeyChecking=accept-new -N -L %HERO3_DB_TUNNEL_LOCAL_PORT%:127.0.0.1:%HERO3_DB_REMOTE_PORT% %HERO3_DB_SSH_USER%@%HERO3_DB_SSH_HOST%"
    timeout /t 2 /nobreak >nul
)

REM 启动 Go 后端
echo [1/3] 启动 Go 后端...
start "Hero3-Go" cmd /c "cd /d "%ROOT_DIR%go" && go run ./cmd/server"

REM 启动 Web 前端
echo [2/3] 启动 Web 前端...
start "Hero3-Web" cmd /c "cd /d "%ROOT_DIR%web" && pnpm dev"

REM 启动 GM 后台
echo [3/3] 启动 GM 后台...
start "Hero3-Admin" cmd /c "cd /d "%ROOT_DIR%admin" && pnpm dev"

echo.
echo ===== Hero3 开发环境已启动 =====
echo   前端: http://localhost:5173
echo   后台: http://localhost:5174
echo   后端: http://localhost:8080
echo.
echo 关闭此窗口或按任意键停止所有服务...
pause >nul

REM 停止所有服务
echo.
echo 正在停止所有服务...
taskkill /fi "WINDOWTITLE eq Hero3-Go" /t /f >nul 2>&1
taskkill /fi "WINDOWTITLE eq Hero3-Web" /t /f >nul 2>&1
taskkill /fi "WINDOWTITLE eq Hero3-Admin" /t /f >nul 2>&1
taskkill /fi "WINDOWTITLE eq DB-Tunnel" /t /f >nul 2>&1
echo 已停止。
