#!/bin/bash
# docker run can't direct "run" shell script

# rm -rf /tmp/localtime
# cp -fvLT --remove-destination "/var/data/kUser/zoneinfo/Asia/Shanghai" "/etc/_localtime_"
cp -fvLT --remove-destination "/var/data/kUser/zoneinfo/Asia/Shanghai" "/etc/localtime"

service ssh restart

"$KAPP_NAME" serve
