package informationsource

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// OkxAnnouncementInformationSourceProxy reads OKX's new-listing announcements.
type OkxAnnouncementInformationSourceProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewOkxAnnouncementInformationSourceProxy(httpClient *http.Client, baseUrl string) *OkxAnnouncementInformationSourceProxy {
	return &OkxAnnouncementInformationSourceProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (okxAnnouncementInformationSourceProxy *OkxAnnouncementInformationSourceProxy) SourceName() string {
	return "okxAnnouncement"
}

func (okxAnnouncementInformationSourceProxy *OkxAnnouncementInformationSourceProxy) FetchInformationItems(
	executionContext context.Context, itemLimit int,
) ([]vo.InformationItemVo, error) {
	announcementList, fetchError := getJson[okxAnnouncementListWire](executionContext,
		okxAnnouncementInformationSourceProxy.httpClient,
		okxAnnouncementInformationSourceProxy.baseUrl+"/api/v5/support/announcements?annType=announcements-new-listings")
	if fetchError != nil {
		return nil, fetchError
	}
	if announcementList.Code != "0" {
		return nil, fmt.Errorf("okx announcements answered %s: %s", announcementList.Code, announcementList.Message)
	}
	if announcementList.Data == nil {
		return nil, fmt.Errorf("okx announcements answered without an announcement list")
	}

	informationItems := []vo.InformationItemVo{}
	for _, page := range announcementList.Data {
		for _, announcement := range page.Details {
			if len(informationItems) == itemLimit {
				return informationItems, nil
			}
			publishMilliseconds, parseError := strconv.ParseInt(announcement.PublishTime, 10, 64)
			if parseError != nil {
				return nil, fmt.Errorf("okx announcement %q has an unreadable publish time %q", announcement.Title, announcement.PublishTime)
			}
			publishedAt := time.UnixMilli(publishMilliseconds).UTC()
			informationItems = append(informationItems, vo.InformationItemVo{
				SourceName:         okxAnnouncementInformationSourceProxy.SourceName(),
				ExternalIdentifier: announcement.Url,
				Title:              announcement.Title,
				Link:               announcement.Url,
				PublishedAt:        &publishedAt,
			})
		}
	}

	return informationItems, nil
}
