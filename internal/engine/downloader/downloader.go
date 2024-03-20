package downloader

type IndexSource struct {
	Type     string
	Location string
}

type Downloader interface {
	Download() (localPath string, revision int64, err error)
	DownloadUpdate(currentRevision int64) (localPath string, revision int64, err error)
}

func NewDownLoader(source *IndexSource) (Downloader, error) {
	return nil, nil
}
