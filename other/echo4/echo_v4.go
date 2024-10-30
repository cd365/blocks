package echo4

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cd365/blocks/log"
	"github.com/cd365/g"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	tzh "github.com/go-playground/validator/v10/translations/zh"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Context struct {
	echo.Context `json:"-"`

	status int // http响应状态码

	/* 响应数据 */
	RespCode  int         `json:"code"`            // 状态码 0:成功, !0:不成功
	RespMsg   string      `json:"msg"`             // 描述语
	RespData  interface{} `json:"data,omitempty"`  // 数据
	RespTotal *int64      `json:"total,omitempty"` // 数据总条数
}

func (s *Context) json() error {
	return s.Context.JSON(s.status, s)
}

func (s *Context) SetStatus(status int) *Context {
	s.status = status
	return s
}

func (s *Context) SetCode(code int) *Context {
	s.RespCode = code
	return s
}

func (s *Context) SetMsg(msg string) *Context {
	s.RespMsg = msg
	return s
}

func (s *Context) SetData(data any) *Context {
	s.RespData = data
	return s
}

func (s *Context) SetTotal(total int64) *Context {
	s.RespTotal = &total
	return s
}

func (s *Context) Ok() error {
	return s.json()
}

func (s *Context) Bad(err error) error {
	s.SetStatus(http.StatusBadRequest).SetCode(1)
	return s.SetMsg(err.Error()).json()
}

func (s *Context) Err(err error) error {
	s.SetCode(http.StatusInternalServerError)
	return s.SetMsg(err.Error()).json()
}

func (s *Context) Failed(msg string) error {
	s.SetCode(1)
	return s.SetMsg(msg).json()
}

func (s *Context) Data(data interface{}, failed error, err error) error {
	if err != nil {
		return s.Err(err)
	}
	if failed != nil {
		return s.Failed(failed.Error())
	}
	return s.SetData(data).Ok()
}

func (s *Context) TotalData(total int64, data interface{}, failed error, err error) error {
	if err != nil {
		return s.Err(err)
	}
	if failed != nil {
		return s.Failed(failed.Error())
	}
	return s.SetTotal(total).SetData(data).Ok()
}

func (s *Context) Result(failed error, err error) error {
	return s.Data(nil, failed, err)
}

const (
	defaultResponseStatus = http.StatusOK
	defaultResponseCode   = 0
	defaultResponseMsg    = "success"
)

const (
	RoutePathLevelNormal    = "normal"
	RoutePathLevelPrimary   = "primary"
	RoutePathLevelImportant = "important"
)

type Route struct {
	buffer  *sync.Pool
	context *sync.Pool

	level      map[string]string
	levelMutex *g.Mutex

	name      map[string]string
	nameMutex *g.Mutex
}

func NewRoute() *Route {
	return &Route{
		buffer:  &sync.Pool{New: func() interface{} { return bytes.NewBuffer(nil) }},
		context: &sync.Pool{New: func() interface{} { return &Context{} }},

		level:      make(map[string]string, 32),
		levelMutex: g.NewMutex(),

		name:      make(map[string]string, 32),
		nameMutex: g.NewMutex(),
	}
}

func (s *Route) GetBuffer() *bytes.Buffer {
	return s.buffer.Get().(*bytes.Buffer)
}

func (s *Route) PutBuffer(b *bytes.Buffer) {
	b.Reset()
	s.buffer.Put(b)
}

func (s *Route) GetContext(c echo.Context) *Context {
	resp := s.context.Get().(*Context)
	resp.Context = c
	resp.status = defaultResponseStatus
	resp.RespCode = defaultResponseCode
	resp.RespMsg = defaultResponseMsg
	resp.RespData = nil
	return resp
}

func (s *Route) PutContext(c *Context) {
	c.Context = nil
	c.status = defaultResponseStatus
	c.RespCode = defaultResponseCode
	c.RespMsg = defaultResponseMsg
	c.RespData = nil
	s.context.Put(c)
}

func (s *Route) register(handler func(c *Context) error) func(c echo.Context) error {
	return func(c echo.Context) error {
		resp := s.GetContext(c)
		defer s.PutContext(resp)
		return handler(resp)
	}
}

func (s *Route) Register(
	group *echo.Group,
	method string,
	routePathLevel string,
	routePathName string,
	routePath string,
	handler func(c *Context) error,
	m ...echo.MiddlewareFunc,
) *Route {
	var route *echo.Route
	switch method {
	case http.MethodGet:
		route = group.GET(routePath, s.register(handler), m...)
	case http.MethodHead:
		route = group.HEAD(routePath, s.register(handler), m...)
	case http.MethodPost:
		route = group.POST(routePath, s.register(handler), m...)
	case http.MethodPut:
		route = group.PUT(routePath, s.register(handler), m...)
	case http.MethodPatch:
		route = group.PATCH(routePath, s.register(handler), m...)
	case http.MethodDelete:
		route = group.DELETE(routePath, s.register(handler), m...)
	case http.MethodConnect:
		route = group.CONNECT(routePath, s.register(handler), m...)
	case http.MethodOptions:
		route = group.OPTIONS(routePath, s.register(handler), m...)
	case http.MethodTrace:
		route = group.TRACE(routePath, s.register(handler), m...)
	default:
	}
	if route == nil {
		return s
	}
	if routePathLevel == "" {
		routePathLevel = RoutePathLevelNormal
	}
	route.Name = routePathName
	s.levelMutex.WithLock(func() { s.level[routePath] = routePathLevel })
	s.nameMutex.WithLock(func() { s.name[routePath] = routePathName })
	return s
}

func (s *Route) GET(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodGet, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) POST(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodPost, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) PUT(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodPut, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) DELETE(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodDelete, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) HEAD(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodHead, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) PATCH(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodPatch, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) CONNECT(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodConnect, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) OPTIONS(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodOptions, routePathLevel, routePathName, routePath, handler, m...)
}

func (s *Route) TRACE(group *echo.Group, routePathLevel string, routePathName string, routePath string, handler func(c *Context) error, m ...echo.MiddlewareFunc) *Route {
	return s.Register(group, http.MethodTrace, routePathLevel, routePathName, routePath, handler, m...)
}

type CustomResponseWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (s *CustomResponseWriter) Write(b []byte) (int, error) {
	s.body.Write(b)
	return s.ResponseWriter.Write(b)
}

func (s *CustomResponseWriter) WriteHeader(statusCode int) {
	s.ResponseWriter.WriteHeader(statusCode)
}

func (s *Route) LoggerRequestResponse(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		start := time.Now()

		resp := s.GetContext(c)
		defer s.PutContext(resp)

		reqBody := bytes.NewBuffer(nil)
		if _, err := io.Copy(reqBody, c.Request().Body); err != nil {
			return resp.Err(err)
		}
		reqBodyBytes := reqBody.Bytes()

		// reset request body
		c.Request().Body = io.NopCloser(reqBody)

		writer := &CustomResponseWriter{
			ResponseWriter: c.Response().Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Response().Writer = writer

		if err := next(c); err != nil {
			return err
		}

		end := time.Now()
		latency := end.Sub(start)

		clientIp := c.RealIP()
		req := c.Request()
		res := c.Response()
		method := req.Method

		switch method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		default:
			return nil
		}

		var id int64
		var username string
		if tmp, ok := c.Get("id").(int64); ok {
			id = tmp
		}
		if tmp, ok := c.Get("username").(string); ok {
			username = tmp
		}
		module := fmt.Sprintf("%v", c.Get("module"))
		urlPath := req.URL.Path

		// build request text
		bufReq := s.GetBuffer()
		defer s.PutBuffer(bufReq)
		{
			// request line
			bufReq.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto))
			// request header
			for k, v := range req.Header {
				bufReq.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ",")))
			}
			// empty line
			bufReq.WriteString("\r\n")
			// request body
			bufReq.Write(reqBodyBytes)
		}

		// build response text
		bufRes := s.GetBuffer()
		defer s.PutBuffer(bufRes)
		{
			// response line
			bufRes.WriteString(fmt.Sprintf("%s %d %s\r\n", req.Proto, res.Status, http.StatusText(res.Status)))
			// response header
			for k, v := range writer.Header() {
				bufRes.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ",")))
			}
			// empty line
			bufRes.WriteString("\r\n")
			// response body
			bufRes.Write(writer.body.Bytes())
		}

		reqStr, repStr := bufReq.String(), bufRes.String()

		respStatus := fmt.Sprintf("%d", res.Status)

		// response status
		{
			respCtx := &Context{}
			if err := json.Unmarshal(writer.body.Bytes(), respCtx); err != nil {
				return err
			}
			respStatus = fmt.Sprintf("%s.%d", respStatus, respCtx.RespCode)
		}

		// write log
		log.Info().
			Str("module", module).
			Str("client_ip", clientIp).
			Str("url_path", urlPath).
			Str("uri", req.RequestURI).
			Str("method", method).
			Int64("id", id).
			Str("username", username).
			Str("request", reqStr).
			Str("response", repStr).
			Str("response_status", respStatus).
			Str("latency", latency.String()).
			Send()

		return nil

	}
}

type Validator struct {
	Validator *validator.Validate
}

func (s *Validator) Validate(i interface{}) error {
	return s.Validator.Struct(i)
}

func NewValidator() (validate *Validator, err error) {
	validate = &Validator{
		Validator: validator.New(),
	}

	/*
	 * 1. string类型字段非必填时必须要在最前面加 "omitempty"
	 * 2. []string 要加 "dive" 才会生效
	 * 3. 有空格的字符串不能使用 "alpha"
	 * 4. []map[string]string 类型需要使用两个 dive 才能控制 key 和 value 的校验规则
	 */

	// 自定义校验规则

	// 校验GET请求的query参数order
	err = validate.Validator.RegisterValidation(
		"order",
		func(fl validator.FieldLevel) bool {
			field := fl.Field()
			switch field.Kind() {
			case reflect.String:
				return regexp.MustCompile(`^([a-zA-Z][A-Za-z0-9_]{0,29}:[ad])(,[a-zA-Z][A-Za-z0-9_]{0,29}:[ad])*$`).MatchString(field.String())
			default:
				return false
			}
		},
	)
	if err != nil {
		return
	}

	return
}

type Binder struct {
	trans          ut.Translator
	validate       *validator.Validate
	defaultBuilder echo.Binder
}

func NewBinder() (echo.Binder, error) {
	uni := ut.New(zh.New())
	trans, _ := uni.GetTranslator("zh")
	vld, err := NewValidator()
	if err != nil {
		return nil, err
	}
	b := &Binder{
		trans:          trans,
		validate:       vld.Validator,
		defaultBuilder: &echo.DefaultBinder{},
	}
	if err = tzh.RegisterDefaultTranslations(b.validate, b.trans); err != nil {
		return nil, err
	}
	b.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		tag := fld.Tag.Get("json")
		if tag != "" && tag != "-" {
			for _, v := range strings.Split(tag, ",") {
				if v != "omitempty" {
					return v
				}
			}
		}
		return fld.Name
	})
	return b, nil
}

// Bind for bind and validate request parameter
func (s *Binder) Bind(i interface{}, c echo.Context) error {
	if err := s.defaultBuilder.Bind(i, c); err != nil {
		return fmt.Errorf("param format parsing failed")
	}
	refValue := reflect.ValueOf(i)
	refKind := refValue.Kind()
	for refKind == reflect.Pointer {
		refValue = refValue.Elem()
		refKind = refValue.Kind()
	}
	if refKind == reflect.Slice {
		for index := 0; index < refValue.Len(); index++ {
			if err := s.validator(refValue.Index(index).Interface()); err != nil {
				return err
			}
		}
		return nil
	}
	if refKind == reflect.Struct {
		if err := s.validator(i); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unsupported binding type: %s", reflect.ValueOf(i).Type().String())
}

// validator validate request parameter
func (s *Binder) validator(i interface{}) error {
	refType := reflect.TypeOf(i)
	refKind := refType.Kind()
	for refKind == reflect.Pointer {
		refType = refType.Elem()
		refKind = refType.Kind()
	}
	if refKind != reflect.Struct {
		return nil
	}
	if err := s.validate.Struct(i); err != nil {
		var tmp validator.ValidationErrors
		if !errors.As(err, &tmp) {
			return err
		}
		for _, v := range tmp {
			return fmt.Errorf("%s", v.Translate(s.trans)) // try to translate the error message
		}
	}
	return nil
}
