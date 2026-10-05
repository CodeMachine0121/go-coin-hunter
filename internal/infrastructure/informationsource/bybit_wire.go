package informationsource

type bybitAnnouncementListWire struct {
	// ReturnCode is a pointer so an answer without it is told apart from success (0).
	ReturnCode    *int   `json:"retCode"`
	ReturnMessage string `json:"retMsg"`
	Result        struct {
		List []struct {
			Title         string `json:"title"`
			Url           string `json:"url"`
			DateTimestamp int64  `json:"dateTimestamp"`
		} `json:"list"`
	} `json:"result"`
}
