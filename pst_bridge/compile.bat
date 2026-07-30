@echo off
call "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvars64.bat" >nul 2>&1
cd /d "c:\Users\Administrator\Documents\trea\palworld-server-tool-main\palworld-server-tool-main\pst_bridge"
call build.bat
