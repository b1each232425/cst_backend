protoc \
  --go_opt=paths=source_relative \
  --go_out=. \
  --go-grpc_opt=paths=source_relative \
  --go-grpc_out=. \
  --grpc_out=. \
  --plugin=protoc-gen-grpc=$(which grpc_cpp_plugin) \
  --cpp_out=. \
  w2wproto/w2wservice.proto

rm -f w2wproto/{*.h,*.cc}
