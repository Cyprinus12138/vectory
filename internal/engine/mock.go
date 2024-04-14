package engine

import (
	"context"
	"errors"
	"sync"
)

type MockIndex struct {
	rw *sync.RWMutex

	shard    Shard
	revision int64

	manifest *IndexManifest
}

func newMockIndex(ctx context.Context, manifest *IndexManifest, shard Shard) (*MockIndex, error) {
	if manifest.Meta.Name == "failed" {
		return nil, errors.New("mock error")
	}

	return &MockIndex{
		rw:       &sync.RWMutex{},
		shard:    shard,
		revision: 123,
		manifest: manifest,
	}, nil
}

func (m *MockIndex) Search(x []float32, k int64) (distances []float32, labels []string, err error) {
	return []float32{0.5, 0.5, 0.5}, []string{"1", "2", "3"}, nil
}

func (m *MockIndex) Delete() {
	return
}

func (m *MockIndex) VectorCount() int64 {
	return 123
}

func (m *MockIndex) MetricType() string {
	return "mockType"
}

func (m *MockIndex) InputDim() int {
	return m.manifest.Meta.InputDim
}

func (m *MockIndex) CheckAvailable() error {
	return nil
}

func (m *MockIndex) Revision() int64 {
	m.rw.RLock()
	defer m.rw.RUnlock()

	return m.revision
}

func (m *MockIndex) Reload(ctx context.Context) error {
	return nil
}

func (m *MockIndex) Meta() IndexMeta {
	return m.manifest.Meta
}

func (m *MockIndex) Shard() *Shard {
	return &m.shard
}

func (m *MockIndex) startReload(setting ReloadSetting) (err error) {
	return nil
}
