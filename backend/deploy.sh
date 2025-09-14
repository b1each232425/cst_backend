#!/bin/sh
DATA_PATH=/var/data
BIN_PATH=/var/data/kApps
DEPLOY_PATH=/var/deploy


export GOROOT=$BIN_PATH/goLang
export GOPATH=$DATA_PATH/kUser/goUser
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn

export NODE_HOME=$BIN_PATH/node
export PNPM_HOME=/var/data/pnpm

mkdir -p $PNPM_HOME

export PATH="$BIN_PATH/pgsql/bin:$GOPATH/bin:$GOROOT/bin:$NODE_HOME/bin:$PNPM_HOME:$BIN_PATH/bin:$BIN_PATH/docker:$PATH"

if [ -z "$(which pnpm)" ];then
  npm install -g pnpm
fi

pnpm config set store-dir "$PNPM_HOME"
pnpm config set registry https://registry.npmmirror.com

# -----------------
set -e

if [ -z "$(which stringer)" ]; then
  echo "install stringer ..."
  go install golang.org/x/tools/cmd/stringer@latest
  echo "stringer installed"
fi

if [ -z "$(which mockgen)" ]; then
  echo "install mockgen ..."
  go install github.com/golang/mock/mockgen@v1.6.0
  echo "mockgen installed"
fi

export backEndFileName=$1
if [ -z "$backEndFileName" ] ;then
  export backEndFileName=devmentor
fi
export app_port=$2
if [ -z "$app_port" ] ;then
  export app_port=6610
fi
export ssh_port=$3
if [ -z "$ssh_port" ] ;then
  export ssh_port=6222
fi
export deployDst=$DEPLOY_PATH/$backEndFileName
export workspace=$PWD
# correct go build syntax
# go build -o kzz.io -ldflags \
#  "-X main.buildVer=r0.b_nonCI(2018-02-21_09:46:15) '-extldflags=-v -static'"

echo "====== build from source ======"
echo "       workspace: $workspace"
echo "       deployDst: $deployDst"
echo " backEndFileName: $backEndFileName"
echo "        app_port: $app_port"
echo "        ssh_port: $ssh_port"
echo "==============================="

# export GO111MODULE=off
# ./b.sh
readonly BE_VER=$(git rev-parse HEAD)_$(date '+%Y-%m-%dT%H:%M:%S')

readonly stringer=$(which stringer)
if [ -z "$stringer" ] ;then
  echo "install stringer"
  go install golang.org/x/tools/cmd/stringer@latest
fi

go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB="sum.golang.google.cn"


chmod +x api-enroll.sh
./api-enroll.sh

go mod tidy

go build \
  -o $backEndFileName \
  -ldflags "-X main.buildVer=${BE_VER} '-extldflags=-v -static'"


echo "  build complete"

echo "stop $backEndFileName container"
d stop $backEndFileName > /dev/null 2> /dev/null || :
echo "  $backEndFileName stopped"

echo "publishing artifact for remote"

mkdir -p $deployDst/f
mkdir -p $deployDst/admin-fe

rm -f "$backEndFileName"_r
mv $backEndFileName "$backEndFileName"_r
chmod +x "$backEndFileName"_r
chmod +x run.sh

if [ ! -f "${deployDst}/.config_linux.json" ] ;then
  cp .config_linux_sample.json "${deployDst}/.config_linux.json"
  echo generate ${deployDst}/.config_linux.json
else
  echo ${deployDst}/.config_linux.json exists already
fi

rsync -cruzEL run.sh \
  "$backEndFileName"_r \
  Shanghai \
  $deployDst/

echo "  artifact published"

echo "drop old $backEndFileName container"
d container rm $backEndFileName > /dev/null 2> /dev/null || :

appName="$backEndFileName"_r
echo "start new $backEndFileName container"

d run --name=$backEndFileName \
 -d --restart=always \
 -p $app_port:6610 \
 -p $ssh_port:22 \
 -v data:/var/data \
 -v deploy:/var/deploy \
 -v assess_ssh:/etc/ssh \
 -w $deployDst \
 -e KAPP_NAME="$deployDst/$appName" \
 -e PATH="$PATH" \
 --network qnear \
 ubuntu:ci "$deployDst/run.sh"

echo "deploy done"
