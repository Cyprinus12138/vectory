package engine

import "github.com/DataIntelligenceCrew/go-faiss"

type FaissIndex struct {
	name  string
	index faiss.Index
}

func (f *FaissIndex) Search(x []float32, k int64) (distances []float32, labels []int64, err error) {
	return f.index.Search(x, k)
}

func (f *FaissIndex) Delete() {
	f.index.Delete()
}

func (f *FaissIndex) VectorCount() int64 {
	return f.index.Ntotal()
}

func (f *FaissIndex) MetricType() string {
	return metricType[f.index.MetricType()]
}

func (f *FaissIndex) InputDim() int {
	return f.index.D()
}
