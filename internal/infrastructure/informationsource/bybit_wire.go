package informationsource

type bybitAnnouncementListWire struct {
	ReturnCode    int    `json:"retCode"`
	ReturnMessage string `json:"retMsg"`
	Result        struct {
		List []struct {
			Title         string `json:"title"`
			Url           string `json:"url"`
			DateTimestamp int64  `json:"dateTimestamp"`
		} `json:"list"`
	} `json:"result"`
}
