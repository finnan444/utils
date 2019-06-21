package transport

// UTM универсальная структура для хранения utm-меток
type UTM struct {
	UtmSource   *string `json:"utm_source,omitempty"`
	UtmMedium   *string `json:"utm_medium,omitempty"`
	UtmCampaign *string `json:"utm_campaign,omitempty"`
	UtmContent  *string `json:"utm_content,omitempty"`
	UtmTerm     *string `json:"utm_term,omitempty"`
}

// RequestInfo полная информация о запросе
type RequestInfo struct {
	HeadersOrigin *string `json:"headers_origin,omitempty"`
	BodyOrigin    *string `json:"body_origin,omitempty"`
	URIOrigin     *string `json:"uri_origin,omitempty"`
}
