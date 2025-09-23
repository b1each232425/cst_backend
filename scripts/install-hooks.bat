@echo off
echo Installing git commit-msg hooks...

for /f "delims=" %%i in ('git rev-parse --show-toplevel 2^>nul') do set "project_root=%%i"

echo Project root: %project_root%

echo Building commit message checker...
go build -o .bin\check_commit_message.exe .\check_commit_message.go

echo Creating commit-msg hook...
echo #!/bin/sh > "%project_root%/.git/hooks/commit-msg"
echo "%project_root%/scripts/.bin/check_commit_message.exe" $1 >> "%project_root%/.git/hooks/commit-msg"

echo Git hooks installed successfully!

pause