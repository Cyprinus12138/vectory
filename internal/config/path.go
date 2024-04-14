package config

import (
	"fmt"
)

const (
	DefaultIndexPath = "./etc/index"
)

func GetRootPath() string {
	return GetEnvEtCDRoot()
}

func GetNodeMetaPathPrefix() string {
	return fmt.Sprintf("%s/node/meta", GetRootPath())
}

func GetNodeMetaPath(id string) string {
	return fmt.Sprintf("%s/%s", GetNodeMetaPathPrefix(), id)
}

func GetNodeLoadPath(id string) string {
	return fmt.Sprintf("%s/node/load/%s", GetRootPath(), id)
}

func GetLocalIndexRootPath() string {
	return GetEnvLocalIndexPath()
}

func GetLocalIndexPath(fileName string) string {
	return fmt.Sprintf("%s/%s", GetLocalIndexRootPath(), fileName)
}

func GetIndexManifestPathPrefix() string {
	return fmt.Sprintf("%s/index", GetRootPath())
}
