package engine

import (
	"context"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/robfig/cron/v3"
	"strconv"
	"strings"
)

var scheduler = cron.New()

func GetScheduler() *cron.Cron {
	return scheduler
}

func init() {
	GetScheduler().Start()
}

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
	shardKeyFmt          = "%s:%d:%d" // {indexName}:{shardId}:{replicaId}
	shardKeyWoReplicaFmt = "%s:%d"    // {indexName}:{shardId}
	shardKeySep          = ":"
)

type IndexType string

const (
	Faiss IndexType = "faiss"
	ScaNN IndexType = "scann"
	RediS IndexType = "rs"
	Mock  IndexType = "mock" // Used only for unit tests.
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
	Name     string    `json:"name,omitempty" yaml:"name"`
	Type     IndexType `json:"type,omitempty" yaml:"type"`
	InputDim int       `json:"input_dim,omitempty" yaml:"input_dim"`

	Shards   int32 `json:"shards,omitempty" yaml:"shards"`
	Replicas int32 `json:"replicas,omitempty" yaml:"replicas"`

	CTime   int64  `json:"c_time,omitempty" yaml:"c_time"`
	MTime   int64  `json:"m_time,omitempty" yaml:"m_time"`
	Version string `json:"version,omitempty" yaml:"version"`
}

type ScheduleSetting struct {
	Type ScheduleType `json:"type,omitempty" yaml:"type"`

	// Crontab is used for cron mode
	Crontab string `json:"crontab,omitempty" yaml:"crontab"`

	// Interval is used for interval mode, the format should be like "1h2m3s"
	Interval string `json:"interval,omitempty" yaml:"interval"`

	// Every and Time are used for fixed mode, will schedule the reloading yearly/monthly/weekly/daily.
	Every TimeUnit `json:"every,omitempty" yaml:"every"`
	Time  string   `json:"time,omitempty" yaml:"time"`

	// RandomDwellTime If turned on, the reloading process will be delay a few seconds randomly, avoiding the burst load for the data source of index.
	RandomDwellTime bool `json:"random_dwell_time,omitempty" yaml:"random_dwell_time"`
}

type ReloadSetting struct {
	Enable bool       `json:"enable,omitempty" yaml:"enable"`
	Mode   ReloadMode `json:"mode,omitempty" yaml:"mode"`

	Schedule ScheduleSetting `json:"schedule" yaml:"schedule"`
}

type IndexManifest struct {
	Meta   IndexMeta      `json:"meta" yaml:"meta"`
	Source *IndexSource   `json:"source,omitempty" yaml:"source"`
	Reload *ReloadSetting `json:"reload" yaml:"reload"`
}

func (m *IndexManifest) Validate() error {
	// TODO
	return nil
}

type Shard struct {
	IndexName string `json:"index_name,omitempty" yaml:"index_name"`
	ShardId   int32  `json:"shard_id,omitempty" yaml:"shard_id"`
	ReplicaId int32  `json:"replica_id,omitempty" yaml:"replica_id"`
}

// ShardKey concatenates the IndexName ShardId and ReplicaId.
// Used to determine which node should load this shard.
func (s *Shard) ShardKey() string {
	return fmt.Sprintf(shardKeyFmt, s.IndexName, s.ShardId, s.ReplicaId)
}

// UniqueShardKey concatenates the IndexName and ShardId without ReplicaId.
// Used to maintain the unique shards on each node.
func (s *Shard) UniqueShardKey() string {
	return fmt.Sprintf(shardKeyWoReplicaFmt, s.IndexName, s.ShardId)
}

func (s *Shard) FromString(key string) {
	parts := strings.Split(key, shardKeySep)
	switch len(parts) {
	case 1:
		s.IndexName = parts[0]
	case 2:
		if shardId, ok := strconv.Atoi(parts[1]); ok == nil {
			s.IndexName = parts[0]
			s.ShardId = int32(shardId)
		}
	case 3:
		shardId, ok1 := strconv.Atoi(parts[1])
		repId, ok2 := strconv.Atoi(parts[2])
		if ok1 == nil && ok2 == nil {
			s.IndexName = parts[0]
			s.ShardId = int32(shardId)
			s.ReplicaId = int32(repId)
		}
	}
}

func (s *Shard) GenerateReplicaKeys(replicas int) []string {
	if !s.Valid() {
		return nil
	}
	result := make([]string, replicas)
	for i := 0; i < replicas; i++ {
		result[i] = fmt.Sprintf(shardKeyFmt, s.IndexName, s.ShardId, i)
	}
	return result
}

func (s *Shard) Valid() bool {
	return s.IndexName != ""
}

func (m *IndexManifest) GenerateShards() []Shard {
	meta := m.Meta
	result := make([]Shard, 0, meta.Shards*meta.Replicas)
	var shard, replica int32
	for shard = 0; shard < meta.Shards; shard++ {
		for replica = 0; replica < meta.Replicas; replica++ {
			result = append(result, Shard{
				IndexName: meta.Name,
				ShardId:   shard,
				ReplicaId: replica,
			})
		}
	}
	return result
}

// GenerateUniqueShards ignores the ReplicaId ensuring each single unique shards
// are only return just once.
func (m *IndexManifest) GenerateUniqueShards() []Shard {
	meta := m.Meta
	result := make([]Shard, 0, meta.Shards*meta.Replicas)
	var shard int32
	for shard = 0; shard < meta.Shards; shard++ {
		result = append(result, Shard{
			IndexName: meta.Name,
			ShardId:   shard,
		})
	}
	return result
}

type Index interface {
	// Search queries the index with the vectors in x.
	// Returns the IDs of the k nearest neighbors for each query vector and the
	// corresponding distances.
	Search(x []float32, k int64) (distances []float32, labels []string, err error)

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

	// Revision returns the loaded index revision.
	Revision() int64

	// Reload triggers a reloading action for an index, can be call manually or scheduled.
	Reload(ctx context.Context) error

	// Meta returns the loaded index meta info.
	Meta() IndexMeta

	// Shard returns the loaded shardInfo
	Shard() *Shard
}

func NewIndex(ctx context.Context, manifest *IndexManifest, shard Shard) (Index, error) {
	logger.CtxInfo(ctx, "loading shard", logger.String("shardKey", shard.ShardKey()))
	switch manifest.Meta.Type {
	case Faiss:
		return newFaissIndex(ctx, manifest, shard)
	case Mock:
		return newMockIndex(ctx, manifest, shard)
	default:
		logger.CtxError(
			ctx,
			"invalid index type",
			logger.String("type", manifest.Meta.Type.ToString()),
			logger.String("shardKey", shard.ShardKey()),
		)
		return nil, config.ErrInvalidIndexType
	}
}
