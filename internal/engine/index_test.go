package engine

import (
	"context"
	"reflect"
	"testing"

	"github.com/Cyprinus12138/vectory/internal/config"
)

func TestShard_ShardKey(t *testing.T) {
	tests := []struct {
		name  string
		shard Shard
		want  string
	}{
		{
			name:  "basic",
			shard: Shard{IndexName: "idx", ShardId: 0, ReplicaId: 0},
			want:  "idx:0:0",
		},
		{
			name:  "with ids",
			shard: Shard{IndexName: "my_index", ShardId: 3, ReplicaId: 2},
			want:  "my_index:3:2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.shard.ShardKey(); got != tt.want {
				t.Errorf("ShardKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShard_UniqueShardKey(t *testing.T) {
	s := Shard{IndexName: "idx", ShardId: 5, ReplicaId: 3}
	want := "idx:5"
	if got := s.UniqueShardKey(); got != want {
		t.Errorf("UniqueShardKey() = %v, want %v", got, want)
	}
}

func TestShard_FromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Shard
	}{
		{
			name:  "full key with 3 parts",
			input: "my_index:2:1",
			want:  Shard{IndexName: "my_index", ShardId: 2, ReplicaId: 1},
		},
		{
			name:  "unique key with 2 parts",
			input: "my_index:3",
			want:  Shard{IndexName: "my_index", ShardId: 3},
		},
		{
			name:  "index name only",
			input: "my_index",
			want:  Shard{IndexName: "my_index"},
		},
		{
			name:  "invalid shard id",
			input: "my_index:abc",
			want:  Shard{},
		},
		{
			name:  "invalid replica id",
			input: "my_index:1:abc",
			want:  Shard{},
		},
		{
			name:  "empty string",
			input: "",
			want:  Shard{IndexName: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Shard
			s.FromString(tt.input)
			if !reflect.DeepEqual(s, tt.want) {
				t.Errorf("FromString(%q) = %+v, want %+v", tt.input, s, tt.want)
			}
		})
	}
}

func TestShard_Valid(t *testing.T) {
	tests := []struct {
		name  string
		shard Shard
		want  bool
	}{
		{"valid", Shard{IndexName: "idx"}, true},
		{"empty name", Shard{IndexName: ""}, false},
		{"with shard id", Shard{IndexName: "idx", ShardId: 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.shard.Valid(); got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShard_GenerateReplicaKeys(t *testing.T) {
	tests := []struct {
		name     string
		shard    Shard
		replicas int
		want     []string
	}{
		{
			name:     "3 replicas",
			shard:    Shard{IndexName: "idx", ShardId: 0},
			replicas: 3,
			want:     []string{"idx:0:0", "idx:0:1", "idx:0:2"},
		},
		{
			name:     "1 replica",
			shard:    Shard{IndexName: "idx", ShardId: 2},
			replicas: 1,
			want:     []string{"idx:2:0"},
		},
		{
			name:     "invalid shard returns nil",
			shard:    Shard{IndexName: ""},
			replicas: 3,
			want:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.shard.GenerateReplicaKeys(tt.replicas)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateReplicaKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexManifest_GenerateShards(t *testing.T) {
	m := &IndexManifest{
		Meta: IndexMeta{Name: "idx", Shards: 2, Replicas: 2},
	}
	shards := m.GenerateShards()
	if len(shards) != 4 {
		t.Fatalf("expected 4 shards, got %d", len(shards))
	}
	expected := []Shard{
		{IndexName: "idx", ShardId: 0, ReplicaId: 0},
		{IndexName: "idx", ShardId: 0, ReplicaId: 1},
		{IndexName: "idx", ShardId: 1, ReplicaId: 0},
		{IndexName: "idx", ShardId: 1, ReplicaId: 1},
	}
	if !reflect.DeepEqual(shards, expected) {
		t.Errorf("GenerateShards() = %v, want %v", shards, expected)
	}
}

func TestIndexManifest_GenerateUniqueShards(t *testing.T) {
	m := &IndexManifest{
		Meta: IndexMeta{Name: "idx", Shards: 3, Replicas: 2},
	}
	shards := m.GenerateUniqueShards()
	if len(shards) != 3 {
		t.Fatalf("expected 3 unique shards, got %d", len(shards))
	}
	for i, s := range shards {
		if s.ShardId != int32(i) {
			t.Errorf("shard[%d].ShardId = %d, want %d", i, s.ShardId, i)
		}
		if s.ReplicaId != 0 {
			t.Errorf("shard[%d].ReplicaId = %d, want 0", i, s.ReplicaId)
		}
	}
}

func TestIndexManifest_GenerateShards_Zero(t *testing.T) {
	m := &IndexManifest{
		Meta: IndexMeta{Name: "idx", Shards: 0, Replicas: 0},
	}
	shards := m.GenerateShards()
	if len(shards) != 0 {
		t.Errorf("expected 0 shards, got %d", len(shards))
	}
}

func TestNewIndex_Mock(t *testing.T) {
	manifest := &IndexManifest{
		Meta: IndexMeta{Name: "test", Type: Mock, InputDim: 10, Shards: 1, Replicas: 1},
	}
	shard := Shard{IndexName: "test", ShardId: 0, ReplicaId: 0}
	idx, err := NewIndex(context.Background(), manifest, shard)
	if err != nil {
		t.Fatalf("NewIndex() error = %v", err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if idx.InputDim() != 10 {
		t.Errorf("InputDim() = %d, want 10", idx.InputDim())
	}
}

func TestNewIndex_MockFailed(t *testing.T) {
	manifest := &IndexManifest{
		Meta: IndexMeta{Name: "failed", Type: Mock, InputDim: 10},
	}
	shard := Shard{IndexName: "failed", ShardId: 0, ReplicaId: 0}
	_, err := NewIndex(context.Background(), manifest, shard)
	if err == nil {
		t.Fatal("expected error for 'failed' mock index")
	}
}

func TestNewIndex_InvalidType(t *testing.T) {
	manifest := &IndexManifest{
		Meta: IndexMeta{Name: "test", Type: "unknown"},
	}
	shard := Shard{IndexName: "test", ShardId: 0}
	_, err := NewIndex(context.Background(), manifest, shard)
	if err != config.ErrInvalidIndexType {
		t.Errorf("expected ErrInvalidIndexType, got %v", err)
	}
}

func TestIndexType_ToString(t *testing.T) {
	f := Faiss
	if f.ToString() != "faiss" {
		t.Errorf("expected 'faiss', got '%s'", f.ToString())
	}

	var nilType *IndexType
	if nilType.ToString() != "" {
		t.Errorf("expected empty string for nil, got '%s'", nilType.ToString())
	}
}

func TestMockIndex_Search(t *testing.T) {
	manifest := &IndexManifest{
		Meta: IndexMeta{Name: "test", Type: Mock, InputDim: 3},
	}
	shard := Shard{IndexName: "test", ShardId: 0}
	idx, err := newMockIndex(context.Background(), manifest, shard)
	if err != nil {
		t.Fatal(err)
	}

	// Valid search
	distances, labels, err := idx.Search([]float32{1.0, 2.0, 3.0}, 3)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(distances) != 3 || len(labels) != 3 {
		t.Errorf("expected 3 results, got %d distances and %d labels", len(distances), len(labels))
	}

	// Wrong dimension
	_, _, err = idx.Search([]float32{1.0}, 3)
	if err != config.ErrWrongInputDimension {
		t.Errorf("expected ErrWrongInputDimension, got %v", err)
	}
}

func TestMockIndex_Methods(t *testing.T) {
	manifest := &IndexManifest{
		Meta: IndexMeta{Name: "test", Type: Mock, InputDim: 5},
	}
	shard := Shard{IndexName: "test", ShardId: 1, ReplicaId: 2}
	idx, _ := newMockIndex(context.Background(), manifest, shard)

	if idx.VectorCount() != 123 {
		t.Errorf("VectorCount() = %d, want 123", idx.VectorCount())
	}
	if idx.MetricType() != "mockType" {
		t.Errorf("MetricType() = %s, want mockType", idx.MetricType())
	}
	if idx.InputDim() != 5 {
		t.Errorf("InputDim() = %d, want 5", idx.InputDim())
	}
	if idx.CheckAvailable() != nil {
		t.Errorf("CheckAvailable() should be nil")
	}
	if idx.Revision() != 123 {
		t.Errorf("Revision() = %d, want 123", idx.Revision())
	}
	if idx.Meta().Name != "test" {
		t.Errorf("Meta().Name = %s, want test", idx.Meta().Name)
	}
	if idx.Shard().ShardId != 1 {
		t.Errorf("Shard().ShardId = %d, want 1", idx.Shard().ShardId)
	}
	if err := idx.Reload(context.Background()); err != nil {
		t.Errorf("Reload() error = %v", err)
	}
	if err := idx.startReload(ReloadSetting{}); err != nil {
		t.Errorf("startReload() error = %v", err)
	}
	idx.Delete() // should not panic
}
