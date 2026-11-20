@echo off
setlocal
powershell.exe -NoLogo -NoProfile -ExecutionPolicy RemoteSigned -File "%~dp0backend\ops\Start-VMS.ps1" %*
if errorlevel 1 (
    pause
    exit /b 1
)
