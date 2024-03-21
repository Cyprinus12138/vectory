package engine

import (
	"context"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/engine/downloader"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/robfig/cron/v3"
)

var scheduler = cron.New()

var metricType = map[int]string{
	0: "INNER_PRODUCT", //< maximum inner product search
	1: "L2",            //< squared L2 search
	2: "L1",            //< L1 (aka cityblock)
	3: "Linf",          //< infinity distance
	4: "Lp",            //< L_p distance, p is given by a faiss::Index
	// metric_arg

	// some additional metrics defined in scipy.spatial.distance
	20: "Canberra",
	21: "BrayCurtis",
	22: "JensenShannon",
	23: "Jaccard", //< defined as: sum_i(min(a_i, b_i)) / sum_i(max(a_i, b_i))   <-- where a_i, b_i > 0
}

const (
	shardKeyFmt = "%s_%d_%d"
)

type IndexType string

const (
	Faiss IndexType = "faiss"
	ScaNN IndexType = "scann"
	RediS IndexType = "rs"
)

func (i *IndexType) ToString() string {
	if i != nil {
		return string(*i)
	}
	return ""
}

type ReloadMode string

const (
	Active  ReloadMode = "active"
	Passive ReloadMode = "passive"
)

type ScheduleType string

const (
	Cron      ScheduleType = "cron"
	Internal  ScheduleType = "interval"
	FixedTime ScheduleType = "fixed" // Not really supported now.
)

func (i *ScheduleType) ToString() string {
	if i != nil {
		return string(*i)
	}
	return ""
}

type TimeUnit string

const (
	Year  TimeUnit = "year"
	Month TimeUnit = "month"
	Week  TimeUnit = "week"
	Day   TimeUnit = "day"
)

type IndexMeta struct {
	Name     string
	Type     IndexType
	InputDim int

	Shards   int32
	Replicas int32

	CTime   int64
	MTime   int64
	Version string
}

type ScheduleSetting struct {
	Type ScheduleType

	// Crontab is used for cron mode
	Crontab string

	// Interval is used for interval mode, the format should be like "1h2m3s"
	Interval string

	// Every and Time are used for fixed mode, will schedule the reloading yearly/monthly/weekly/daily.
	Every TimeUnit
	Time  string

	// RandomDwellTime If turned on, the reloading process will be delay a few seconds randomly, avoiding the burst load for the data source of index.
	RandomDwellTime bool
}

type ReloadSetting struct {
	Enable bool
	Mode   ReloadMode

	Schedule ScheduleSetting
}

type IndexManifest struct {
	Meta   IndexMeta
	Source *downloader.IndexSource
	Reload ReloadSetting
}

func (m *IndexManifest) Validate() error {
	// TODO
	return nil
}

type Shard struct {
	ShardId   int32
	ReplicaId int32
}

func (m *IndexManifest) GenerateShards() []Shard {
	meta := m.Meta
	result := make([]Shard, 0, meta.Shards*meta.Replicas)
	var shard, replica int32
	for shard = 0; shard < meta.Shards; shard++ {
		for replica = 0; replica < meta.Replicas; replica++ {
			result = append(result, Shard{
				ShardId:   shard,
				ReplicaId: replica,
			})
		}
	}
	return result
}

type Index interface {
	// Search queries the index with the vectors in x.
	// Returns the IDs of the k nearest neighbors for each query vector and the
	// corresponding distances.
	Search(x []float32, k int64) (distances []float32, labels []int64, err error)

	// Delete frees the memory used by the index.
	Delete()

	// VectorCount returns the number of indexed vectors.
	VectorCount() int64

	// MetricType returns the metric type of the index.
	MetricType() string

	// InputDim returns the dimension of the indexed vectors.
	InputDim() int

	// CheckAvailable is used to check whether the index is available before calling the index searching.
	CheckAvailable() error

	// ShardKey returns a shard key string with format of: {indexName}_{shardId}_{replicaId}
	ShardKey() string

	// Revision returns the loaded index revision.
	Revision() int64

	// Reload triggers a reloading action for an index, can be call manually or scheduled.
	Reload(ctx context.Context) error
}

func NewIndex(ctx context.Context, manifest *IndexManifest, shard Shard) (Index, error) {
	switch manifest.Meta.Type {
	case Faiss:
		return newFaissIndex(ctx, manifest, shard)
	default:
		logger.CtxError(ctx, "invalid index type", logger.String("type", manifest.Meta.Type.ToString()))
		return nil, config.ErrInvalidIndexType
	}
}

func GetShardKey(idx IndexManifest, shard Shard) string {
	return fmt.Sprintf(shardKeyFmt, idx.Meta.Name, shard.ShardId, shard.ReplicaId)
}
