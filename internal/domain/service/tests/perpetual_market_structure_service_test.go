package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestFindMarketStructuresAsksABoundedNumberOfCoinsAtOnce(t *testing.T) {
	controller := gomock.NewController(t)
	exchange := mocks.NewMockIPerpetualMarketStructureProxy(controller)
	inFlight, mostInFlight := 0, 0
	inFlightLock := sync.Mutex{}
	exchange.EXPECT().FindMarketStructure(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, coinSymbol string) (vo.PerpetualMarketStructureVo, bool, error) {
			inFlightLock.Lock()
			inFlight++
			mostInFlight = max(mostInFlight, inFlight)
			inFlightLock.Unlock()
			time.Sleep(20 * time.Millisecond)
			inFlightLock.Lock()
			inFlight--
			inFlightLock.Unlock()
			return vo.PerpetualMarketStructureVo{ExchangeName: coinSymbol}, coinSymbol != "GONE", nil
		}).Times(6)
	marketStructureService := service.NewPerpetualMarketStructureService([]domaininterface.IPerpetualMarketStructureProxy{exchange}, time.Second, 2)

	marketStructures := marketStructureService.FindMarketStructures(context.Background(), []string{"PENGU", "STRK", "ARB", "OP", "TIA", "GONE"})

	assert.Equal(t, 2, mostInFlight)
	assert.Len(t, marketStructures, 5)
	assert.Equal(t, "PENGU", marketStructures["PENGU"].ExchangeName)
	assert.NotContains(t, marketStructures, "GONE")
}

func TestFindMarketStructuresPassesOverAFailingExchangeAndStopsWhenTheContextEnds(t *testing.T) {
	controller := gomock.NewController(t)
	binance := mocks.NewMockIPerpetualMarketStructureProxy(controller)
	bybit := mocks.NewMockIPerpetualMarketStructureProxy(controller)
	binance.EXPECT().FindMarketStructure(gomock.Any(), "PENGU").Return(vo.PerpetualMarketStructureVo{}, false, errors.New("429 too many requests"))
	bybit.EXPECT().FindMarketStructure(gomock.Any(), "PENGU").Return(vo.PerpetualMarketStructureVo{ExchangeName: "Bybit"}, true, nil)
	marketStructureService := service.NewPerpetualMarketStructureService([]domaininterface.IPerpetualMarketStructureProxy{binance, bybit}, time.Second, 0)

	assert.Equal(t, "Bybit", marketStructureService.FindMarketStructures(context.Background(), []string{"PENGU"})["PENGU"].ExchangeName)

	ended, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Empty(t, marketStructureService.FindMarketStructures(ended, []string{"STRK"}), "no exchange is asked once the context has ended")
}
