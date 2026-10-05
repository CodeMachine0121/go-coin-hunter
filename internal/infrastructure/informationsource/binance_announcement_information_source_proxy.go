package informationsource

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// binanceNewListingCatalogID is Binance's "New Cryptocurrency Listing" announcement catalog.
const binanceNewListingCatalogID = 48

// BinanceAnnouncementInformationSourceProxy reads Binance's new-listing announcements.
type BinanceAnnouncementInformationSourceProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewBinanceAnnouncementInformationSourceProxy(httpClient *http.Client, baseUrl string) *BinanceAnnouncementInformationSourceProxy {
	return &BinanceAnnouncementInformationSourceProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (binanceAnnouncementInformationSourceProxy *BinanceAnnouncementInformationSourceProxy) SourceName() string {
	return "binanceAnnouncement"
}

func (binanceAnnouncementInformationSourceProxy *BinanceAnnouncementInformationSourceProxy) FetchInformationItems(
	executionContext context.Context, itemLimit int,
) ([]vo.InformationItemVo, error) {
	announcementList, fetchError := getJson[binanceAnnouncementListWire](executionContext,
		binanceAnnouncementInformationSourceProxy.httpClient,
		fmt.Sprintf("%s/bapi/composite/v1/public/cms/article/list/query?type=1&catalogId=%d&pageNo=1&pageSize=%d",
			binanceAnnouncementInformationSourceProxy.baseUrl, binanceNewListingCatalogID, itemLimit))
	if fetchError != nil {
		return nil, fetchError
	}
	if announcementList.Code != "000000" || len(announcementList.Data.Catalogs) == 0 {
		return nil, fmt.Errorf("binance announcements answered code %q without a catalog", announcementList.Code)
	}
	if announcementList.Data.Catalogs[0].Articles == nil {
		return nil, fmt.Errorf("binance announcements answered a catalog without an article list")
	}

	informationItems := []vo.InformationItemVo{}
	for _, article := range announcementList.Data.Catalogs[0].Articles {
		if len(informationItems) == itemLimit {
			break
		}
		publishedAt := time.UnixMilli(article.ReleaseDate).UTC()
		informationItems = append(informationItems, vo.InformationItemVo{
			SourceName:         binanceAnnouncementInformationSourceProxy.SourceName(),
			ExternalIdentifier: article.Code,
			Title:              article.Title,
			Link:               "https://www.binance.com/en/support/announcement/" + article.Code,
			PublishedAt:        &publishedAt,
		})
	}

	return informationItems, nil
}
