package transport

import "sync"

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

// KernelBaseRequest базовая сигнатура запросов между сервисами
type KernelBaseRequest struct {
	Token   string      `json:"token"`
	Payload interface{} `json:"payload"`
}

var requestPool = sync.Pool{New: func() interface{} { return &KernelBaseRequest{} }}

func (s *KernelBaseRequest) Reuse() {
	s.Token, s.Payload = "", nil
	requestPool.Put(s)
}

// GetKernelBaseReq берет из пула стандартный запрос
func GetKernelBaseReq() *KernelBaseRequest {
	return requestPool.Get().(*KernelBaseRequest)
}
