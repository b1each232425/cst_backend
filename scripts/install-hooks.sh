#!/bin/bash

echo "Installing git hooks..."

# 获取项目根目录
project_root=$(git rev-parse --show-toplevel)

echo "Building commit message checker..."
go build -o .bin/check_commit_message ./check_commit_message.go

# 确保 .git/hooks 目录存在
mkdir -p "$project_root/.git/hooks"

mkdir -p "$project_root/scripts"

cat > "$project_root/.git/hooks/commit-msg" <<'EOF'
#!/bin/bash

# 获取提交信息文件路径
commit_msg_file=$1

# 检查可执行文件是否存在
if [ ! -f scripts/.bin/check_commit_message ]; then
    echo "Error: Commit message checker not found. Please run 'go build' first."
    exit 1
fi

# 调用 Go 程序并传递提交信息文件路径
scripts/.bin/check_commit_message "$commit_msg_file"
EOF

# 赋予可执行权限
chmod +x "$project_root/.git/hooks/commit-msg"

echo "Git hooks installed successfully!"