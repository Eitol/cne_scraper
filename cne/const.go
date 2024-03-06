package cne

import (
	"errors"
	"github.com/valyala/fasthttp"
	"strings"
)

const (
	host              = "www.cne.gob.ve:80"
	baseURL           = host + "/web/registro_electoral/ce.php?nacionalidad=V&cedula="
	responseSizeLimit = 1024 * 7
)

var httpClientSingleton = &fasthttp.HostClient{
	Addr: host,
}

var ErrNotFound = errors.New("not found")
var ErrUserBlocked = errors.New("user blocked")

var munReplacer = strings.NewReplacer(
	"MP. ", "",
	"MP.", "",
	"BLVNO ", "",
)

var estateReplacer = strings.NewReplacer(
	"EDO. ", "",
)

var parishReplacer = strings.NewReplacer(
	"CM. ", "",
	"PQ. ", "",
)
