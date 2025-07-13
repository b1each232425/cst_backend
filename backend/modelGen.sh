#!/bin/zsh

dgw --schema="kuser" \
  --package="cmn" \
  --output cmn/models.go \
  --template=tmpl/struct.tmpl \
  --typemap=tmpl/typemap.toml \
  -x t_trial -x v_user -x c_mobile_region \
  postgres://kuser:ak47-Ever@localhost:16900/kdb\?sslmode=require
