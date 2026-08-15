package branch

type RestoreResult struct {
	Restoration Restoration
	Err         error
}

func (result RestoreResult) Restored() bool {
	return result.Err == nil
}
