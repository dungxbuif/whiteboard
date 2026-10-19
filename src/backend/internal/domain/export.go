package domain

const ExportAssetMaxBytes int64 = 10 * 1024 * 1024

type ExportObjectReader interface {
	ReadExportObject(Context, string, string, int64) ([]byte, error)
}
type ExportAsset struct {
	MimeType string `json:"mimeType"`
	Data     []byte `json:"data"`
}
