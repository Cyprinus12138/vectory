package engine

import (
	"context"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/DataIntelligenceCrew/go-faiss"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"sync"
	"sync/atomic"
)

type FaissIndex struct {
	rw          *sync.RWMutex
	index       faiss.Index
	reloading   *atomic.Bool
	reloadEntry cron.EntryID
	shard       Shard
	revision    int64

	manifest *IndexManifest
}

func newFaissIndex(ctx context.Context, manifest *IndexManifest, shard Shard) (*FaissIndex, error) {
	dl, err := NewDownLoader(ctx, manifest.Source, shard)
	if err != nil {
		logger.CtxError(ctx, "create downloader failed", logger.Err(err), logger.Interface("source", manifest.Source))
		return nil, err
	}

	localPath, revision, err := dl.Download()
	if err != nil {
		logger.CtxError(ctx, "download index file failed", logger.Err(err), logger.Interface("source", manifest.Source))
		return nil, err
	}

	rawIndex, err := faiss.ReadIndex(localPath, faiss.IOFlagReadOnly)
	if err != nil {
		logger.CtxError(ctx, "load index file failed", logger.Err(err), logger.Interface("source", manifest.Source))
		return nil, err
	}

	reloading := &atomic.Bool{}
	reloading.Store(false)
	index := &FaissIndex{
		rw:        &sync.RWMutex{},
		index:     rawIndex,
		reloading: reloading,

		shard:    shard,
		revision: revision,
		manifest: manifest,
	}

	if manifest.Reload.Enable {
		err = index.startReload(manifest.Reload)
		if err != nil {
			logger.CtxError(ctx, "load index file failed", logger.Err(err), logger.Interface("source", manifest.Source), logger.Interface("reload", manifest.Reload))
			return nil, err
		}
	}

	return index, nil
}

func (f *FaissIndex) Search(x []float32, k int64) (distances []float32, labels []int64, err error) {
	f.rw.RLock()
	defer f.rw.RUnlock()

	if err = f.CheckAvailable(); err != nil {
		return nil, nil, err
	}

	if len(x) == 0 {
		return nil, nil, config.ErrEmptyInput
	}

	if len(x) < f.InputDim() {
		return nil, nil, config.ErrWrongInputDimension
	}

	return f.index.Search(x, k)
}

func (f *FaissIndex) Delete() {
	f.rw.Lock()
	defer f.rw.Unlock()

	f.index.Delete()
}

func (f *FaissIndex) VectorCount() int64 {
	f.rw.RLock()
	defer f.rw.RUnlock()

	return f.index.Ntotal()
}

func (f *FaissIndex) MetricType() string {
	f.rw.RLock()
	defer f.rw.RUnlock()

	return metricType[f.index.MetricType()]
}

func (f *FaissIndex) InputDim() int {
	f.rw.RLock()
	defer f.rw.RUnlock()

	return f.index.D()
}

func (f *FaissIndex) CheckAvailable() error {
	f.rw.RLock()
	defer f.rw.RUnlock()

	if f.index == nil {
		return config.ErrNilIndex
	}
	return nil
}

func (f *FaissIndex) Revision() int64 {
	f.rw.RLock()
	defer f.rw.RUnlock()

	return f.revision
}

func (f *FaissIndex) Reload(ctx context.Context) error {
	if f.reloading.Load() {
		logger.CtxError(ctx, "index is reloading", logger.Interface("index", f.manifest.Meta))
		return config.ErrAlreadyReloading
	}

	f.reloading.Store(true)
	defer f.reloading.Store(false)

	source := f.manifest.Source
	dl, err := NewDownLoader(ctx, source, f.shard)
	if err != nil {
		logger.CtxError(ctx, "create downloader failed", logger.Err(err), logger.Interface("source", source))
		return err
	}

	f.rw.RLock() // Lock for accessing revision: should be considered together with index itself.
	localPath, revision, err := dl.DownloadUpdate(f.revision)
	f.rw.RUnlock()
	if err != nil && !errors.Is(err, config.ErrIndexRevisionUpToDate) {
		logger.CtxError(ctx, "download index file failed", logger.Err(err), logger.Interface("source", source))
		return err
	}

	rawIndex, err := faiss.ReadIndex(localPath, faiss.IOFlagReadOnly)
	if err != nil {
		logger.CtxError(ctx, "load index file failed", logger.Err(err), logger.Interface("source", source))
		return err
	}

	f.rw.Lock()
	f.index = rawIndex
	f.revision = revision
	f.rw.Unlock()

	return nil
}

func (f *FaissIndex) Meta() IndexMeta {
	return f.manifest.Meta
}

func (f *FaissIndex) Shard() *Shard {
	return &f.shard
}

func (f *FaissIndex) startReload(setting ReloadSetting) (err error) {
	if setting.Mode == Passive {
		return nil
	}
	if setting.Mode == Active {
		schedule := setting.Schedule
		var cronStr string
		switch schedule.Type {
		case Cron:
			cronStr = schedule.Crontab
		case Internal:
			cronStr = fmt.Sprintf("@every %s", schedule.Interval)
		case FixedTime:
		default:
			logger.Error("invalid or unsupported schedule type", logger.String("schedule_type", schedule.Type.ToString()))
			return config.ErrInvalidScheduleType
		}

		f.reloadEntry, err = GetScheduler().AddFunc(cronStr, func() {
			err := f.Reload(context.Background())
			if err != nil {
				logger.Error("reload failed", logger.Err(err))
				return
			}
		})
		if err != nil {
			logger.Error(
				"invalid schedule config",
				logger.Interface("setting", schedule),
				logger.String("cron_str", cronStr),
				logger.Err(err),
			)
			return err
		}
	}

	return nil
}
