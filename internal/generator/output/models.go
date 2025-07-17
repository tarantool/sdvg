package output

const (
	BackupName       = "backup.json"
	CheckpointSuffix = "_checkpoint.json"
)

type Checkpoint struct {
	SavedRows uint64 `json:"saved_rows"`
}
