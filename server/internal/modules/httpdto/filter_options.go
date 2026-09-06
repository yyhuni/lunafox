package httpdto

type FilterOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Count int64  `json:"count,omitempty"`
}

type FilterOptionsResponse struct {
	Results []FilterOption `json:"results"`
}
