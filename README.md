# Go-Utils

Many useful golang tools, support >= v1.12.

![GitHub release](https://img.shields.io/github/release/Laisky/go-utils.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Commitizen friendly](https://img.shields.io/badge/commitizen-friendly-brightgreen.svg)](http://commitizen.github.io/cz-cli/)
[![Go Report Card](https://goreportcard.com/badge/github.com/Laisky/go-utils)](https://goreportcard.com/report/github.com/Laisky/go-utils)
[![GoDoc](https://godoc.org/github.com/Laisky/go-utils?status.svg)](https://godoc.org/github.com/Laisky/go-utils)
[![Build Status](https://travis-ci.org/Laisky/go-utils.svg?branch=master)](https://travis-ci.org/Laisky/go-utils)
[![codecov](https://codecov.io/gh/Laisky/go-utils/branch/master/graph/badge.svg)](https://codecov.io/gh/Laisky/go-utils)

Install:

```sh
go get github.com/Laisky/go-utils
```

## Usage

```go
import (
    "github.com/Laisky/go-utils"
)
```

There are many useful tools including:

* `Clock`: high performance lazy load clock
  * create
    * `NewClock(context.Context, time.Duration) *ClockType`
  * use
    * `Clock.Close()`
    * `Clock.GetUTCNow`
    * `Clock.GetTimeInRFC3339Nano`
    * `Clock.SetupInterval(time.Duration)`
    * `Clock.GetTimeInHex() string`
    * `Clock.GetNanoTimeInHex() string`
* `Settings`: configuration manager that support yml and spring-cloud-config-server
  * setup
    * `SetupFromDir(dirPath string) error`
    * `SetupFromFile(filePath string) error`
    * `SetupFromConfigServer(cfg *ConfigServerCfg) (err error)`
    * `SetupFromConfigServerWithRawYaml(cfg *ConfigServerCfg, key string)`
  * use
    * `Settings.Get(key string) interface{}`
    * `Settings.GetString(key string) string`
    * `Settings.GetStringSlice(key string) []string`
    * `Settings.GetBool(key string) bool`
    * `Settings.GetInt(key string) int`
    * `Settings.GetInt64(key string) int64`
    * `Settings.GetDuration(key string) time.Duration`
    * `Settings.Set(key string, val interface{})`
    * `Settings.IsSet(key string) bool`
    * `Settings.GetStringMap(key string) map[string]interface{}`
    * `Settings.GetStringMapString(key string) map[string]string`
* `Counter`: counter and rotate counter
  * create
    * `NewCounter() *Counter`
    * `NewCounterFromN(n int64) *Counter`
    * `NewRotateCounterWithCtx(ctx context.Context, rotatePoint int64) (*RotateCounter, error)`
    * `NewRotateCounterFromNWithCtx(ctx context.Context, n, rotatePoint int64) (*RotateCounter, error)`
    * `NewUint32Counter() *Uint32Counter`
    * `NewUint32CounterFromN(n uint32) *Uint32Counter`
    * `NewParallelCounter(quoteStep, rotatePoint int64) (*ParallelCounter, error)`
    * `NewParallelCounterFromN(n, quoteStep, rotatePoint int64) (*ParallelCounter, error)`
* `Mail`: simply email sender
  * create
    * `NewMail(host string, port int) *Mail`
  * use
    * `Mail.Login(username, password string)`
    * `Mail.Send(frAddr, toAddr, frName, toName, subject, content string) (err error)`
* encrypt.go:
  * create:
    * `NewJWT(cfg *JwtCfg) (*JWT, error)`
  * use:
    * `GeneratePasswordHash()`: generate hashed password
    * `ValidatePasswordHash()`: validate hashed password
    * `JWT.GenerateToken(userID interface{}, expiresAt time.Time, payload map[string]interface{}) (tokenStr string, err error)`
    * `JWT.Validate(tokenStr string) (payload jwt.MapClaims, err error)`
* `RequestJSON`: simply http client that send json request and unmarshal response by json
* `Logger`: high performance structrued logger based by zap
  * setup:
    * `Logger.ChangeLevel(level string) (err error)`
  * use
    * `Logger.Info(msg string, fields ...zap.Field)`
    * `Logger.Debug(msg string, fields ...zap.Field)`
    * `Logger.Warn(msg string, fields ...zap.Field)`
    * `Logger.Error(msg string, fields ...zap.Field)`
    * `Logger.Panic(msg string, fields ...zap.Field)`
    * `Logger.DebugSample(sample int, msg string, fields ...zap.Field)`
    * `Logger.InfoSample(sample int, msg string, fields ...zap.Field)`
    * `Logger.WarnSample(sample int, msg string, fields ...zap.Field)`
* `AlertHook` add laisky-alert-bot pusher hook to Logger
* `Math`: some simply math functions
  * `Round(val float64, roundOn float64, places int) (newVal float64)`: get round of float
* `Throttle`: throttling to limit throughput
* time: some useful time functions
  * `UTCNow() time.Time`
  * `ParseUnix2String(int64) string`
  * `ParseUnix2UTC(int64) time.Time`
  * `ParseUnixNano2UTC(int64) time.Time`
  * `ParseHex2UTC(string) time.Time`
  * `ParseHexNano2UTC(string) time.Time`
* utils: some tool functions
  * `URLMasking(url, mask string) string`
  * `GetFuncName(f interface{}) string`
  * `FallBack(orig func() interface{}, fallback interface{}) (ret interface{})`
  * `RegexNamedSubMatch(r *regexp.Regexp, str string, subMatchMap map[string]string) error`
  * `FlattenMap(data map[string]interface{}, delimiter string)`
  * `ForceGCBlocking()`
  * `ForceGCUnBlocking()`
  * `TemplateWithMap(tpl string, data map[string]interface{}) string`
* `GZCompressor`
  * create
    * `NewGZCompressor(cfg *GZCompressorCfg) (c *GZCompressor, err error)`
  * use
    * `GZCompressor.Write(d []byte) (int, error)`
    * `GZCompressor.WriteString(d string) (int, error)`
    * `GZCompressor.Flush() (err error)`
    * `GZCompressor.WriteFooter() (err error)`

see more examples in  tests or [document](https://godoc.org/github.com/Laisky/go-utils)

## Usage

some examples

### Settings

load settings from commandline arguments, yaml file or spring-cloud-config-server, then use it anywhere.

1. load settings from commandline arguments:

    ```go
    func setupArgs() {
        pflag.Bool("debug", false, "run in debug mode")
        pflag.Bool("dry", false, "run in dry mode")
        pflag.String("config", "/etc/go-fluentd/settings", "config file directory path")
        pflag.Parse()
        utils.Settings.BindPFlags(pflag.CommandLine)
    }

    func main() {
        // load settings
        setupArgs()

        // use settings anywhere
        if utils.Settings.GetBool("debug") {
            fmt.Println("run in debug mode")
        }
    }
    ```

2. load settings from yaml file:

    prepare settings file:

    ```
    mkdir -p /etc/your-app-name/
    echo "key: val" > /etc/your-app-name/settings.yml
    ```

    load from yaml file:

    ```go
    // load from yaml file
    utils.SetupFromFile("/etc/your-app-name/settings.yml")

    // load from directory (with default filename `settings.yml`)
    utils.Setup("/etc/your-app-name")
    ```

after loading, then you can use `utils.Settings` anywhere:

```go

import "github.com/Laisky/go-utils"

func foo() {
    // set
    utils.Settings.Set("key", "anything")

    // get (latest setted value)
    utils.Settings.Get("key")  // return interface
    utils.Settings.GetString("key")  // return string
    utils.Settings.GetStringSlice("key")  // return []string
    utils.Settings.GetBool("key")  // return bool
    utils.Settings.GetInt64("key")  // return int64
    utils.Settings.GetDuration("key")  // return time.Duration
}
```

### Logger

high performance and simple logging tool based on [zap](https://github.com/uber-go/zap).

```go
// setup basic log level
utils.Logger.ChangeLevel("info")  // info/debug/warn/error

// use as zap
utils.Logger.Debug("some msg", zap.String("key", "val"))
utils.Logger.Info("some msg", zap.String("key", "val"))
utils.Logger.Warn("some msg", zap.String("key", "val"))
utils.Logger.Error("some msg", zap.String("key", "val"))
utils.Logger.Panic("some msg", zap.String("key", "val"))  // will raise panic
```

### Clock

high performance `time.Now` especially on heavy load.

```sh
BenchmarkClock/normal_time-4         	20000000	       107 ns/op	       0 B/op	       0 allocs/op
BenchmarkClock/clock_time_with_500ms-4         	20000000	        62.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkClock/clock_time_with_100ms-4         	20000000	        64.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkClock/clock_time_with_1ms-4           	20000000	        69.1 ns/op	       0 B/op	       0 allocs/op
```

usage:

```go
// use default clock (update per 500ms)
utils.Clock.GetUTCNow()


// setup custom Clock
clock := utils.NewClock(1 * time.Second)
clock.GetUTCNow()
```


### Encrypt

JWT token and hashed password tools.

1. generate and validate JWT token

    [Introduction to JSON Web Tokens](https://jwt.io/introduction/)

    ```go
    func ExampleJWT() {
        jwt, err := utils.NewJWT(utils.NewJWTCfg([]byte("your secret key")))
        if err != nil {
            utils.Logger.Panic("try to init jwt got error", zap.Error(err))
        }

        // generate jwt token for user
        // GenerateToken(userId string, expiresAt time.Time, payload map[string]interface{}) (tokenStr string, err error)
        token, err := jwt.GenerateToken("laisky", time.Now().Add(7*24*time.Hour), map[string]interface{}{"display_name": "Laisky"})
        if err != nil {
            utils.Logger.Error("try to generate jwt token got error", zap.Error(err))
            return
        }
        fmt.Println("got token:", token)

        // validate token
        payload, err := jwt.Validate(token)
        if err != nil {
            utils.Logger.Error("token invalidate")
            return
        }
        fmt.Printf("got payload from token: %+v\n", payload)
    }
    ```


2. generate and validate hashed password

    [Why should I hash passwords?](https://security.stackexchange.com/a/36838/200559)

    ```go
    func ExampleGeneratePasswordHash() {
        // generate hashed password
        rawPassword := []byte("1234567890")
        hashedPassword, err := utils.GeneratePasswordHash(rawPassword)
        if err != nil {
            utils.Logger.Error("try to generate password got error", zap.Error(err))
            return
        }
        fmt.Printf("got new hashed pasword: %v\n", string(hashedPassword))

        // validate passowrd
        if !utils.ValidatePasswordHash(hashedPassword, rawPassword) {
            utils.Logger.Error("password invalidate", zap.Error(err))
            return
        }
    }
    ```

### Math

some useful math functions


1. Round

   ```go
   func ExampleRound() {
       utils.Round(123.555555, .5, 3) // got 123.556
   }
   ```


### Utils

some useful funtions

1. `GetFuncName(f interface{}) string`

    ```go
    func foo() {}

    func ExampleGetFuncName() {
        utils.GetFuncName(foo) // "github.com/Laisky/go-utils_test.foo"
    }
    ```

2. `FallBack(orig func() interface{}, fallback interface{}) (ret interface{})`

   return `fallback` if origin func got error.

    ```go
    func ExampleFallBack() {
        targetFunc := func() interface{} {
            panic("someting wrong")
        }

        utils.FallBack(targetFunc, 10) // got 10
    }
    ```

3. `RegexNamedSubMatch(r *regexp.Regexp, str string, subMatchMap map[string]string) error`

    ```go
    func ExampleRegexNamedSubMatch() {
        reg := regexp.MustCompile(`(?P<key>\d+.*)`)
        str := "12345abcde"
        groups := map[string]string{}
        if err := utils.RegexNamedSubMatch(reg, str, groups); err != nil {
            utils.Logger.Error("try to group match got error", zap.Error(err))
        }

        fmt.Printf("got: %+v", groups) // map[string]string{"key": 12345}
    }
    ```


4. `FlattenMap(data map[string]interface{}, delimiter string)`

    ```go
    func ExampleFlattenMap() {
        data := map[string]interface{}{
            "a": "1",
            "b": map[string]interface{}{
                "c": 2,
                "d": map[string]interface{}{
                    "e": 3,
                },
            },
        }
        utils.FlattenMap(data, "__") // {"a": "1", "b__c": 2, "b__d__e": 3}
    }
    ```
