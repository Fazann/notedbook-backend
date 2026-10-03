package service

import "github.com/Fazann/notedbook-backend/internal/plans/repo"

type PlanServiceImpl interface{}

type planService struct {
	planRepo repo.PlanRepoImpl
}

func NewPlanService() *planService {
	return &planService{
		planRepo: repo.NewPlanRepo(),
	}
}
