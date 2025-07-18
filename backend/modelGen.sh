#!/bin/zsh

./dgw --schema="assessuser" \
  --package="cmn" \
  --output cmn/models.go \
  --template=tmpl/struct.tmpl \
  --typemap=tmpl/typemap.toml \
  -x t_trial -x v_user -x c_mobile_region \
  postgres://assessuser:as142857@w2w.io:6900/assessdb\?sslmode=require
