package engine

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Cyprinus12138/vectory/internal/config"
)

// --- Mocks ---

type mockFaissHandle struct {
	d          int
	ntotal     int64
	metricType int
	searchFn   func([]float32, int64) ([]float32, []int64, error)
	deleted    bool
}

func (m *mockFaissHandle) D() int          { return m.d }
func (m *mockFaissHandle) Ntotal() int64   { return m.ntotal }
func (m *mockFaissHandle) MetricType() int { return m.metricType }
func (m *mockFaissHandle) Delete()         { m.deleted = true }
func (m *mockFaissHandle) Search(x []float32, k int64) ([]float32, []int64, error) {
	if m.searchFn != nil {
		return m.searchFn(x, k)
	}
	return nil, nil, nil
}

type mockDl struct {
	downloadFn       func() (string, int64, error)
	downloadUpdateFn func(int64) (string, int64, error)
}

func (m *mockDl) Download() (string, int64, error) {
	return m.downloadFn()
}
func (m *mockDl) DownloadUpdate(rev int64) (string, int64, error) {
	return m.downloadUpdateFn(rev)
}

// --- Helpers ---

func newTestFaissIndex(handle FaissIndexHandle, revision int64) *FaissIndex {
	reloading := &atomic.Bool{}
	reloading.Store(false)
	return &FaissIndex{
		rw:        &sync.RWMutex{},
		index:     handle,
		reloading: reloading,
		shard:     Shard{IndexName: "test_idx", ShardId: 0, ReplicaId: 0},
		revision:  revision,
		manifest: &IndexManifest{
			Meta:   IndexMeta{Name: "test_idx", Type: Faiss, InputDim: 3},
			Source: &IndexSource{Type: LocalPath, Location: "/tmp"},
		},
	}
}

// --- Search tests ---

func TestFaissIndex_Search(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		handle := &mockFaissHandle{
			d: 3,
			searchFn: func(x []float32, k int64) ([]float32, []int64, error) {
				return []float32{0.1, 0.2}, []int64{10, 20}, nil
			},
		}
		f := newTestFaissIndex(handle, 1)
		distances, labels, err := f.Search([]float32{1, 2, 3}, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(distances) != 2 {
			t.Errorf("expected 2 distances, got %d", len(distances))
		}
		if labels[0] != "10" || labels[1] != "20" {
			t.Errorf("label conversion failed: got %v", labels)
		}
	})

	t.Run("nil index", func(t *testing.T) {
		f := newTestFaissIndex(nil, 1)
		_, _, err := f.Search([]float32{1, 2, 3}, 2)
		if !errors.Is(err, config.ErrNilIndex) {
			t.Errorf("expected ErrNilIndex, got %v", err)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		handle := &mockFaissHandle{d: 3}
		f := newTestFaissIndex(handle, 1)
		_, _, err := f.Search([]float32{}, 2)
		if !errors.Is(err, config.ErrEmptyInput) {
			t.Errorf("expected ErrEmptyInput, got %v", err)
		}
	})

	t.Run("wrong dimension", func(t *testing.T) {
		handle := &mockFaissHandle{d: 3}
		f := newTestFaissIndex(handle, 1)
		_, _, err := f.Search([]float32{1, 2}, 2)
		if !errors.Is(err, config.ErrWrongInputDimension) {
			t.Errorf("expected ErrWrongInputDimension, got %v", err)
		}
	})

	t.Run("underlying error propagates", func(t *testing.T) {
		searchErr := errors.New("faiss search failed")
		handle := &mockFaissHandle{
			d: 3,
			searchFn: func(x []float32, k int64) ([]float32, []int64, error) {
				return nil, nil, searchErr
			},
		}
		f := newTestFaissIndex(handle, 1)
		_, _, err := f.Search([]float32{1, 2, 3}, 2)
		if err != searchErr {
			t.Errorf("expected search error, got %v", err)
		}
	})

	t.Run("negative label conversion", func(t *testing.T) {
		handle := &mockFaissHandle{
			d: 2,
			searchFn: func(x []float32, k int64) ([]float32, []int64, error) {
				return []float32{0.0, 0.5, 1.0}, []int64{-1, 0, 999}, nil
			},
		}
		f := newTestFaissIndex(handle, 1)
		_, labels, err := f.Search([]float32{1, 2}, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if labels[0] != "-1" || labels[1] != "0" || labels[2] != "999" {
			t.Errorf("expected [\"-1\", \"0\", \"999\"], got %v", labels)
		}
	})
}

// --- CheckAvailable tests ---

func TestFaissIndex_CheckAvailable(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		if err := f.CheckAvailable(); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("nil index", func(t *testing.T) {
		f := newTestFaissIndex(nil, 1)
		if err := f.CheckAvailable(); !errors.Is(err, config.ErrNilIndex) {
			t.Errorf("expected ErrNilIndex, got %v", err)
		}
	})
}

// --- Accessor tests ---

func TestFaissIndex_Revision(t *testing.T) {
	f := newTestFaissIndex(&mockFaissHandle{}, 42)
	if got := f.Revision(); got != 42 {
		t.Errorf("Revision() = %d, want 42", got)
	}
}

func TestFaissIndex_Meta(t *testing.T) {
	f := newTestFaissIndex(&mockFaissHandle{}, 1)
	if f.Meta().Name != "test_idx" {
		t.Errorf("Meta().Name = %s, want test_idx", f.Meta().Name)
	}
}

func TestFaissIndex_Shard(t *testing.T) {
	f := newTestFaissIndex(&mockFaissHandle{}, 1)
	s := f.Shard()
	if s.IndexName != "test_idx" || s.ShardId != 0 {
		t.Errorf("Shard() = %+v, unexpected", s)
	}
}

func TestFaissIndex_VectorCount(t *testing.T) {
	handle := &mockFaissHandle{ntotal: 500}
	f := newTestFaissIndex(handle, 1)
	if got := f.VectorCount(); got != 500 {
		t.Errorf("VectorCount() = %d, want 500", got)
	}
}

func TestFaissIndex_MetricType(t *testing.T) {
	tests := []struct {
		name       string
		metricType int
		want       string
	}{
		{"inner_product", 0, "INNER_PRODUCT"},
		{"l2", 1, "L2"},
		{"unknown", 99, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle := &mockFaissHandle{metricType: tt.metricType}
			f := newTestFaissIndex(handle, 1)
			if got := f.MetricType(); got != tt.want {
				t.Errorf("MetricType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFaissIndex_InputDim(t *testing.T) {
	handle := &mockFaissHandle{d: 128}
	f := newTestFaissIndex(handle, 1)
	if got := f.InputDim(); got != 128 {
		t.Errorf("InputDim() = %d, want 128", got)
	}
}

// --- Delete tests ---

func TestFaissIndex_Delete(t *testing.T) {
	handle := &mockFaissHandle{}
	f := newTestFaissIndex(handle, 1)
	f.Delete()
	if !handle.deleted {
		t.Error("Delete() did not call handle.Delete()")
	}
}

// --- Reload tests ---

func TestFaissIndex_Reload(t *testing.T) {
	t.Run("already reloading", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{d: 3}, 1)
		f.reloading.Store(true)
		err := f.Reload(context.Background())
		if !errors.Is(err, config.ErrAlreadyReloading) {
			t.Errorf("expected ErrAlreadyReloading, got %v", err)
		}
	})

	t.Run("downloader creation fails", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{d: 3}, 1)
		dlErr := errors.New("downloader failed")
		f.newDownloadFn = func(ctx context.Context, source *IndexSource, shard Shard) (Downloader, error) {
			return nil, dlErr
		}
		err := f.Reload(context.Background())
		if err != dlErr {
			t.Errorf("expected downloader error, got %v", err)
		}
		// reloading flag should be reset
		if f.reloading.Load() {
			t.Error("reloading flag not reset after error")
		}
	})

	t.Run("download update fails", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{d: 3}, 1)
		dlErr := errors.New("download failed")
		f.newDownloadFn = func(ctx context.Context, source *IndexSource, shard Shard) (Downloader, error) {
			return &mockDl{
				downloadUpdateFn: func(rev int64) (string, int64, error) {
					return "", 0, dlErr
				},
			}, nil
		}
		err := f.Reload(context.Background())
		if err != dlErr {
			t.Errorf("expected download error, got %v", err)
		}
	})

	t.Run("readIndex fails", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{d: 3}, 1)
		f.newDownloadFn = func(ctx context.Context, source *IndexSource, shard Shard) (Downloader, error) {
			return &mockDl{
				downloadUpdateFn: func(rev int64) (string, int64, error) {
					return "/tmp/new_index.faiss", 10, nil
				},
			}, nil
		}
		readErr := errors.New("read index failed")
		f.readIndexFn = func(filename string, ioflags int) (FaissIndexHandle, error) {
			return nil, readErr
		}
		err := f.Reload(context.Background())
		if err != readErr {
			t.Errorf("expected readIndex error, got %v", err)
		}
	})

	t.Run("successful reload", func(t *testing.T) {
		oldHandle := &mockFaissHandle{d: 3}
		f := newTestFaissIndex(oldHandle, 1)

		newHandle := &mockFaissHandle{d: 3, ntotal: 1000}
		f.newDownloadFn = func(ctx context.Context, source *IndexSource, shard Shard) (Downloader, error) {
			return &mockDl{
				downloadUpdateFn: func(rev int64) (string, int64, error) {
					if rev != 1 {
						t.Errorf("DownloadUpdate called with revision %d, want 1", rev)
					}
					return "/tmp/new.faiss", 42, nil
				},
			}, nil
		}
		f.readIndexFn = func(filename string, ioflags int) (FaissIndexHandle, error) {
			if filename != "/tmp/new.faiss" {
				t.Errorf("readIndex called with %s, want /tmp/new.faiss", filename)
			}
			return newHandle, nil
		}

		err := f.Reload(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.Revision() != 42 {
			t.Errorf("revision = %d, want 42", f.Revision())
		}
		if f.VectorCount() != 1000 {
			t.Errorf("VectorCount() = %d, want 1000", f.VectorCount())
		}
		if f.reloading.Load() {
			t.Error("reloading flag not reset after success")
		}
	})
}

// --- startReload tests ---

func TestFaissIndex_startReload(t *testing.T) {
	t.Run("passive mode", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		err := f.startReload(&ReloadSetting{Mode: Passive})
		if err != nil {
			t.Errorf("expected nil for passive mode, got %v", err)
		}
	})

	t.Run("active cron", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		err := f.startReload(&ReloadSetting{
			Mode: Active,
			Schedule: ScheduleSetting{
				Type:    Cron,
				Crontab: "0 * * * *",
			},
		})
		if err != nil {
			t.Errorf("expected nil for valid cron, got %v", err)
		}
	})

	t.Run("active interval", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		err := f.startReload(&ReloadSetting{
			Mode: Active,
			Schedule: ScheduleSetting{
				Type:     Internal,
				Interval: "10s",
			},
		})
		if err != nil {
			t.Errorf("expected nil for valid interval, got %v", err)
		}
	})

	t.Run("fixed time falls through with empty cron", func(t *testing.T) {
		// FixedTime is not implemented — it matches its own case but does nothing,
		// leaving cronStr empty. The cron library then returns an error.
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		err := f.startReload(&ReloadSetting{
			Mode: Active,
			Schedule: ScheduleSetting{
				Type: FixedTime,
			},
		})
		if err == nil {
			t.Error("expected error for unimplemented FixedTime schedule")
		}
	})

	t.Run("truly invalid schedule type", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		err := f.startReload(&ReloadSetting{
			Mode: Active,
			Schedule: ScheduleSetting{
				Type: "bogus",
			},
		})
		if !errors.Is(err, config.ErrInvalidScheduleType) {
			t.Errorf("expected ErrInvalidScheduleType, got %v", err)
		}
	})

	t.Run("invalid cron string", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		err := f.startReload(&ReloadSetting{
			Mode: Active,
			Schedule: ScheduleSetting{
				Type:    Cron,
				Crontab: "not a cron",
			},
		})
		if err == nil {
			t.Error("expected error for invalid cron string")
		}
	})

	t.Run("unknown mode is no-op", func(t *testing.T) {
		f := newTestFaissIndex(&mockFaissHandle{}, 1)
		err := f.startReload(&ReloadSetting{
			Mode: "unknown",
		})
		if err != nil {
			t.Errorf("expected nil for unknown mode, got %v", err)
		}
	})
}
