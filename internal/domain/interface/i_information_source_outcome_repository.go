package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

//go:generate go tool mockgen -source=i_information_source_outcome_repository.go -destination=mocks/mock_i_information_source_outcome_repository.go -package=mocks

type IInformationSourceOutcomeRepository interface {
	CreateAll(executionContext context.Context, informationSourceOutcomes []entities.InformationSourceOutcome) error
}
