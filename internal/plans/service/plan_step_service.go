package service

import "github.com/Fazann/notedbook-backend/internal/plans/repo"

type PlanStepServiceImpl interface {
}

type planStepService struct {
	plStepRepo repo.PlanStepRepoImpl
}

func NewPlanStepService() *planStepService {
	return &planStepService{
		plStepRepo: repo.NewPlanStepRepo(),
	}
}
