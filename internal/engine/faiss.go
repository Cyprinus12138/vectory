package engine

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/DataIntelligenceCrew/go-faiss"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

// FaissIndexHandle abstracts the subset of faiss.Index methods used by FaissIndex,
// allowing unit tests to inject a mock without the FAISS C library.
type FaissIndexHandle interface {
	D() int
	Ntotal() int64
	MetricType() int
	Search(x []float32, k int64) ([]float32, []int64, error)
	Delete()
}

// readIndexFunc reads a FAISS index from disk. Defaults to faiss.ReadIndex.
type readIndexFunc func(filename string, ioflags int) (FaissIndexHandle, error)

// newDownloaderFunc creates a Downloader. Defaults to NewDownLoader.
type newDownloaderFunc func(ctx context.Context, source *IndexSource, shard Shard) (Downloader, error)

type FaissIndex struct {
	rw          *sync.RWMutex
	index       FaissIndexHandle
	reloading   *atomic.Bool
	reloadEntry cron.EntryID
	shard       Shard
	revision    int64

	manifest      *IndexManifest
	readIndexFn   readIndexFunc
	newDownloadFn newDownloaderFunc
}

// defaultReadIndex wraps faiss.ReadIndex to match the readIndexFunc signature.
func defaultReadIndex(filename string, ioflags int) (FaissIndexHandle, error) {
	return faiss.ReadIndex(filename, ioflags)
}

func newFaissIndex(ctx context.Context, manifest *IndexManifest, shard Shard) (*FaissIndex, error) {
	reloading := &atomic.Bool{}
	reloading.Store(false)
	index := &FaissIndex{
		rw:            &sync.RWMutex{},
		reloading:     reloading,
		shard:         shard,
		manifest:      manifest,
		readIndexFn:   defaultReadIndex,
		newDownloadFn: NewDownLoader,
	}

	dl, err := index.newDownloadFn(ctx, manifest.Source, shard)
	if err != nil {
		logger.CtxError(ctx, "create downloader failed", logger.Err(err), logger.Interface("source", manifest.Source))
		return nil, err
	}

	localPath, revision, err := dl.Download()
	if err != nil {
		logger.CtxError(ctx, "download index file failed", logger.Err(err), logger.Interface("source", manifest.Source))
		return nil, err
	}

	rawIndex, err := index.readIndexFn(localPath, faiss.IOFlagReadOnly)
	if err != nil {
		logger.CtxError(ctx, "load index file failed", logger.Err(err), logger.Interface("source", manifest.Source))
		return nil, err
	}

	index.index = rawIndex
	index.revision = revision

	if manifest.Reload != nil && manifest.Reload.Enable {
		err = index.startReload(manifest.Reload)
		if err != nil {
			logger.CtxError(ctx, "load index file failed", logger.Err(err), logger.Interface("source", manifest.Source), logger.Interface("reload", manifest.Reload))
			return nil, err
		}
	}

	return index, nil
}

func (f *FaissIndex) Search(x []float32, k int64) (distances []float32, labels []string, err error) {
	f.rw.RLock()
	defer f.rw.RUnlock()

	// Inline nil and dim checks to avoid re-acquiring RLock (nested RLock deadlocks
	// when a writer is waiting on the same mutex).
	if f.index == nil {
		return nil, nil, config.ErrNilIndex
	}
	if len(x) == 0 {
		return nil, nil, config.ErrEmptyInput
	}
	if len(x) < f.index.D() {
		return nil, nil, config.ErrWrongInputDimension
	}

	distances, intLabels, err := f.index.Search(x, k)
	if err != nil {
		return nil, nil, err
	}

	labels = make([]string, len(intLabels))
	for i, label := range intLabels {
		labels[i] = strconv.FormatInt(label, 10)
	}

	return distances, labels, nil
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
	if !f.reloading.CompareAndSwap(false, true) {
		logger.CtxError(ctx, "index is reloading", logger.Interface("index", f.manifest.Meta))
		return config.ErrAlreadyReloading
	}
	defer f.reloading.Store(false)

	source := f.manifest.Source
	dl, err := f.newDownloadFn(ctx, source, f.shard)
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

	rawIndex, err := f.readIndexFn(localPath, faiss.IOFlagReadOnly)
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

func (f *FaissIndex) startReload(setting *ReloadSetting) (err error) {
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
