echo "Installing protoc plugins"
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
export PATH="$PATH:$(go env GOPATH)/bin"

echo "Generating pb"
protoc --go_out=proto   \
    --go-grpc_out=proto  \
    proto/vectory/*.proto