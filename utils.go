package utils

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/GoWebProd/uuid7"
	"github.com/Laisky/errors/v2"
	"github.com/Laisky/zap"
	"github.com/google/go-cpy/cpy"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/sync/singleflight"

	"github.com/Laisky/go-utils/v4/algorithm"
	"github.com/Laisky/go-utils/v4/common"
	"github.com/Laisky/go-utils/v4/json"
	"github.com/Laisky/go-utils/v4/log"
)

type jsonT struct {
	jsoniter.API
}

var (
	// JSON effective json
	//
	// Deprecated: use github.com/Laisky/go-utils/v4/json instead
	JSON = jsonT{API: jsoniter.ConfigCompatibleWithStandardLibrary}

	internalSFG singleflight.Group

	// for compatibility to old version
	// =====================================

	// Str2Bytes unsafe convert str to bytes
	Str2Bytes = common.Str2Bytes
	// Bytes2Str unsafe convert bytes to str
	Bytes2Str = common.Bytes2Str
	// Number2Roman convert number to roman
	Number2Roman = common.Number2Roman
)

const (
	defaultCgroupMemLimitPath = "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	defaultGCMemRatio         = uint64(85)
)

func init() {
	if _, err := maxprocs.Set(maxprocs.Logger(func(s string, i ...interface{}) {
		log.Shared.Debug(fmt.Sprintf(s, i...))
	})); err != nil {
		log.Shared.Error("auto set maxprocs", zap.Error(err))
	}
}

var cloner = cpy.New(
	cpy.IgnoreAllUnexported(),
)

// DeepClone deep clone a struct
//
// will ignore all unexported fields
func DeepClone[T any](src T) (dst T) {
	return cloner.Copy(src).(T) //nolint:forcetypeassert
}

var dedentMarginChar = regexp.MustCompile(`^[ \t]*`)

type dedentOpt struct {
	replaceTabBySpaces int
}

func (d *dedentOpt) fillDefault() *dedentOpt {
	d.replaceTabBySpaces = 4
	return d
}

func (d *dedentOpt) applyOpts(optfs ...DedentOptFunc) *dedentOpt {
	for _, optf := range optfs {
		optf(d)
	}
	return d
}

// SilentClose close and ignore error
//
// Example
//
//	defer SilentClose(fp)
func SilentClose(v interface{ Close() error }) {
	_ = v.Close()
}

// SilentFlush flush and ignore error
func SilentFlush(v interface{ Flush() error }) {
	_ = v.Flush()
}

// CloseWithLog close and log error.
// logger could be nil, then will use internal log.Shared logger instead.
func CloseWithLog(ins interface{ Close() error },
	logger interface{ Error(string, ...zap.Field) }) {
	LogErr(ins.Close, logger)
}

// LogErr invoke f and log error if got error.
func LogErr(f func() error, logger interface{ Error(string, ...zap.Field) }) {
	if logger == nil {
		logger = log.Shared
	}

	if err := f(); err != nil {
		logger.Error("close ins", zap.Error(err))
	}
}

// FlushWithLog flush and log error.
// logger could be nil, then will use internal log.Shared logger instead.
func FlushWithLog(ins interface{ Flush() error },
	logger interface{ Error(string, ...zap.Field) }) {
	if logger == nil {
		logger = log.Shared
	}

	if err := ins.Flush(); err != nil {
		logger.Error("flush ins", zap.Error(err))
	}
}

// DedentOptFunc dedent option
type DedentOptFunc func(opt *dedentOpt)

// WithReplaceTabBySpaces replace tab to spaces
func WithReplaceTabBySpaces(spaces int) DedentOptFunc {
	return func(opt *dedentOpt) {
		opt.replaceTabBySpaces = spaces
	}
}

// Dedent removes leading whitespace or tab from the beginning of each line
//
// will replace all tab to 4 blanks.
func Dedent(v string, optfs ...DedentOptFunc) string {
	opt := new(dedentOpt).fillDefault().applyOpts(optfs...)
	ls := strings.Split(v, "\n")
	var (
		NSpaceTobeTrim int
		firstLine      = true
		result         = make([]string, 0, len(ls))
	)
	for _, l := range ls {
		if strings.TrimSpace(l) == "" {
			if !firstLine {
				result = append(result, "")
			}

			continue
		}

		m := dedentMarginChar.FindString(l)
		spaceIndent := strings.ReplaceAll(m, "\t", strings.Repeat(" ", opt.replaceTabBySpaces))
		n := len(spaceIndent)
		l = strings.Replace(l, m, spaceIndent, 1)
		if firstLine {
			NSpaceTobeTrim = n
			firstLine = false
		} else if n != 0 && n < NSpaceTobeTrim {
			// choose the smallest margin
			NSpaceTobeTrim = n
		}

		result = append(result, l)
	}

	for i := range result {
		if result[i] == "" {
			continue
		}

		result[i] = result[i][NSpaceTobeTrim:]
	}

	// remove tail blank lines
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] == "" {
			result = result[:i]
		} else {
			break
		}
	}

	return strings.Join(result, "\n")
}

// HasField check is struct has field
//
// inspired by https://mrwaggel.be/post/golang-reflect-if-initialized-struct-has-member-method-or-fields/
func HasField(st any, fieldName string) bool {
	valueIface := reflect.ValueOf(st)

	// Check if the passed interface is a pointer
	if valueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface's Type, so we have a pointer to work with
		valueIface = reflect.New(reflect.TypeOf(st))
	}

	// 'dereference' with Elem() and get the field by name
	field := valueIface.Elem().FieldByName(fieldName)
	return field.IsValid()
}

// HasMethod check is struct has method
//
// inspired by https://mrwaggel.be/post/golang-reflect-if-initialized-struct-has-member-method-or-fields/
func HasMethod(st any, methodName string) bool {
	valueIface := reflect.ValueOf(st)

	// Check if the passed interface is a pointer
	if valueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface, so we have a pointer to work with
		valueIface = reflect.New(reflect.TypeOf(st))
	}

	// Get the method by name
	method := valueIface.MethodByName(methodName)
	return method.IsValid()
}

// MD5JSON calculate md5(jsonify(data))
func MD5JSON(data any) (string, error) {
	if NilInterface(data) {
		return "", errors.New("data is nil")
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", md5.Sum(b)), nil
}

// NilInterface make sure data is nil interface or another type with nil value
//
// Example:
//
//	type foo struct{}
//	var f *foo
//	var v any
//	v = f
//	v == nil // false
//	NilInterface(v) // true
func NilInterface(data any) bool {
	if data == nil {
		return true
	}

	if reflect.TypeOf(data).Kind() == reflect.Ptr &&
		reflect.ValueOf(data).IsNil() {
		return true
	}

	return false
}

// GetStructFieldByName get struct field by name
func GetStructFieldByName(st any, fieldName string) any {
	stv := reflect.ValueOf(st)
	if IsPtr(st) {
		stv = stv.Elem()
	}

	v := stv.FieldByName(fieldName)
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Slice,
		reflect.Array,
		reflect.Interface,
		reflect.Ptr,
		reflect.Map:
		if v.IsNil() {
			return nil
		}
	default:
		// do nothing
	}

	return v.Interface()
}

// ValidateFileHash validate file content with hashed string
//
// Args:
//   - filepath: file path to check
//   - hashed: hashed string, like `sha256: xxxx`
func ValidateFileHash(filepath string, hashed string) error {
	hs := strings.Split(hashed, ":")
	if len(hs) != 2 {
		return errors.Errorf("unknown hashed format, expect is `sha256:xxxx`, but got `%s`", hashed)
	}

	var hasher HashType
	switch hs[0] {
	case "sha256":
		hasher = HashTypeSha256
	case "md5":
		hasher = HashTypeMD5
	default:
		return errors.Errorf("unknown hasher `%s`", hs[0])
	}

	fp, err := os.Open(filepath)
	if err != nil {
		return errors.Wrapf(err, "open file `%s`", filepath)
	}
	defer SilentClose(fp)

	sig, err := Hash(hasher, fp)
	if err != nil {
		return errors.Wrapf(err, "calculate hash for file %q", filepath)
	}

	actualHash := hex.EncodeToString(sig)
	if hs[1] != actualHash {
		return errors.Errorf("hash `%s` not match expect `%s`", actualHash, hs[1])
	}

	return nil
}

// GetFuncName return the name of func
func GetFuncName(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// FallBack return the fallback when orig got error
// utils.FallBack(func() any { return getIOStatMetric(fs) }, &IOStat{}).(*IOStat)
func FallBack(orig func() any, fallback any) (ret any) {
	defer func() {
		if recover() != nil {
			ret = fallback
		}
	}()

	ret = orig()
	return
}

// RegexNamedSubMatch extract key:val map from string by group match
//
// Deprecated: use RegexNamedSubMatch2 instead
func RegexNamedSubMatch(r *regexp.Regexp, str string, subMatchMap map[string]string) error {
	match := r.FindStringSubmatch(str)
	names := r.SubexpNames()
	if len(names) != len(match) {
		return errors.New("the number of args in `regexp` and `str` not matched")
	}

	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" {
			subMatchMap[name] = match[i]
		}
	}

	return nil
}

// RegexNamedSubMatch2 extract key:val map from string by group match
func RegexNamedSubMatch2(r *regexp.Regexp, str string) (subMatchMap map[string]string, err error) {
	match := r.FindStringSubmatch(str)
	names := r.SubexpNames()
	if len(names) != len(match) {
		return nil, errors.New("the number of args in `regexp` and `str` not matched")
	}

	subMatchMap = make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" {
			subMatchMap[name] = match[i]
		}
	}

	return subMatchMap, nil
}

// FlattenMap make embedded map into flatten map
func FlattenMap(data map[string]any, delimiter string) {
	for k, vi := range data {
		if v2i, ok := vi.(map[string]any); ok {
			FlattenMap(v2i, delimiter)
			for k3, v3i := range v2i {
				data[k+delimiter+k3] = v3i
			}
			delete(data, k)
		}
	}
}

// ForceGCBlocking force to run blocking manual gc.
func ForceGCBlocking() {
	log.Shared.Debug("force gc")
	runtime.GC()
	debug.FreeOSMemory()
}

// ForceGCUnBlocking trigger GC unblocking
func ForceGCUnBlocking() {
	go func() {
		_, _, _ = internalSFG.Do("ForceGCUnBlocking", func() (any, error) {
			ForceGC()
			return struct{}{}, nil
		})
	}()
}

type gcOption struct {
	memRatio         uint64
	memLimitFilePath string
}

// GcOptFunc option for GC utils
type GcOptFunc func(*gcOption) error

// WithGCMemRatio set mem ratio trigger for GC
//
// default to 85
func WithGCMemRatio(ratio int) GcOptFunc {
	return func(opt *gcOption) error {
		if ratio <= 0 {
			return errors.Errorf("ratio must > 0, got %d", ratio)
		}
		if ratio > 100 {
			return errors.Errorf("ratio must <= 0, got %d", ratio)
		}

		log.Shared.Debug("set memRatio", zap.Int("ratio", ratio))
		opt.memRatio = uint64(ratio)
		return nil
	}
}

// WithGCMemLimitFilePath set memory limit file
func WithGCMemLimitFilePath(path string) GcOptFunc {
	return func(opt *gcOption) error {
		if _, err := os.Open(path); err != nil {
			return errors.Wrapf(err, "try open path `%s`", path)
		}

		log.Shared.Debug("set memLimitFilePath", zap.String("file", path))
		opt.memLimitFilePath = path
		return nil
	}
}

// AutoGC auto trigger GC when memory usage exceeds the custom ration
//
// default to /sys/fs/cgroup/memory/memory.limit_in_bytes
func AutoGC(ctx context.Context, opts ...GcOptFunc) (err error) {
	opt := &gcOption{
		memRatio:         defaultGCMemRatio,
		memLimitFilePath: defaultCgroupMemLimitPath,
	}
	for _, optf := range opts {
		if err = optf(opt); err != nil {
			return errors.Wrap(err, "set option")
		}
	}

	var (
		fp       *os.File
		memByte  []byte
		memLimit uint64
	)
	if fp, err = os.Open(opt.memLimitFilePath); err != nil {
		return errors.Wrapf(err, "open file got error: %+v", opt.memLimitFilePath)
	}
	defer SilentClose(fp)

	if memByte, err = io.ReadAll(fp); err != nil {
		return errors.Wrap(err, "read cgroup mem limit file")
	}

	if err = fp.Close(); err != nil {
		log.Shared.Error("close cgroup mem limit file", zap.Error(err), zap.String("file", opt.memLimitFilePath))
	}

	if memLimit, err = strconv.ParseUint(string(bytes.TrimSpace(memByte)), 10, 64); err != nil {
		return errors.Wrap(err, "parse cgroup memory limit")
	}
	if memLimit == 0 {
		return errors.Errorf("mem limit should > 0, but got: %d", memLimit)
	}
	log.Shared.Info("enable auto gc", zap.Uint64("ratio", opt.memRatio), zap.Uint64("limit", memLimit))

	go func(ctx context.Context) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		var (
			m     runtime.MemStats
			ratio uint64
		)
		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
			runtime.ReadMemStats(&m)
			ratio = (m.Alloc * 100) / memLimit
			log.Shared.Debug("mem stat",
				zap.Uint64("mem", m.Alloc),
				zap.Uint64("limit_mem", memLimit),
				zap.Uint64("ratio", ratio),
				zap.Uint64("limit_ratio", opt.memRatio),
			)
			if ratio >= opt.memRatio {
				ForceGCBlocking()
			}
		}
	}(ctx)

	return nil
}

var (
	// ForceGC force to start gc blocking
	ForceGC = ForceGCBlocking
	// TriggerGC force to start gc unblocking
	TriggerGC = ForceGCUnBlocking
)

var defaultTemplateWithMappReg = regexp.MustCompile(`(?sm)\$\{([^}]+)\}`)

// TemplateWithMap replace `${var}` in template string
func TemplateWithMap(tpl string, data map[string]any) string {
	return TemplateWithMapAndRegexp(defaultTemplateWithMappReg, tpl, data)
}

// TemplateWithMapAndRegexp replace `${var}` in template string
func TemplateWithMapAndRegexp(tplReg *regexp.Regexp, tpl string, data map[string]any) string {
	var (
		k, vs string
		vi    any
	)
	for _, kg := range tplReg.FindAllStringSubmatch(tpl, -1) {
		k = kg[1]
		vi = data[k]
		switch vi := vi.(type) {
		case string:
			vs = vi
		case []byte:
			vs = string(vi)
		case int:
			vs = strconv.FormatInt(int64(vi), 10)
		case int64:
			vs = strconv.FormatInt(vi, 10)
		case float64:
			vs = strconv.FormatFloat(vi, 'f', -1, 64)
		}
		tpl = strings.ReplaceAll(tpl, "${"+k+"}", vs)
	}

	return tpl
}

var (
	urlMaskingRegexp = regexp.MustCompile(`(\S+:)\S+(@\w+)`)
)

// URLMasking masking password in url
func URLMasking(url, mask string) string {
	return urlMaskingRegexp.ReplaceAllString(url, `${1}`+mask+`${2}`)
}

// SetStructFieldsBySlice set field value of structs slice by values slice
func SetStructFieldsBySlice(structs, vals any) (err error) {
	sv := reflect.ValueOf(structs)
	vv := reflect.ValueOf(vals)

	typeCheck := func(name string, v *reflect.Value) error {
		switch v.Kind() {
		case reflect.Slice:
		case reflect.Array:
		default:
			return errors.Errorf(name + " must be array/slice")
		}

		return nil
	}
	if err = typeCheck("structs", &sv); err != nil {
		return err
	}
	if err = typeCheck("vals", &vv); err != nil {
		return err
	}

	var (
		eachGrpValsV    reflect.Value
		iField, nFields int
	)
	for i := 0; i < Min(sv.Len(), vv.Len()); i++ {
		eachGrpValsV = vv.Index(i)
		if err = typeCheck("vals."+strconv.FormatInt(int64(i), 10), &eachGrpValsV); err != nil {
			return err
		}
		switch sv.Index(i).Kind() {
		case reflect.Ptr:
			nFields = sv.Index(i).Elem().NumField()
		default:
			nFields = sv.Index(i).NumField()
		}
		for iField = 0; iField < Min(eachGrpValsV.Len(), nFields); iField++ {
			switch sv.Index(i).Kind() {
			case reflect.Ptr:
				sv.Index(i).Elem().Field(iField).Set(eachGrpValsV.Index(iField))
			default:
				sv.Index(i).Field(iField).Set(eachGrpValsV.Index(iField))
			}
		}
	}

	return
}

// UniqueStrings remove duplicate string in slice
func UniqueStrings(vs []string) []string {
	seen := make(map[string]struct{})
	j := 0
	for _, v := range vs {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			vs[j] = v
			j++
		}
	}

	clear(vs[j:])
	return vs[:j:j]
}

// RemoveEmpty remove duplicate string in slice
func RemoveEmpty(vs []string) (r []string) {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			r = append(r, v)
		}
	}

	return
}

// TrimEleSpaceAndRemoveEmpty remove duplicate string in slice
func TrimEleSpaceAndRemoveEmpty(vs []string) (r []string) {
	for _, v := range vs {
		v = strings.TrimSpace(v)
		if v != "" {
			r = append(r, v)
		}
	}

	return
}

// Contains if collection contains ele
func Contains[V comparable](collection []V, ele V) bool {
	return slices.Contains(collection, ele)
}

// IsPtr check if t is pointer
func IsPtr(t any) bool {
	return reflect.TypeOf(t).Kind() == reflect.Ptr
}

var reInvalidCMDChars = regexp.MustCompile(`[;&|]`)

// SanitizeCMDArgs sanitizes the given command arguments.
func SanitizeCMDArgs(args []string) (sanitizedArgs []string, err error) {
	for i, arg := range args {
		// Check for invalid characters using a regular expression
		if reInvalidCMDChars.MatchString(arg) {
			return nil, errors.New("invalid characters in args")
		}

		// Check for command substitution
		if strings.Contains(arg, "$(") || strings.Contains(arg, "`") {
			return nil, errors.New("invalid command substitution in args")
		}

		// Trim leading and trailing whitespace
		args[i] = strings.TrimSpace(arg)
	}

	return args, nil
}

// RunCMD run command script
func RunCMD(ctx context.Context, app string, args ...string) (stdout []byte, err error) {
	return RunCMDWithEnv(ctx, app, args, nil)
}

// RunCMDWithEnv run command with environments
//
// # Args
//   - envs: []string{"FOO=BAR"}
func RunCMDWithEnv(ctx context.Context, app string,
	args []string, envs []string) (stdout []byte, err error) {
	cmd := exec.CommandContext(ctx, app, args...)

	if len(envs) != 0 {
		cmd.Env = append(cmd.Env, envs...)
	}

	stdout, err = cmd.CombinedOutput()
	if err != nil {
		cmd := strings.Join(append([]string{app}, args...), " ")
		return stdout, errors.Wrapf(err, "run %q got %q", cmd, stdout)
	}

	return stdout, nil
}

// RunCMD2 run command script and handle stdout/stderr by pipe
func RunCMD2(ctx context.Context, app string,
	args []string, envs []string,
	stdoutHandler, stderrHandler func(string),
) (err error) {
	cmd := exec.CommandContext(ctx, app, args...)
	cmd.Env = append(cmd.Env, envs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "get stdout")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "get stderr")
	}

	if stdoutHandler == nil {
		stdoutHandler = func(s string) {
			log.Shared.Debug("run cmd", zap.String("msg", s), zap.String("app", app))
		}
	}

	if stderrHandler == nil {
		stderrHandler = func(s string) {
			log.Shared.Error("run cmd", zap.String("msg", s), zap.String("app", app))
		}
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "start cmd")
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			out := scanner.Text()
			stdoutHandler(out)
		}

		if err := scanner.Err(); err != nil {
			log.Shared.Warn("read stdout", zap.Error(err))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			out := scanner.Text()
			stderrHandler(out)
		}

		if err := scanner.Err(); err != nil {
			log.Shared.Warn("read stderr", zap.Error(err))
		}
	}()

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "wait cmd")
	}

	return nil
}

// EncodeByBase64 encode bytes to string by base64
func EncodeByBase64(raw []byte) string {
	return base64.URLEncoding.EncodeToString(raw)
}

// DecodeByBase64 decode string to bytes by base64
func DecodeByBase64(encoded string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(encoded)
}

var (
	// EncodeByHex encode bytes to string by hex
	EncodeByHex = hex.EncodeToString
	// DecodeByHex decode string to bytes by hex
	DecodeByHex = hex.DecodeString
)

// ConvertMap2StringKey convert any map to `map[string]any`
func ConvertMap2StringKey(inputMap any) map[string]any {
	v := reflect.ValueOf(inputMap)
	if v.Kind() != reflect.Map {
		return nil
	}

	m2 := map[string]any{}
	ks := v.MapKeys()
	for _, k := range ks {
		if k.Kind() == reflect.Interface {
			m2[k.Elem().String()] = v.MapIndex(k).Interface()
		} else {
			m2[fmt.Sprint(k)] = v.MapIndex(k).Interface()
		}
	}

	return m2
}

// func CalculateCRC(cnt []byte) {
// 	cw := crc64.New(crc64.MakeTable(crc64.ISO))
// }

// IsPanic is `f()` throw panic
//
// if you want to get the data throwed by panic, use `IsPanic2`
func IsPanic(f func()) (isPanic bool) {
	defer func() {
		if deferErr := recover(); deferErr != nil {
			isPanic = true
		}
	}()

	f()
	return false
}

// IsPanic2 check is `f()` throw panic, and return panic as error
func IsPanic2(f func()) (err error) {
	defer func() {
		if panicRet := recover(); panicRet != nil {
			err = errors.Errorf("panic: %v", panicRet)
		}
	}()

	f()
	return nil
}

var onlyOneSignalHandler = make(chan struct{})

type stopSignalOpt struct {
	closeSignals []os.Signal
	// closeFunc    func()
}

// StopSignalOptFunc options for StopSignal
type StopSignalOptFunc func(*stopSignalOpt)

// WithStopSignalCloseSignals set signals that will trigger close
func WithStopSignalCloseSignals(signals ...os.Signal) StopSignalOptFunc {
	if len(signals) == 0 {
		log.Shared.Panic("signals cannot be empty")
	}

	return func(opt *stopSignalOpt) {
		opt.closeSignals = signals
	}
}

// // WithStopSignalCloseFunc set func that will be called when signal is triggered
// func WithStopSignalCloseFunc(f func()) StopSignalOptFunc {
// 	if f == nil {
// 		log.Shared.Panic("f cannot be nil")
// 	}

// 	return func(opt *stopSignalOpt) {
// 		opt.closeFunc = f
// 	}
// }

// StopSignal registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
//
// Copied from https://github.com/kubernetes/sample-controller
func StopSignal(optfs ...StopSignalOptFunc) (stopCh <-chan struct{}) {
	opt := &stopSignalOpt{
		closeSignals: []os.Signal{syscall.SIGTERM, syscall.SIGINT},
		// closeFunc:    func() { os.Exit(1) },
	}
	for _, optf := range optfs {
		optf(opt)
	}

	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		close(stop)
	}()

	return stop
}

// PanicIfErr panic if err is not nil
func PanicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

// GracefulCancel is a function that will be called when the process is about to be terminated.
func GracefulCancel(cancel func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	cancel()
}

// EmptyAllChans receive all thins in all chans
func EmptyAllChans[T any](chans ...chan T) {
	for _, c := range chans {
		for range c { //nolint: revive
		}
	}
}

type prettyBuildInfoOption struct {
	withDeps bool
}

func (o *prettyBuildInfoOption) apply(fs ...PrettyBuildInfoOption) *prettyBuildInfoOption {
	for _, f := range fs {
		f(o)
	}

	return o
}

// PrettyBuildInfoOption options for PrettyBuildInfo
type PrettyBuildInfoOption func(*prettyBuildInfoOption)

// WithPrettyBuildInfoDeps include deps in build info
func WithPrettyBuildInfoDeps() PrettyBuildInfoOption {
	return func(opt *prettyBuildInfoOption) {
		opt.withDeps = true
	}
}

// PrettyBuildInfo get build info in formatted json
//
// Print:
//
//	{
//	  "Path": "github.com/Laisky/go-ramjet",
//	  "Version": "v0.0.0-20220718014224-2b10e57735f1",
//	  "Sum": "h1:08Ty2gR+Xxz0B3djHVuV71boW4lpNdQ9hFn4ZIGrhec=",
//	  "Replace": null
//	}
func PrettyBuildInfo(opts ...PrettyBuildInfoOption) string {
	opt := new(prettyBuildInfoOption).apply(opts...)

	info, ok := debug.ReadBuildInfo()
	if !ok {
		log.Shared.Error("failed to read build info")
		return ""
	}

	if !opt.withDeps {
		info.Deps = nil
	}

	ver, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Shared.Error("failed to marshal version", zap.Error(err))
		return ""
	}

	return string(ver)
}

// IsEmpty is empty
func IsEmpty(val any) bool {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)
	if t.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true
		}

		if v.Elem().IsZero() {
			return true
		}
	} else {
		if v.IsZero() {
			return true
		}
	}

	return false
}

// NotEmpty val should not be empty, with pretty error msg
func NotEmpty(val any, name string) error {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)
	if t.Kind() == reflect.Ptr {
		if v.IsNil() {
			return errors.Errorf("%q is empty pointer", name)
		}

		if v.Elem().IsZero() {
			return errors.Errorf("%q is point to empty elem", name)
		}
	} else {
		if v.IsZero() {
			return errors.Errorf("%q is empty elem", name)
		}
	}

	return nil
}

// OptionalVal return optionval if not empty
func OptionalVal[T any](ptr *T, optionalVal T) T {
	if IsEmpty(ptr) {
		return optionalVal
	}

	return *ptr
}

// CostSecs convert duration to string like `0.25s`
func CostSecs(cost time.Duration) string {
	return fmt.Sprintf("%.2fs", float64(cost)/float64(time.Second))
}

// Pipeline run f(v) for all funcs
func Pipeline[T any](funcs []func(T) error, v T) (T, error) {
	for _, f := range funcs {
		if err := f(v); err != nil {
			return v, err
		}
	}

	return v, nil
}

// UUID1 get uuid version 1
//
// Deprecated: use UUID7 instead
func UUID1() string {
	return uuid.Must(uuid.NewUUID()).String()
}

// UUID4 get uuid version 4
func UUID4() string {
	return uuid.Must(uuid.NewRandom()).String()
}

var uuid7Gen = uuid7.New()

// UUID7 get uuid version 7
func UUID7() string {
	return uuid7Gen.Next().String()
}

// UUID7Itf general uuid7 interface
type UUID7Itf interface {
	// Timestamp get timestamp of uuid7
	Timestamp() uint64
	// String get string of uuid7
	String() string
	// Empty check if uuid7 is empty
	Empty() bool
}

// ParseUUID7 parse uuid7
func ParseUUID7(val string) (UUID7Itf, error) {
	return uuid7.Parse(val)
}

// Delayer create by NewDelay
//
// do not use this type directly.
type Delayer struct {
	startAt time.Time
	d       time.Duration
}

// NewDelay ensures the execution time of a function is not less than a predefined threshold.
//
//	defer NewDelay(time.Second).Wait()
func NewDelay(d time.Duration) *Delayer {
	return &Delayer{
		startAt: time.Now(),
		d:       d,
	}
}

// Wait wait in defer
func (d *Delayer) Wait() {
	time.Sleep(d.d - time.Since(d.startAt))
}

// FileHashSharding get file hash sharding path
func FileHashSharding(fname string) string {
	hasher := sha1.New()
	if _, err := hasher.Write([]byte(fname)); err != nil {
		log.Shared.Panic("failed to write file name to hasher", zap.Error(err))
	}

	hashed := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(hashed[:2], hashed[2:4], fname)
}

// ReverseSlice reverse slice
func ReverseSlice[T any](s []T) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		opp := len(s) - 1 - i
		s[i], s[opp] = s[opp], s[i]
	}
}

// RemoveEmptyVal remove empty value in map
func RemoveEmptyVal(m map[string]any) map[string]any {
	for k, v := range m {
		if v == nil || reflect.ValueOf(v).IsZero() {
			delete(m, k)
			continue
		}

		switch reflect.TypeOf(v).Kind() {
		case reflect.Map:
			if v == nil || reflect.ValueOf(v).Len() == 0 {
				delete(m, k)
				continue
			}

			switch v := v.(type) {
			case map[string]any:
				if v := RemoveEmptyVal(v); len(v) == 0 {
					delete(m, k)
					continue
				} else {
					m[k] = v
				}
			default:
				continue
			}
		case reflect.Slice, reflect.Array:
			if v == nil || reflect.ValueOf(v).Len() == 0 {
				delete(m, k)
				continue
			}
		default:
			continue
		}
	}

	return m
}

// CombineSortedChain return the intersection of multiple sorted chans
func CombineSortedChain[T Sortable](sortOrder common.SortOrder, chans ...chan T) (result chan T, err error) {
	if len(chans) == 0 {
		return nil, errors.New("chans cannot be empty")
	} else if len(chans) == 1 {
		return chans[0], nil
	}

	heap := algorithm.NewPriorityQ[T](sortOrder)

	activeChans := make(map[int]chan T, len(chans))
	for i, ch := range chans {
		activeChans[i] = ch
	}

	result = make(chan T)
	go func() {
		defer close(result)

		for idx, c := range activeChans {
			v, ok := <-c
			if !ok {
				continue
			}

			heap.Push(algorithm.PriorityItem[T]{
				Val:  v,
				Name: idx,
			})
		}

		for {
			if heap.Len() == 0 {
				return
			}

			it := heap.Pop()
			result <- it.GetVal()

			idx := it.(algorithm.PriorityItem[T]).Name.(int) //nolint:forcetypeassert // panic
			ch, ok := activeChans[idx]
			if !ok { // this chan is already exhausted and removed
				continue
			}

			v, ok := <-ch
			if !ok { // this chan is exhausted
				delete(activeChans, idx)

				// there is no active chans
				if len(activeChans) == 0 {
					for i := 0; i < heap.Len(); i++ {
						it := heap.Pop()
						result <- it.GetVal()
					}

					return
				}

				continue
			}

			heap.Push(algorithm.PriorityItem[T]{
				Val:  v,
				Name: idx,
			})
		}
	}()

	return result, nil
}

// FilterSlice filters a slice inplace
func FilterSlice[T any](s []T, f func(v T) bool) []T {
	var j int
	for _, v := range s {
		if f(v) {
			s[j] = v
			j++
		}
	}

	clear(s[j:])
	return s[:j:j]
}

// GetEnvInsensitive get env case insensitive
func GetEnvInsensitive(key string) (values []string) {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.EqualFold(pair[0], key) {
			values = append(values, pair[1])
		}
	}

	return
}

// RegexpOidFormat check if oid is valid
var RegexpOidFormat = regexp.MustCompile(`^\d(?:\.\d+){0,}$`)

// ParseObjectIdentifier parse oid
func ParseObjectIdentifier(val string) (oid asn1.ObjectIdentifier, err error) {
	if !RegexpOidFormat.MatchString(val) {
		return nil, errors.Errorf("invalid oid format: %q", val)
	}

	vals := strings.Split(val, ".")
	oid = make(asn1.ObjectIdentifier, len(vals))
	for i, v := range vals {
		oid[i], err = strconv.Atoi(v)
		if err != nil {
			return nil, errors.Wrapf(err, "parse oid %q", val)
		}
	}

	return oid, nil
}

// NewHasPrefixWithMagic create a func to check if s has prefix
//
// if the length of prefix is quite short, it will use magic number to check.
func NewHasPrefixWithMagic(prefix []byte) func(s []byte) bool {
	switch l := len(prefix); l {
	case 8:
		prefixMagicNumber := binary.NativeEndian.Uint64(prefix)
		return func(s []byte) bool {
			return len(s) >= l && *(*uint64)(unsafe.Pointer(&s[0])) == prefixMagicNumber
		}
	case 4:
		prefixMagicNumber := binary.NativeEndian.Uint32(prefix)
		return func(s []byte) bool {
			return len(s) >= l && *(*uint32)(unsafe.Pointer(&s[0])) == prefixMagicNumber
		}
	case 2:
		prefixMagicNumber := binary.NativeEndian.Uint16(prefix)
		return func(s []byte) bool {
			return len(s) >= l && *(*uint16)(unsafe.Pointer(&s[0])) == prefixMagicNumber
		}
	case 0:
		return func(s []byte) bool {
			return true
		}
	default:
		return func(s []byte) bool {
			return bytes.HasPrefix(s, prefix)
		}
	}
}
