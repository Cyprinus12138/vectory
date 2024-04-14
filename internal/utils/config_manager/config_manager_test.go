package config_manager

import (
	"github.com/spf13/viper"
	"reflect"
	"testing"
)

const cfgPath = "../../../tests/config.yml"

type ClusterMetaConfig struct {
	ClusterName    string `json:"cluster_name" yaml:"cluster_name"`
	GrpcEnabled    bool   `json:"grpc_enabled" yaml:"grpc_enabled"`
	IntTypeContent int    `json:"int_type_content" yaml:"int_type_content"`
}

func init() {
	err := Init(cfgPath, []viper.RegisteredConfig{
		{
			Key:      "cluster",
			CanBeNil: false,
			Schema:   &ClusterMetaConfig{},
		},
		{
			Key:      "env",
			CanBeNil: true,
		},
	})
	if err != nil {
		panic(err)
	}
}

func TestGet(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "env",
			args: struct{ key string }{key: "env"},
			want: map[string]interface{}{
				"port":     6060,
				"rpc_port": 8999,
				"pod_ip":   "127.0.0.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Get(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "env.port",
			args: struct{ key string }{key: "env.port"},
			want: "6060",
		},
		{
			name: "env.port.not_found",
			args: struct{ key string }{key: "env.port.not_found"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetString(tt.args.key); got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet(t *testing.T) {
	type args struct {
		key string
		val interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "set",
			args: struct {
				key string
				val interface{}
			}{key: "set", val: "set already"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Set(tt.args.key, tt.args.val)
			if got := Get(tt.args.key); !reflect.DeepEqual(got, tt.args.val) {
				t.Errorf("Get() = %v, want %v", got, tt.args.val)
			}
		})
	}
}

func TestInit(t *testing.T) {
	type args struct {
		cfgPath      string
		configSchema []viper.RegisteredConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: struct {
				cfgPath      string
				configSchema []viper.RegisteredConfig
			}{cfgPath: cfgPath, configSchema: []viper.RegisteredConfig{
				{
					Key:      "cluster",
					CanBeNil: false,
					Schema:   &ClusterMetaConfig{},
				},
				{
					Key:      "env",
					CanBeNil: true,
				},
			}},
			wantErr: false,
		},
		{
			name: "miss required",
			args: struct {
				cfgPath      string
				configSchema []viper.RegisteredConfig
			}{cfgPath: cfgPath, configSchema: []viper.RegisteredConfig{
				{
					Key:      "cluster.not_found",
					CanBeNil: false,
					Schema:   &ClusterMetaConfig{},
				},
				{
					Key:      "env",
					CanBeNil: true,
				},
			}},
			wantErr: true,
		},
		{
			name: "wrong type",
			args: struct {
				cfgPath      string
				configSchema []viper.RegisteredConfig
			}{cfgPath: cfgPath, configSchema: []viper.RegisteredConfig{
				{
					Key:      "cluster_invalid_type",
					CanBeNil: false,
					Schema:   &ClusterMetaConfig{},
				},
				{
					Key:      "env",
					CanBeNil: true,
				},
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Init(tt.args.cfgPath, tt.args.configSchema); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				t.Logf("Got expected error %v", err)
			}
		})
	}
}
