package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

//go:generate go tool mockgen -source=i_hunt_board_repository.go -destination=mocks/mock_i_hunt_board_repository.go -package=mocks

type IHuntBoardRepository interface {
	// Rewrite makes the board exactly these entries in one transaction: each coin's row is overwritten or added, and
	// every coin not among them is removed. On failure the board is left as it was.
	Rewrite(executionContext context.Context, huntBoardEntries []entities.HuntBoardEntry) error
	// FindAll lists the board by confidence, highest first, then by coin symbol.
	FindAll(executionContext context.Context) ([]entities.HuntBoardEntry, error)
}
