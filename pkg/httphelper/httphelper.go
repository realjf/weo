package httphelper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx"
	"github.com/julienschmidt/httprouter"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"syscall"
	"time"
	"weo/pkg/ctxhelper"
	"weo/pkg/dialer"
	"weo/pkg/random"
)

type ErrorCode string

var RetryClient = &http.Client{Transport: &http.Transport{Dial: dialer.Retry.Dial}}

const (
	NotFoundErrorCode           ErrorCode = "not_found"
	ObjectNotFoundErrorCode     ErrorCode = "object_not_found"
	ObjectExistsErrorCode       ErrorCode = "object_exists"
	ConflictErrorCode           ErrorCode = "conflict"
	SyntaxErrorCode             ErrorCode = "syntax_error"
	ValidationErrorCode         ErrorCode = "validation_error"
	PreconditionFailedErrorCode ErrorCode = "precondition_failed"
	UnauthorizedErrorCode       ErrorCode = "unauthorized"
	UnknownErrorCode            ErrorCode = "unknown_error"
	RatelimitedErrorCode        ErrorCode = "ratelimited"
	ServiceUnavailableErrorCode ErrorCode = "service_unavailable"
	RequestBodyTooBigErrorCode  ErrorCode = "request_body_too_big"
)

var ErrRequestBodyTooBig = errors.New("httphelper: request body too big")

var errorResponseCodes = map[ErrorCode]int{
	NotFoundErrorCode:           404,
	ObjectNotFoundErrorCode:     404,
	ObjectExistsErrorCode:       409,
	ConflictErrorCode:           409,
	PreconditionFailedErrorCode: 412,
	SyntaxErrorCode:             400,
	ValidationErrorCode:         400,
	RequestBodyTooBigErrorCode:  400,
	UnauthorizedErrorCode:       401,
	UnknownErrorCode:            500,
	RatelimitedErrorCode:        429,
	ServiceUnavailableErrorCode: 503,
}

type JSONError struct {
	Code    ErrorCode       `json:"code"`
	Message string          `json:"message"`
	Detail  json.RawMessage `json:"detail,omitempty"`
	Retry   bool            `json:"retry"`
}

func isJSONErrorWithCode(err error, code ErrorCode) bool {
	e, ok := err.(JSONError)
	return ok && e.Code == code
}

func IsObjectNotFoundError(err error) bool {
	return isJSONErrorWithCode(err, ObjectNotFoundErrorCode)
}

func IsObjectExistsError(err error) bool {
	return isJSONErrorWithCode(err, ObjectExistsErrorCode)
}

func IsPreconditionFailedError(err error) bool {
	return isJSONErrorWithCode(err, PreconditionFailedErrorCode)
}

func IsValidationError(err error) bool {
	return isJSONErrorWithCode(err, ValidationErrorCode)
}

func IsRetryableError(err error) bool {
	e, ok := err.(JSONError)
	return ok && e.Retry
}

var CORSAllowAll = &cors.Options{
	AllowAllOrigins:  true,
	AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
	AllowHeaders:     []string{"Auth-Key", "Authorization", "Accept", "Content-Type", "If-Match", "If-None-Match", "X-GRPC-Web"},
	ExposeHeaders:    []string{"ETag", "Content-Disposition"},
	AllowCredentials: true,
	MaxAge:           time.Hour,
}

type Handler interface {
	ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request)
}

type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (f HandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

func WrapHandler(handler HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		ctx := contextFromResponseWriter(w)
		ctx = ctxhelper.NewContextParams(ctx, params)
		handler.ServeHTTP(ctx, w, req)
	}
}

func ContextInjector(componentName string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		reqID := req.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = random.UUID()
		}
		ctx := ctxhelper.NewContextRequestID(context.Background(), reqID)
		ctx = ctxhelper.NewContextComponentName(ctx, componentName)
		rw := NewResponseWriter(w, ctx)
		handler.ServeHTTP(rw, req)
	})
}

func contextFromResponseWriter(w http.ResponseWriter) context.Context {
	ctx := w.(*ResponseWriter).Context()
	return ctx
}

func (jsonError JSONError) Error() string {
	return fmt.Sprintf("%s: %s", jsonError.Code, jsonError.Message)
}

func logError(w http.ResponseWriter, err error) {
	if rw, ok := w.(*ResponseWriter); ok {
		logger, _ := ctxhelper.LoggerFromContext(rw.Context())
		logger.Error(err.Error())
	} else {
		log.Println(err)
	}
}

func buildJSONError(err error) *JSONError {
	jsonError := &JSONError{
		Code:    UnknownErrorCode,
		Message: "Something went wrong",
	}
	if err == ErrRequestBodyTooBig {
		return &JSONError{
			Code:    RequestBodyTooBigErrorCode,
			Message: "The provided request body is too big",
		}
	}
	switch v := err.(type) {
	case *json.SyntaxError, *json.UnmarshalTypeError:
		jsonError = &JSONError{
			Code:    SyntaxErrorCode,
			Message: "The provided JSON input is invalid",
		}
	case pgx.PgError, *net.OpError, syscall.Errno:
		jsonError.Retry = true
	case JSONError:
		jsonError = &v
	case *JSONError:
		jsonError = v
	default:
		if err == pgx.ErrDeadConn {
			jsonError.Retry = true
		}
	}
	return jsonError
}

func Error(w http.ResponseWriter, err error) {
	if rw, ok := w.(*ResponseWriter); !ok || (ok && rw.Status() == 0) {
		jsonError := buildJSONError(err)
		if jsonError.Code == UnknownErrorCode {
			logError(w, err)
		}
		responseCode, ok := errorResponseCodes[jsonError.Code]
		if !ok {
			responseCode = 500
		}
		JSON(w, responseCode, jsonError)
	} else {
		logError(w, err)
	}
}

func ObjectNotFoundError(w http.ResponseWriter, message string) {
	Error(w, JSONError{Code: ObjectNotFoundErrorCode, Message: message})
}

func ObjectExistsErr(message string) error {
	return JSONError{Code: ObjectExistsErrorCode, Message: message}
}

func ObjectExistsError(w http.ResponseWriter, message string) {
	Error(w, ObjectExistsErr(message))
}

func ConflictError(w http.ResponseWriter, message string) {
	Error(w, JSONError{Code: ConflictErrorCode, Message: message})
}

func PreconditionFailedErr(message string) error {
	return JSONError{Code: PreconditionFailedErrorCode, Message: message}
}

func ServiceUnavailableError(w http.ResponseWriter, message string) {
	Error(w, JSONError{Code: ServiceUnavailableErrorCode, Message: message, Retry: true})
}

func ValidationError(w http.ResponseWriter, field, message string) {
	err := JSONError{Code: ValidationErrorCode, Message: message}
	if field != "" {
		err.Message = fmt.Sprintf("%s %s", field, message)
		err.Detail, _ = json.Marshal(map[string]string{"field": field})
	}
	Error(w, err)
}

func JSON(w http.ResponseWriter, status int, v interface{}) {
	// Encode nil slices as `[]` instead of `null`
	if rv := reflect.ValueOf(v); rv.Type().Kind() == reflect.Slice && rv.IsNil() {
		v = []struct{}{}
	}

	var result []byte
	var err error
	result, err = json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(result)
}

func DecodeJSON(req *http.Request, i interface{}) error {
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		return JSONError{Code: ValidationErrorCode, Message: "Content-Type must be application/json"}
	}
	dec := json.NewDecoder(req.Body)
	dec.UseNumber()
	return dec.Decode(i)
}


