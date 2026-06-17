@echo off
REM Build the clipd binary for Windows
REM
REM Usage:
REM   build-windows.bat           # builds binary into .\build\bin\clipd.exe
REM
REM Prereqs:
REM   - Go 1.22+
REM   - Wails v2 CLI (go install github.com/wailsapp/wails/v2/cmd/wails@latest)
REM   - npm or yarn (for frontend)

setlocal enabledelayedexpansion

cd /d "%~dp0\.."

set PATH=C:\Program Files\Go\bin;%USERPROFILE%\go\bin;%PATH%
set GOCACHE=%TEMP%\clipd-go-cache
set GOMODCACHE=%TEMP%\clipd-go-mod

echo.
echo "==> Building clipd binary for Windows"
call wails build -clean -ldflags="-s -w"

if exist "build\bin\clipd.exe" (
    echo.
    echo "==> Binary built successfully:"
    for %%F in (build\bin\clipd.exe) do echo     %%~sF  %%F
    echo.
    echo "==> Done."
) else (
    echo.
    echo ERROR: Build failed - clipd.exe not found
    exit /b 1
)

endlocal
