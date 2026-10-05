package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_information_source_proxy.go -destination=mocks/mock_i_information_source_proxy.go -package=mocks

// IInformationSourceProxy is one free information source; every source has its own implementation and the
// composition root injects them as a list, so adding a source never touches the discovery rules.
type IInformationSourceProxy interface {
	SourceName() string
	// FetchInformationItems returns at most itemLimit of the source's newest messages, normalized.
	FetchInformationItems(executionContext context.Context, itemLimit int) ([]vo.InformationItemVo, error)
}
