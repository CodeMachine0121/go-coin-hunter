package informationsource

import (
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"net/http"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// BybitAnnouncementInformationSourceProxy reads Bybit's new-listing announcements.
type BybitAnnouncementInformationSourceProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewBybitAnnouncementInformationSourceProxy(httpClient *http.Client, baseUrl string) *BybitAnnouncementInformationSourceProxy {
	return &BybitAnnouncementInformationSourceProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (bybitAnnouncementInformationSourceProxy *BybitAnnouncementInformationSourceProxy) SourceName() string {
	return "bybitAnnouncement"
}

func (bybitAnnouncementInformationSourceProxy *BybitAnnouncementInformationSourceProxy) FetchInformationItems(
	executionContext context.Context, itemLimit int,
) ([]vo.InformationItemVo, error) {
	announcementList, fetchError := utilities.GetJson[bybitAnnouncementListWire](executionContext,
		bybitAnnouncementInformationSourceProxy.httpClient,
		fmt.Sprintf("%s/v5/announcements/index?locale=en-US&type=new_crypto&limit=%d",
			bybitAnnouncementInformationSourceProxy.baseUrl, itemLimit))
	if fetchError != nil {
		return nil, fetchError
	}
	if announcementList.ReturnCode == nil {
		return nil, fmt.Errorf("bybit announcements answered without a return code")
	}
	if *announcementList.ReturnCode != 0 {
		return nil, fmt.Errorf("bybit announcements answered %d: %s", *announcementList.ReturnCode, announcementList.ReturnMessage)
	}
	if announcementList.Result.List == nil {
		return nil, fmt.Errorf("bybit announcements answered without an announcement list")
	}

	informationItems := []vo.InformationItemVo{}
	for _, announcement := range announcementList.Result.List {
		if len(informationItems) == itemLimit {
			break
		}
		publishedAt := time.UnixMilli(announcement.DateTimestamp).UTC()
		informationItems = append(informationItems, vo.InformationItemVo{
			SourceName:         bybitAnnouncementInformationSourceProxy.SourceName(),
			ExternalIdentifier: announcement.Url,
			Title:              announcement.Title,
			Link:               announcement.Url,
			PublishedAt:        &publishedAt,
		})
	}

	return informationItems, nil
}
