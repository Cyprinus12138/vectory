package config

import (
	"fmt"
	"github.com/Cyprinus12138/vectory/pkg"
)

func GetRootPath() string {
	return fmt.Sprintf("/vectory/%s", pkg.ClusterName)
}

func GetNodeMetaPath(id string) string {
	return fmt.Sprintf("%s/node/meta/%s", GetRootPath(), id)
}

func GetNodeLoadPath(id string) string {
	return fmt.Sprintf("%s/node/load/%s", GetRootPath(), id)
}

func GetIndexMetaPath(indexName string) string {
	return fmt.Sprintf("%s/index/%s", GetRootPath(), indexName)
}
