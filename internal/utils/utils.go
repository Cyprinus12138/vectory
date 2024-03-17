package utils

import (
	"fmt"
	"google.golang.org/grpc"
	"strings"
)

func ListProtoMethods(desc grpc.ServiceDesc) (result string) {
	var methods = make([]string, len(desc.Methods))
	for i, method := range desc.Methods {
		methods[i] = method.MethodName
	}
	return fmt.Sprintf("[%s]", strings.Join(methods, ","))
}
