package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/finnan444/utils/transport/response"

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

// PreCheckNew немного доработанная версия без
func PreCheckNew(requestBody []byte, req *transport.KernelBaseRequest, token string) (*response.BasicResponse, logrus.Fields) {
	resp := transport.GetResponse()

	reqBody := &logrus.Fields{}
	if err := json.Unmarshal(requestBody, reqBody); err != nil {
		resp.Msg = "body not json"
		resp.Code = fasthttp.StatusBadRequest
		return resp, logrus.Fields{"error": err, "body": string(requestBody)}
	}

	if err := transport.DecodeJSONBodyNew(requestBody, req); err != nil {
		resp.Msg = "request decode error"
		resp.Code = fasthttp.StatusBadRequest
		return resp, logrus.Fields{"error": err, "body": reqBody}
	}

	if !transport.AuthenticateByTokenNew(req.Token, token) {
		resp.Msg = "unauthorized request"
		resp.Code = fasthttp.StatusUnauthorized
		return resp, logrus.Fields{"error": errors.New("unauthorized request"), "body": reqBody}
	}

	return resp, nil
}

// Elapsed можно вызывать в начале ф-ции defer Elapsed("functionName")
func Elapsed(what string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", what, time.Since(start))
	}
}
