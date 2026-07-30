@echo off
:: Build pst_bridge.dll using Visual Studio (x64 Release)
:: Run this from the pst_bridge directory

where cl >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] cl.exe not found. Run this from a VS Developer Command Prompt.
    echo   e.g. open "x64 Native Tools Command Prompt for VS 2022"
    pause
    exit /b 1
)

if not exist build mkdir build

echo [BUILD] Compiling pst_bridge.dll ...

cl.exe ^
    /LD ^
    /O2 ^
    /GS- ^
    /EHa ^
    /std:c++17 ^
    /nologo ^
    /W3 ^
    /Fe:build\pst_bridge.dll ^
    /Fo:build\ ^
    src\dllmain.cpp ^
    kernel32.lib ^
    /link /DLL /SUBSYSTEM:WINDOWS /MACHINE:X64

if %errorlevel% equ 0 (
    echo [OK] build\pst_bridge.dll compiled successfully.
    if not exist dist mkdir dist
    copy /Y build\pst_bridge.dll dist\pst_bridge.dll >nul
    echo [OK] Copied to dist\pst_bridge.dll
) else (
    echo [FAIL] Compilation failed.
    exit /b 1
)
