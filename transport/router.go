package transport

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/finnan444/utils/math/ints"
	"github.com/valyala/fasthttp"
)

var (
	postRoutes       = make(map[string]RouterFunc)
	postSimpleRoutes = make(map[string]fasthttp.RequestHandler)
	postRegRoutes    = make(map[*regexp.Regexp]RouterFunc)
	getRoutes        = make(map[string]RouterFunc)
	getSimpleRoutes  = make(map[string]fasthttp.RequestHandler)
	getRegRoutes     = make(map[*regexp.Regexp]RouterFunc)
	clientsPool      = sync.Pool{
		New: func() interface{} {
			return &fasthttp.Client{}
		},
	}
	timings      = make(map[string]*median)
	timingsReg   = make(map[*regexp.Regexp]*median)
	logger       = log.New(os.Stdout, "\n-----------------------------\n", log.LstdFlags)
	pingResponse = []byte("OK")
)

func init() {
	AddGetRoute("/internal/stats", handlerInternalStats)
	AddGetRouteSimple("/internal/stats", handlerInternalStatsSimple)
	AddGetRoute("/ping", ping)
	AddGetRouteSimple("/ping", pingSimple)
}

type median struct {
	sync.Mutex
	Min, Max, Total, Count time.Duration
}

func (m *median) Update(d time.Duration) {
	m.Lock()
	if m.Min == 0 || m.Min > d {
		m.Min = d
	}
	if m.Max < d {
		m.Max = d
	}
	m.Total += d
	m.Count++
	m.Unlock()
}

func (m *median) String() string {
	if m.Count > 0 {
		return fmt.Sprintf(": {\"min\":%v, \"max\":%v, \"med\":%v}\n", m.Min, m.Max, m.Total/m.Count)
	}
	return ": Not enough stats\n"
}

// SetLogger sets new logger
func SetLogger(lgr *log.Logger) {
	logger = lgr
}

// RouterFunc router function
type RouterFunc func(*fasthttp.RequestCtx, time.Time, ...string)

// AddGetRoute adds get route
func AddGetRoute(path string, handler RouterFunc) {
	getRoutes[path] = handler
	timings["[GET] "+path] = &median{}
}

func AddGetRouteSimple(path string, handler fasthttp.RequestHandler) {
	getSimpleRoutes[path] = handler
	timings["[GET] "+path] = &median{}
}

func AddPostRouteSimple(path string, handler fasthttp.RequestHandler) {
	postSimpleRoutes[path] = handler
	timings["[POST] "+path] = &median{}
}

// AddGetRegexpRoute adds get route. For example /accounts/([0-9]+)/suggest/.
//The result of regex will be passed as s third parameter in router.RouterFunc
func AddGetRegexpRoute(path string, handler RouterFunc) {
	if re, err := regexp.Compile(path); err == nil {
		getRegRoutes[re] = handler
		timingsReg[re] = &median{}
	}
}

// AddPostRoute adds post route
func AddPostRoute(path string, handler RouterFunc) {
	postRoutes[path] = handler
	timings["[POST] "+path] = &median{}
}

// AddPostRegexpRoute adds post route
func AddPostRegexpRoute(path string, handler RouterFunc) {
	if re, err := regexp.Compile(path); err == nil {
		postRegRoutes[re] = handler
		timingsReg[re] = &median{}
	}
}

// ProcessRouting returns router
// Обрабатывает только GET и POST
// логи обрезаются только у POST запросов
func ProcessRouting(server PathesLogger) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		now := time.Now()
		path := string(ctx.Path())
		reqID := ctx.ID()
		if ctx.IsPost() {
			body := ctx.PostBody()
			if logFlag := server.GetLogFlag(path); (logFlag & ToLog) != 0 {
				if (logFlag & FullLog) != 0 {
					logger.Printf("[POST %s %d][Request] %s\n", path, reqID, body)
				} else {
					logger.Printf("[POST %s %d][Request] %s\n", path, reqID, body[:ints.MinInt(len(body), 255)])
				}
			}
			if handler, ok := postRoutes[path]; ok {
				handler(ctx, now)
				timings["[POST] "+path].Update(time.Since(now))
			} else {
				for k, v := range postRegRoutes {
					adds := k.FindStringSubmatch(path)
					if len(adds) > 1 {
						v(ctx, now, adds[1:]...)
						return
					}
				}
				ctx.Error("Not found", fasthttp.StatusNotFound)
			}
		} else if ctx.IsGet() {
			if logFlag := server.GetLogFlag(path); (logFlag & ToLog) != 0 {
				logger.Printf("[GET %s %d][Request] %s\n", path, reqID, ctx.QueryArgs().QueryString())
			}
			if handler, ok := getRoutes[path]; ok {
				handler(ctx, now)
				timings["[GET] "+path].Update(time.Since(now))
			} else {
				for k, v := range getRegRoutes {
					adds := k.FindStringSubmatch(path)
					if len(adds) > 1 {
						v(ctx, now, adds[1:]...)
						timingsReg[k].Update(time.Since(now))
						return
					}
				}
				ctx.Error("Not found", fasthttp.StatusNotFound)
			}
		} else {
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}
}

// ProcessSimpleRouting тоже самое что ProcessRouting, только без логирования
func ProcessSimpleRouting() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		now := time.Now()
		path := string(ctx.Path())
		if ctx.IsPost() {
			if handler, ok := postRoutes[path]; ok {
				handler(ctx, now)
				timings["[POST] "+path].Update(time.Since(now))
			} else {
				for k, v := range postRegRoutes {
					adds := k.FindStringSubmatch(path)
					if len(adds) > 1 {
						v(ctx, now, adds[1:]...)
						return
					}
				}
				ctx.Error("Not found", fasthttp.StatusNotFound)
			}
		} else if ctx.IsGet() {
			if handler, ok := getRoutes[path]; ok {
				handler(ctx, now)
				timings["[GET] "+path].Update(time.Since(now))
			} else {
				for k, v := range getRegRoutes {
					adds := k.FindStringSubmatch(path)
					if len(adds) > 1 {
						v(ctx, now, adds[1:]...)
						timingsReg[k].Update(time.Since(now))
						return
					}
				}
				ctx.Error("Not found", fasthttp.StatusNotFound)
			}
		} else {
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}
}

// ProcessStandardRouting работает с хендлерами, соотв стандартной сигнатуре fasthttp
// без regexp routes + обрабатывает только GET и POST
// логи обрезаются только у POST запросов
func ProcessStandardRouting(server PathesLogger) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		switch string(ctx.Method()) {
		case "POST":
			body := ctx.PostBody()
			if logFlag := server.GetLogFlag(path); (logFlag & ToLog) != 0 {
				if (logFlag & FullLog) != 0 {
					logger.Printf("[POST %s %d][Request] %s\n", path, ctx.ID(), body)
				} else {
					logger.Printf("[POST %s %d][Request] %s\n", path, ctx.ID(), body[:ints.MinInt(len(body), 255)])
				}
			}
			if handler, ok := postSimpleRoutes[path]; ok {
				handler(ctx)
				timings["[POST] "+path].Update(time.Since(ctx.Time()))
			} else {
				ctx.Error("Not found", fasthttp.StatusNotFound)
			}
		case "GET":
			if logFlag := server.GetLogFlag(path); (logFlag & ToLog) != 0 {
				logger.Printf("[GET %s %d][Request] %s\n", path, ctx.ID(), ctx.QueryArgs().QueryString())
			}
			if handler, ok := getSimpleRoutes[path]; ok {
				handler(ctx)
				timings["[GET] "+path].Update(time.Since(ctx.Time()))
			} else {
				ctx.Error("Not found", fasthttp.StatusNotFound)
			}
		default:
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}
}

// GetHTTPClient returns client from pool
func GetHTTPClient() *fasthttp.Client {
	return clientsPool.Get().(*fasthttp.Client)
}

// PutHTTPClient returns client to pool
func PutHTTPClient(client *fasthttp.Client) {
	clientsPool.Put(client)
}

func handlerInternalStats(ctx *fasthttp.RequestCtx, now time.Time, adds ...string) {
	for k, v := range timings {
		ctx.WriteString(k)
		ctx.WriteString(v.String())
	}
	for k, v := range timingsReg {
		ctx.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
}

func handlerInternalStatsSimple(ctx *fasthttp.RequestCtx) {
	for k, v := range timings {
		ctx.WriteString(k)
		ctx.WriteString(v.String())
	}
	for k, v := range timingsReg {
		ctx.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
}

func pingSimple(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain; charset=utf-8")
	ctx.SetBody(pingResponse)
}

func ping(ctx *fasthttp.RequestCtx, now time.Time, adds ...string) {
	ctx.SetContentType("text/plain; charset=utf-8")
	ctx.SetBody(pingResponse)
}
