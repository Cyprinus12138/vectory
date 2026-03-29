package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexSource_GetLatestRevisionPath(t *testing.T) {
	src := &IndexSource{Location: "/data/index"}
	got := src.GetLatestRevisionPath()
	want := "/data/index/_revision"
	if got != want {
		t.Errorf("GetLatestRevisionPath() = %s, want %s", got, want)
	}
}

func TestIndexSource_GetRevisionDirPath(t *testing.T) {
	src := &IndexSource{Location: "/data/index"}
	got := src.GetRevisionDirPath(42)
	want := "/data/index/42"
	if got != want {
		t.Errorf("GetRevisionDirPath() = %s, want %s", got, want)
	}
}

func TestIndexSource_GetShardFilePath(t *testing.T) {
	tests := []struct {
		name   string
		source *IndexSource
		rev    int64
		shard  Shard
		want   string
	}{
		{
			name: "basic substitution",
			source: &IndexSource{
				Location: "/data",
				NameFmt:  "{index_name}_shard_{shard_id}.faiss",
			},
			rev:   10,
			shard: Shard{IndexName: "my_idx", ShardId: 3},
			want:  "/data/10/my_idx_shard_3.faiss",
		},
		{
			name: "no placeholders",
			source: &IndexSource{
				Location: "/data",
				NameFmt:  "index.faiss",
			},
			rev:   1,
			shard: Shard{IndexName: "idx", ShardId: 0},
			want:  "/data/1/index.faiss",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.source.GetShardFilePath(tt.rev, tt.shard)
			if got != tt.want {
				t.Errorf("GetShardFilePath() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewDownLoader_LocalPath(t *testing.T) {
	source := &IndexSource{Type: LocalPath, Location: "/tmp"}
	shard := Shard{IndexName: "idx", ShardId: 0}
	dl, err := NewDownLoader(context.Background(), source, shard)
	if err != nil {
		t.Fatalf("NewDownLoader() error = %v", err)
	}
	if dl == nil {
		t.Fatal("expected non-nil downloader")
	}
}

func TestNewDownLoader_Unsupported(t *testing.T) {
	source := &IndexSource{Type: Hdfs, Location: "hdfs://cluster/path"}
	shard := Shard{IndexName: "idx", ShardId: 0}
	_, err := NewDownLoader(context.Background(), source, shard)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

func TestLocalDl_Download(t *testing.T) {
	// Create temp directory structure: location/_revision, location/<rev>/<file>
	dir := t.TempDir()
	revision := "5"
	revDir := filepath.Join(dir, revision)
	os.MkdirAll(revDir, 0755)
	os.WriteFile(filepath.Join(dir, "_revision"), []byte(revision), 0644)
	os.WriteFile(filepath.Join(revDir, "idx_shard_0.faiss"), []byte("fake"), 0644)

	source := &IndexSource{
		Type:     LocalPath,
		Location: dir,
		NameFmt:  "{index_name}_shard_{shard_id}.faiss",
	}
	shard := Shard{IndexName: "idx", ShardId: 0}
	dl := newLocalDl(context.Background(), source, shard)

	localPath, rev, err := dl.Download()
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if rev != 5 {
		t.Errorf("revision = %d, want 5", rev)
	}
	expectedPath := filepath.Join(revDir, "idx_shard_0.faiss")
	if localPath != expectedPath {
		t.Errorf("localPath = %s, want %s", localPath, expectedPath)
	}
}

func TestLocalDl_Download_MissingRevision(t *testing.T) {
	dir := t.TempDir()
	source := &IndexSource{Type: LocalPath, Location: dir}
	shard := Shard{IndexName: "idx", ShardId: 0}
	dl := newLocalDl(context.Background(), source, shard)

	_, _, err := dl.Download()
	if err == nil {
		t.Fatal("expected error for missing revision file")
	}
}

func TestLocalDl_Download_InvalidRevision(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_revision"), []byte("not_a_number"), 0644)

	source := &IndexSource{Type: LocalPath, Location: dir}
	shard := Shard{IndexName: "idx", ShardId: 0}
	dl := newLocalDl(context.Background(), source, shard)

	_, _, err := dl.Download()
	if err == nil {
		t.Fatal("expected error for invalid revision content")
	}
}

func TestLocalDl_Download_MissingFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_revision"), []byte("1"), 0644)
	os.MkdirAll(filepath.Join(dir, "1"), 0755)
	// Don't create the actual index file

	source := &IndexSource{
		Type:     LocalPath,
		Location: dir,
		NameFmt:  "{index_name}_shard_{shard_id}.faiss",
	}
	shard := Shard{IndexName: "idx", ShardId: 0}
	dl := newLocalDl(context.Background(), source, shard)

	_, _, err := dl.Download()
	if err == nil {
		t.Fatal("expected error for missing shard file")
	}
}

func TestLocalDl_DownloadUpdate(t *testing.T) {
	dir := t.TempDir()
	revDir := filepath.Join(dir, "10")
	os.MkdirAll(revDir, 0755)
	os.WriteFile(filepath.Join(dir, "_revision"), []byte("10"), 0644)
	os.WriteFile(filepath.Join(revDir, "idx_shard_0.faiss"), []byte("fake"), 0644)

	source := &IndexSource{
		Type:     LocalPath,
		Location: dir,
		NameFmt:  "{index_name}_shard_{shard_id}.faiss",
	}
	shard := Shard{IndexName: "idx", ShardId: 0}
	dl := newLocalDl(context.Background(), source, shard)

	// Current revision is older -> should download
	localPath, rev, err := dl.DownloadUpdate(5)
	if err != nil {
		t.Fatalf("DownloadUpdate() error = %v", err)
	}
	if rev != 10 {
		t.Errorf("revision = %d, want 10", rev)
	}
	if localPath == "" {
		t.Error("expected non-empty localPath")
	}

	// Current revision is same -> should return up-to-date error
	_, _, err = dl.DownloadUpdate(10)
	if err == nil {
		t.Fatal("expected error for up-to-date revision")
	}

	// Current revision is newer -> should return up-to-date error
	_, _, err = dl.DownloadUpdate(15)
	if err == nil {
		t.Fatal("expected error for newer current revision")
	}
}
