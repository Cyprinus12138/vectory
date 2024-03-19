package engine

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

type IndexType string

const (
	Faiss IndexType = "faiss"
	ScaNN IndexType = "scann"
	RediS IndexType = "rs"
)

type ReloadMode string

const (
	Active  ReloadMode = "active"
	Passive ReloadMode = "passive"
)

type ScheduleType string

const (
	Cron      ScheduleType = "cron"
	Internal  ScheduleType = "interval"
	FixedTime ScheduleType = "fixed"
)

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

type IndexSource struct {
	Type     string
	Location string

	Reload ReloadSetting
}

type IndexManifest struct {
	Meta   IndexMeta
	Source IndexSource
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
}
