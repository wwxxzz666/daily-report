@echo off
chcp 65001 >nul
set PATH=D:\anaconda3\pkgs\go-1.26.3-h5588389_0\go\bin;C:\Users\37119\go\bin;%PATH%
set GOROOT=D:\anaconda3\pkgs\go-1.26.3-h5588389_0\go
set GOPROXY=https://goproxy.cn,direct
cd /D "d:\study\日报助手"
wails build
echo.
echo EXIT CODE: %ERRORLEVEL%
pause
