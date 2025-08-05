package tasks

import "github.com/ringecosystem/degov-apps/services"

type ProposalTrackingTask struct {
	daoService *services.DaoService
}

func NewProposalTrackingTask() *ProposalTrackingTask {
	return &ProposalTrackingTask{
		daoService: services.NewDaoService(),
	}
}

// Name returns the task name
func (t *ProposalTrackingTask) Name() string {
	return "proposal-tracking-sync"
}

// Execute performs the DAO synchronization
func (t *ProposalTrackingTask) Execute() error {
	return t.TrackingProposal()
}

// TrackingProposal tracks proposals for DAOs
func (t *ProposalTrackingTask) TrackingProposal() error {
	return nil
}
