@echo off
setlocal
call "%~dp0Start VMS.cmd" -Stop
exit /b %errorlevel%
