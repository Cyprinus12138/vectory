# TODO install protoc

go mod download || echo "go mod download failed"
mkdir "bin/server" || echo "directory already exists"
echo "Build Go"
go build -v -a -o  bin/server/ cmd/server/main.go
