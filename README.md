# Development Quick Start Vectory

## Before Start
- Make sure you have protoc installed already.
- Setup the go environment variable `GOPRIVATE="github.com/Cyprinus12138/*"`
- Rename the package name in the go.mod with your service name.
- Update the `ServiceName` const under the directory pkg/pkg.go with your service name.
- Replace the `proto/mq/example`.proto with your service name dot `proto`, and run `make gen` command under the project root directory.
- Update the `service.go` under the directory `internal/grpc_handler`: make sure the class name of the proto server, the register method name and the Desc variable name are identical with those generated in the pb file.
- Update the tag name in the `build_image.sh`.
- Update the k8s manifest files under deploy directory, so that we can deploy our service quite conveniently via github actions. 

## Notes
- Recommended protoc version: libprotoc 3.6.1 (used to build this template)
- Based on the grpc design standard, we shouldn't use any generic proto pkg name. `We recommend that every .proto file have a package name that is deliberately chosen to be universally unique (for example, prefixed with the name of a company).` [namespace-conflict](https://protobuf.dev/reference/go/faq/#namespace-conflict)
- Make sure adding a `GIT_TOKEN` secret with read-write permission to the dependency internal/private repo, if any.
