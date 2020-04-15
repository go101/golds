package f

type AAA interface {
	error
	mmm()
}


type apiClientExperimental interface {
	CheckpointAPIClient
}

// CheckpointAPIClient defines API client methods for the checkpoints
type CheckpointAPIClient interface {
	CheckpointCreate() error
	CheckpointDelete() error
	CheckpointList() (error)
}

