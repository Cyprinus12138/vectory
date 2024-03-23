package utils

import (
	"fmt"
	"google.golang.org/grpc"
	"os"
	"strings"
)

func ListProtoMethods(desc grpc.ServiceDesc) (result string) {
	var methods = make([]string, len(desc.Methods))
	for i, method := range desc.Methods {
		methods[i] = method.MethodName
	}
	return fmt.Sprintf("[%s]", strings.Join(methods, ","))
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
