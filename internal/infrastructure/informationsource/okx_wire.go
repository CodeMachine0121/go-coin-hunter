package informationsource

type okxAnnouncementListWire struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
	Data    []struct {
		Details []struct {
			Title string `json:"title"`
			Url   string `json:"url"`
			// PublishTime is milliseconds since the epoch, sent as a string.
			PublishTime string `json:"pTime"`
		} `json:"details"`
	} `json:"data"`
}
