package tasks

type TrackingVoteTask struct {
	// daoService       *services.DaoService
}

func NewTrackingVoteTask() *TrackingVoteTask {
	return &TrackingVoteTask{}
}

// Name returns the task name
func (t *TrackingVoteTask) Name() string {
	return "tracking-vote"
}

// Execute performs the DAO synchronization
func (t *TrackingVoteTask) Execute() error {
	return t.TrackingVote()
}

func (t *TrackingVoteTask) TrackingVote() error {
	return nil
}
