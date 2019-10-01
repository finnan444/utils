package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/finnan444/utils/pool"
	"strconv"
	"time"

	"github.com/finnan444/utils/transport"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"

	"github.com/nyaruka/phonenumbers"
)

// SplitPhoneNumber парсит номер, дефолтная локаль - Россия
func SplitPhoneNumber(p *string) (code, phone string, err error) {
	var cc, nn string

	if p == nil {
		return cc, nn, errors.New("input phone is nil")
	}

	num, err := phonenumbers.Parse(*p, "RU")
	if err != nil {
		return cc, nn, err
	}

	if num.CountryCode == nil {
		return cc, nn, errors.New("country code is null")
	}

	cc = strconv.Itoa(int(*num.CountryCode))
	if num.NationalNumber == nil {
		return "", nn, errors.New("national number is null")
	}

	nn = strconv.Itoa(int(*num.NationalNumber))

	return cc, nn, nil
}

// PreCheck проверяет что запрос корректный с точки зрения структуры KernelBaseRequest и сверяет токен
func PreCheck(ctx *fasthttp.RequestCtx, req *transport.KernelBaseRequest, logger2 *logrus.Logger, token string) bool {
	reqBody := &logrus.Fields{}
	if err := json.Unmarshal(ctx.Request.Body(), reqBody); err != nil {
		logger2.WithFields(logrus.Fields{"error": err, "body": string(ctx.Request.Body())}).Warn("body not json")
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return false
	}

	if err := transport.DecodeJSONBody(ctx, req); err != nil {
		logger2.WithFields(logrus.Fields{"error": err, "body": reqBody}).Warn("request decode error")
		return false
	}

	if !transport.AuthenticateByToken(ctx, req.Token, token) {
		logger2.WithFields(logrus.Fields{"body": reqBody}).Warn("unauthorized request")
		return false
	}

	return true
}

// PreCheckNew
func PreCheckNew(ctx *fasthttp.RequestCtx, req *transport.KernelBaseRequest, logger2 *logrus.Logger, token string) pool.Reusable {
	response := transport.GetResponse()

	reqBody := &logrus.Fields{}
	if err := json.Unmarshal(ctx.Request.Body(), reqBody); err != nil {
		response.Msg = "body not json"
		response.Code = fasthttp.StatusBadRequest
		response.Payload = logrus.Fields{"error": err, "body": string(ctx.Request.Body())}
		return response
	}

	if err := transport.DecodeJSONBody(ctx, req); err != nil {
		response.Msg = "request decode error"
		response.Code = fasthttp.StatusBadRequest
		response.Payload = logrus.Fields{"error": err, "body": reqBody}
		return response
	}

	if !transport.AuthenticateByTokenNew(req.Token, token) {
		response.Msg = "unauthorized request"
		response.Code = fasthttp.StatusUnauthorized
		response.Payload = logrus.Fields{"body": reqBody}
		return response
	}

	return response
}

// Elapsed можно вызывать в начале ф-ции defer Elapsed("functionName")
func Elapsed(what string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", what, time.Since(start))
	}
}
