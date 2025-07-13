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

export PATH="$GOPATH/bin:$GOROOT/bin:$NODE_HOME/bin:$PNPM_HOME:$BIN_PATH/bin:$BIN_PATH/docker:$PATH"

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

export containerName=i3l_p
export backEndFileName=i3l
export deployDst=$DEPLOY_PATH/i3l_p
export workspace=$PWD
# correct go build syntax
# go build -o kzz.io -ldflags \
#  "-X main.buildVer=r0.b_nonCI(2018-02-21_09:46:15) '-extldflags=-v -static'"

echo "build from source"

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

echo "stop ${containerName} container"
d stop ${containerName} > /dev/null 2> /dev/null || :
echo "  ${containerName} stoped"

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

echo "drop old ${containerName} container"
d container rm ${containerName} > /dev/null 2> /dev/null || :

appName="$backEndFileName"_r
echo "start new ${containerName} container"

d run --name=${containerName} \
 -d --restart=always \
 -p 6712:6712 \
 -v data:/var/data \
 -v deploy:/var/deploy \
 -e KAPP_NAME="$deployDst/$appName" \
 --network qnear \
 ubuntu "$deployDst/run.sh"

echo "deploy done"
