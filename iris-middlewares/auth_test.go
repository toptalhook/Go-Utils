package irisMiddlewares_test

import (
	"net/http"
	"time"

	irisMiddlewares "github.com/Laisky/go-utils/iris-middlewares"
	"github.com/kataras/iris"
)

func ExampleAuthType() {
	cfg := irisMiddlewares.AuthCfg
	cfg.Secret = "f32lifj2f32fj"
	auth := irisMiddlewares.NewAuth(cfg)

	uid := "123"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	payload := map[string]interface{}{"a": "b"}
	auth.GenerateToken(uid, expiresAt, payload)

	Server := iris.New()
	Server.Handle("ANY", "/authorized/", irisMiddlewares.FromStd(DemoHandle))
}

func DemoHandle(w http.ResponseWriter, r *http.Request) {
	// irisMiddlewares
	w.Write([]byte("hello"))
}
