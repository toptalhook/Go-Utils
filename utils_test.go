package utils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Laisky/errors/v2"
	"github.com/Laisky/zap"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/Laisky/go-utils/v4/common"
	"github.com/Laisky/go-utils/v4/json"
	"github.com/Laisky/go-utils/v4/log"
)

type testEmbeddedSt struct{}

type testStCorrect1 struct {
	testEmbeddedSt
}
type testStCorrect2 struct {
	testEmbeddedSt string
}
type testStFail struct {
}

func (t *testStCorrect1) PointerMethod() {

}
func (t *testStCorrect1) Method() {

}

func TestHasMethod(t *testing.T) {
	st1 := testStCorrect1{}
	st1p := &testStCorrect1{}
	st2 := testStFail{}
	st2p := &testStFail{}

	_ = st1.testEmbeddedSt
	_ = st1p.testEmbeddedSt

	if !HasMethod(st1, "Method") {
		t.Fatal()
	}
	if !HasMethod(st1, "PointerMethod") {
		t.Fatal()
	}
	if !HasMethod(st1p, "Method") {
		t.Fatal()
	}
	if !HasMethod(st1p, "PointerMethod") {
		t.Fatal()
	}
	if HasMethod(st2, "Method") {
		t.Fatal()
	}
	if HasMethod(st2, "PointerMethod") {
		t.Fatal()
	}
	if HasMethod(st2p, "Method") {
		t.Fatal()
	}
	if HasMethod(st2p, "PointerMethod") {
		t.Fatal()
	}
}

func TestHasField(t *testing.T) {
	t.Parallel()

	st1 := testStCorrect1{}
	st1p := &testStCorrect1{}
	st2 := testStCorrect2{}
	st2p := &testStCorrect2{}
	st3 := testStFail{}
	st3p := &testStFail{}

	_ = st2.testEmbeddedSt

	if !HasField(st1, "testEmbeddedSt") {
		t.Fatal()
	}
	if !HasField(st1p, "testEmbeddedSt") {
		t.Fatal()
	}
	if !HasField(st2, "testEmbeddedSt") {
		t.Fatal()
	}
	if !HasField(st2p, "testEmbeddedSt") {
		t.Fatal()
	}
	if HasField(st3, "testEmbeddedSt") {
		t.Fatal()
	}
	if HasField(st3p, "testEmbeddedSt") {
		t.Fatal()
	}
}

func TestValidateFileHash(t *testing.T) {
	t.Parallel()

	fp, err := os.CreateTemp("", "go-utils-*")
	require.NoError(t, err)
	defer os.Remove(fp.Name())
	defer fp.Close()

	content := []byte("jijf32ijr923e890dsfuodsafjlj;f9o2ur9re")
	_, err = fp.Write(content)
	require.NoError(t, err)

	err = ValidateFileHash(fp.Name(), "sha256:123")
	require.Error(t, err)

	err = ValidateFileHash(fp.Name(), "md5:123")
	require.Error(t, err)

	err = ValidateFileHash(fp.Name(), "sha254:123")
	require.Error(t, err)

	err = ValidateFileHash(fp.Name(), "")
	require.Error(t, err)

	err = ValidateFileHash(
		fp.Name(),
		"sha256:aea7e26c0e0b12ad210a8a0e45c379d0325b567afdd4b357158059b0ef03ae67",
	)
	require.NoError(t, err)

	err = ValidateFileHash(
		fp.Name(),
		"md5:794e37eea6b3df6e6eba69eb02f9b8c7",
	)
	require.NoError(t, err)
}

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal", func(t *testing.T) {
		jb, err := json.Marshal("123")
		require.NoError(t, err)

		var v string
		json.Unmarshal(jb, &v)
		require.NoError(t, err)
		require.Equal(t, "123", v)
	})

	t.Run("marshal string", func(t *testing.T) {
		jb, err := json.MarshalToString("123")
		require.NoError(t, err)

		var v string
		json.UnmarshalFromString(jb, &v)
		require.NoError(t, err)
		require.Equal(t, "123", v)
	})

	t.Run("json not support comment", func(t *testing.T) {
		d := struct {
			K string
		}{}
		raw := `{
			// comment
			"k": "v"  // comment
			}`
		err := json.Unmarshal([]byte(raw), &d)
		require.ErrorContains(t, err, "invalid character '/'")
	})

	t.Run("json support comment", func(t *testing.T) {
		d := struct {
			K string
		}{}
		raw := `{
			// comment
			"k": "v"  // comment
			}`
		err := json.UnmarshalComment([]byte(raw), &d)
		require.NoError(t, err)
		require.Equal(t, "v", d.K)
	})
}

func TestIsPtr(t *testing.T) {
	vp := &struct{}{}
	vt := struct{}{}

	if !IsPtr(vp) {
		t.Fatal()
	}
	if IsPtr(vt) {
		t.Fatal()
	}
}

func testFoo() {}

func TestGetFuncName(t *testing.T) {
	t.Parallel()

	if name := GetFuncName(testFoo); name != "github.com/Laisky/go-utils/v4.testFoo" {
		t.Fatalf("want `testFoo`, got `%v`", name)
	}
}

func ExampleGetFuncName() {
	GetFuncName(testFoo) // "github.com/Laisky/go-utils.testFoo"
}

func TestFallBack(t *testing.T) {
	t.Parallel()

	fail := func() any {
		panic("got error")
	}
	expect := 10
	got := FallBack(fail, 10)
	if expect != got.(int) {
		t.Errorf("expect %v got %v", expect, got)
	}
}

func ExampleFallBack() {
	targetFunc := func() any {
		panic("someting wrong")
	}

	FallBack(targetFunc, 10) // got 10
}

func TestRegexNamedSubMatch2(t *testing.T) {
	t.Parallel()

	reg := regexp.MustCompile(`^(?P<time>.{23}) {0,}\| {0,}(?P<app>[^ ]+) {0,}\| {0,}(?P<level>[^ ]+) {0,}\| {0,}(?P<thread>[^ ]+) {0,}\| {0,}(?P<class>[^ ]+) {0,}\| {0,}(?P<line>\d+) {0,}([\|:] {0,}(?P<args>\{.*\})){0,1}([\|:] {0,}(?P<message>.*)){0,1}`)
	str := "2018-04-02 02:02:10.928 | sh-datamining | INFO | http-nio-8080-exec-80 | com.pateo.qingcloud.gateway.core.zuul.filters.post.LogFilter | 74 | xxx"
	submatchMap, err := RegexNamedSubMatch2(reg, str)
	require.NoError(t, err)

	for k, v := range submatchMap {
		fmt.Println(">>", k, ":", v)
	}

	if v1, ok := submatchMap["level"]; !ok {
		t.Fatalf("`level` should exists")
	} else if v1 != "INFO" {
		t.Fatalf("`level` shoule be `INFO`, but got: %v", v1)
	}
	if v2, ok := submatchMap["line"]; !ok {
		t.Fatalf("`line` should exists")
	} else if v2 != "74" {
		t.Fatalf("`line` shoule be `74`, but got: %v", v2)
	}
}

func TestRegexNamedSubMatch(t *testing.T) {
	t.Parallel()

	reg := regexp.MustCompile(`^(?P<time>.{23}) {0,}\| {0,}(?P<app>[^ ]+) {0,}\| {0,}(?P<level>[^ ]+) {0,}\| {0,}(?P<thread>[^ ]+) {0,}\| {0,}(?P<class>[^ ]+) {0,}\| {0,}(?P<line>\d+) {0,}([\|:] {0,}(?P<args>\{.*\})){0,1}([\|:] {0,}(?P<message>.*)){0,1}`)
	str := "2018-04-02 02:02:10.928 | sh-datamining | INFO | http-nio-8080-exec-80 | com.pateo.qingcloud.gateway.core.zuul.filters.post.LogFilter | 74 | xxx"
	submatchMap := map[string]string{}
	if err := RegexNamedSubMatch(reg, str, submatchMap); err != nil {
		t.Fatalf("got error: %+v", err)
	}

	for k, v := range submatchMap {
		fmt.Println(">>", k, ":", v)
	}

	if v1, ok := submatchMap["level"]; !ok {
		t.Fatalf("`level` should exists")
	} else if v1 != "INFO" {
		t.Fatalf("`level` shoule be `INFO`, but got: %v", v1)
	}
	if v2, ok := submatchMap["line"]; !ok {
		t.Fatalf("`line` should exists")
	} else if v2 != "74" {
		t.Fatalf("`line` shoule be `74`, but got: %v", v2)
	}
}

func ExampleRegexNamedSubMatch() {
	reg := regexp.MustCompile(`(?P<key>\d+.*)`)
	str := "12345abcde"
	groups := map[string]string{}
	if err := RegexNamedSubMatch(reg, str, groups); err != nil {
		log.Shared.Error("try to group match got error", zap.Error(err))
	}

	fmt.Println(groups)
	// Output: map[key:12345abcde]

}

func TestFlattenMap(t *testing.T) {
	data := map[string]any{}
	j := []byte(`{"a": "1", "b": {"c": 2, "d": {"e": 3}}, "f": 4, "g": {}}`)
	if err := json.Unmarshal(j, &data); err != nil {
		t.Fatalf("got error: %+v", err)
	}

	FlattenMap(data, ".")
	if data["a"].(string) != "1" {
		t.Fatalf("expect %v, got %v", "1", data["a"])
	}
	if int(data["b.c"].(float64)) != 2 {
		t.Fatalf("expect %v, got %v", 2, data["b.c"])
	}
	if int(data["b.d.e"].(float64)) != 3 {
		t.Fatalf("expect %v, got %v", 3, data["b.d.e"])
	}
	if int(data["f"].(float64)) != 4 {
		t.Fatalf("expect %v, got %v", 4, data["f"])
	}
	if _, ok := data["g"]; ok {
		t.Fatalf("g should not exists")
	}
}

func ExampleFlattenMap() {
	data := map[string]any{
		"a": "1",
		"b": map[string]any{
			"c": 2,
			"d": map[string]any{
				"e": 3,
			},
		},
	}
	FlattenMap(data, "__")
	fmt.Println(data)
	// Output: map[a:1 b__c:2 b__d__e:3]
}

func TestTriggerGC(t *testing.T) {
	TriggerGC()
	ForceGC()
}

func TestTemplateWithMap(t *testing.T) {
	t.Parallel()

	tpl := `123${k1} + ${k2}:${k-3} 22`
	data := map[string]any{
		"k1":  41,
		"k2":  "abc",
		"k-3": 213.11,
	}
	want := `12341 + abc:213.11 22`
	got := TemplateWithMap(tpl, data)
	if got != want {
		t.Fatalf("want `%v`, got `%v`", want, got)
	}
}

func TestURLMasking(t *testing.T) {
	t.Parallel()

	type testcase struct {
		input  string
		output string
	}

	var (
		ret  string
		mask = "*****"
	)
	for _, tc := range []*testcase{
		{
			"http://12ijij:3j23irj@jfjlwef.ffe.com",
			"http://12ijij:" + mask + "@jfjlwef.ffe.com",
		},
		{
			"https://12ijij:3j23irj@123.1221.14/13",
			"https://12ijij:" + mask + "@123.1221.14/13",
		},
	} {
		ret = URLMasking(tc.input, mask)
		if ret != tc.output {
			t.Fatalf("expect %v, got %v", tc.output, ret)
		}
	}
}

func ExampleURLMasking() {
	originURL := "http://12ijij:3j23irj@jfjlwef.ffe.com"
	newURL := URLMasking(originURL, "*****")
	fmt.Println(newURL)
	// Output: http://12ijij:*****@jfjlwef.ffe.com
}

func TestAutoGC(t *testing.T) {
	t.Parallel()

	var err error
	if err = log.Shared.ChangeLevel("debug"); err != nil {
		t.Fatalf("%+v", err)
	}

	var fp *os.File
	if fp, err = os.CreateTemp("", "test-gc*"); err != nil {
		t.Fatalf("%+v", err)
	}
	defer fp.Close()

	if _, err = fp.WriteString("123456789"); err != nil {
		t.Fatalf("%+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = AutoGC(ctx,
		WithGCMemRatio(85),
		WithGCMemLimitFilePath(fp.Name()),
	)
	require.NoError(t, err)
	<-ctx.Done()
	// t.Error()

	// case: test err arguments
	{
		err = AutoGC(ctx, WithGCMemRatio(-1))
		require.Error(t, err)

		err = AutoGC(ctx, WithGCMemRatio(0))
		require.Error(t, err)

		err = AutoGC(ctx, WithGCMemRatio(101))
		require.Error(t, err)

		err = AutoGC(ctx, WithGCMemLimitFilePath(RandomStringWithLength(100)))
		require.Error(t, err)
	}
}

func ExampleAutoGC() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := AutoGC(
		ctx,
		WithGCMemRatio(85), // default
		WithGCMemLimitFilePath("/sys/fs/cgroup/memory/memory.limit_in_bytes"), // default
	); err != nil {
		log.Shared.Error("enable autogc", zap.Error(err))
	}
}

func TestForceGCBlocking(t *testing.T) {
	t.Parallel()

	ForceGCBlocking()
}

func ExampleForceGCBlocking() {
	ForceGCBlocking()
}

func ExampleForceGCUnBlocking() {
	ForceGCUnBlocking()
}

func TestForceGCUnBlocking(t *testing.T) {
	t.Parallel()

	ForceGCUnBlocking()

	var pool errgroup.Group
	for i := 0; i < 1000; i++ {
		pool.Go(func() error {
			ForceGCUnBlocking()
			return nil
		})
	}

	require.NoError(t, pool.Wait())
}

func TestReflectSet(t *testing.T) {
	t.Parallel()

	type st struct{ A, B string }
	ss := []*st{{}, {}}
	nFields := reflect.ValueOf(ss[0]).Elem().NumField()
	vs := [][]string{{"x1", "y1"}, {"x2", "y2"}}

	for i, s := range ss {
		for j := 0; j < nFields; j++ {
			// if reflect.ValueOf(s).Type() != reflect.Ptr {
			// 	sp = &s
			// }
			reflect.ValueOf(s).Elem().Field(j).Set(reflect.ValueOf(vs[i][j]))
		}
	}

	t.Logf("s0: %+v", ss[0])
	t.Logf("s1: %+v", ss[1])
	// t.Error()
}

func ExampleSetStructFieldsBySlice() {
	type ST struct{ A, B string }
	var (
		err error
		ss  = []*ST{{}, {}}
		vs  = [][]string{
			{"x0", "y0"},
			{"x1", "y1"},
		}
	)
	if err = SetStructFieldsBySlice(ss, vs); err != nil {
		log.Shared.Error("set struct val", zap.Error(err))
		return
	}

	fmt.Printf("%+v\n", ss)
	// ss = []*ST{{A: "x0", B: "y0"}, {A: "x1", B: "y1"}}
}

func TestSetStructFieldsBySlice(t *testing.T) {
	t.Parallel()

	type ST struct{ A, B string }
	var (
		err error
		ss  = []*ST{
			{},
			{},
			{},
			{},
			{},
			{},
		}
		vs = [][]string{
			{"x0", "y0"},       // 0
			{"x1", "y1"},       // 1
			{},                 // 2
			{"x3", "y3", "z3"}, // 3
			{"x4"},             // 4
		}
	)
	if err = SetStructFieldsBySlice(ss, vs); err != nil {
		t.Fatalf("%+v", err)
	}

	t.Logf("s0: %+v", ss[0])
	t.Logf("s1: %+v", ss[1])
	t.Logf("s2: %+v", ss[2])
	t.Logf("s3: %+v", ss[3])
	t.Logf("s4: %+v", ss[4])
	t.Logf("s5: %+v", ss[5])

	if ss[0].A != "x0" ||
		ss[0].B != "y0" ||
		ss[1].A != "x1" ||
		ss[1].B != "y1" ||
		ss[2].A != "" ||
		ss[2].B != "" ||
		ss[3].A != "x3" ||
		ss[3].B != "y3" ||
		ss[4].A != "x4" ||
		ss[4].B != "" ||
		ss[5].A != "" ||
		ss[5].B != "" {
		t.Fatalf("incorrect")
	}

	// non-pointer struct
	ss2 := []ST{
		{},
		{},
	}
	if err = SetStructFieldsBySlice(ss2, vs); err != nil {
		t.Fatalf("%+v", err)
	}
	t.Logf("s0: %+v", ss2[0])
	t.Logf("s1: %+v", ss2[1])
	if ss2[0].A != "x0" ||
		ss2[0].B != "y0" ||
		ss2[1].A != "x1" ||
		ss2[1].B != "y1" {
		t.Fatalf("incorrect")
	}
}

func TestRunCMD(t *testing.T) {
	ctx := context.Background()
	type args struct {
		app  string
		args []string
	}
	tests := []struct {
		name       string
		args       args
		wantStdout []byte
		wantErr    bool
	}{
		{"sleep", args{"sleep", []string{"0.1"}}, []byte{}, false},
		{"sleep-err", args{"sleep", nil}, []byte("sleep: missing operand"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStdout, err := RunCMD(ctx, tt.args.app, tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCMD() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Contains(gotStdout, tt.wantStdout) {
				t.Errorf("RunCMD() = %s, want %s", gotStdout, tt.wantStdout)
			}
		})
	}
}

// linux pipe has 16MB default buffer
func TestRunCMDForHugeFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "run_cmd-*")
	require.NoError(t, err)
	defer os.Remove(dir)

	fpath := filepath.Join(dir, "test.txt")
	fp, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0664)
	require.NoError(t, err)

	for i := 0; i < 1024*18; i++ {
		_, err = fp.Write([]byte(RandomStringWithLength(1024)))
		require.NoError(t, err)
	}
	err = fp.Close()
	require.NoError(t, err)

	ctx := context.Background()
	out, err := RunCMD(ctx, "cat", fpath)
	require.NoError(t, err)
	require.Equal(t, len(out), 18*1024*1024)
}

func TestRemoveEmpty(t *testing.T) {
	type args struct {
		vs []string
	}
	tests := []struct {
		name  string
		args  args
		wantR []string
	}{
		{"0", args{[]string{"1"}}, []string{"1"}},
		{"1", args{[]string{"1", ""}}, []string{"1"}},
		{"2", args{[]string{"1", "", "  "}}, []string{"1"}},
		{"3", args{[]string{"1", "", "  ", "2", ""}}, []string{"1", "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotR := RemoveEmpty(tt.args.vs); !reflect.DeepEqual(gotR, tt.wantR) {
				t.Errorf("RemoveEmpty() = %v, want %v", gotR, tt.wantR)
			}
		})
	}
}

func TestTrimEleSpaceAndRemoveEmpty(t *testing.T) {
	type args struct {
		vs []string
	}
	tests := []struct {
		name  string
		args  args
		wantR []string
	}{
		{"0", args{[]string{"1"}}, []string{"1"}},
		{"1", args{[]string{"1", ""}}, []string{"1"}},
		{"2", args{[]string{"1", "", "  "}}, []string{"1"}},
		{"3", args{[]string{"1", "", "  ", "2", ""}}, []string{"1", "2"}},
		{"4", args{[]string{"1", "", "  ", "2   ", ""}}, []string{"1", "2"}},
		{"5", args{[]string{"1", "", "  ", "   2   ", ""}}, []string{"1", "2"}},
		{"6", args{[]string{"1", "", "  ", "   2", ""}}, []string{"1", "2"}},
		{"7", args{[]string{"   1", "", "  ", "   2", ""}}, []string{"1", "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotR := TrimEleSpaceAndRemoveEmpty(tt.args.vs); !reflect.DeepEqual(gotR, tt.wantR) {
				t.Errorf("TrimEleSpaceAndRemoveEmpty() = %v, want %v", gotR, tt.wantR)
			}
		})
	}
}

func TestGetStructFieldByName(t *testing.T) {
	t.Parallel()

	type foo struct {
		A string
		B *string
		C int
		E *string
	}

	s := "2"

	f := foo{"1", &s, 2, nil}
	if v := GetStructFieldByName(f, "A"); v.(string) != "1" {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(f, "B"); v.(*string) != &s {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(f, "C"); v.(int) != 2 {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(f, "D"); v != nil {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(f, "E"); v != nil {
		t.Fatalf("got %+v", v)
	}

	fi := &foo{"1", &s, 2, nil}
	if v := GetStructFieldByName(fi, "A"); v.(string) != "1" {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(fi, "B"); v.(*string) != &s {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(fi, "C"); v.(int) != 2 {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(fi, "D"); v != nil {
		t.Fatalf("got %+v", v)
	}
	if v := GetStructFieldByName(fi, "E"); v != nil {
		t.Fatalf("got %+v", v)
	}
}

/*
cpu: Intel(R) Core(TM) i7-4790 CPU @ 3.60GHz
Benchmark_Str2Bytes/normal_str2bytes-8         	  868298	      1156 ns/op	    1024 B/op	       1 allocs/op
Benchmark_Str2Bytes/normal_bytes2str-8         	 1000000	      1216 ns/op	    1024 B/op	       1 allocs/op
Benchmark_Str2Bytes/unsafe_str2bytes-8         	11335250	        92.66 ns/op	       0 B/op	       0 allocs/op
Benchmark_Str2Bytes/unsafe_bytes2str-8         	11320952	       106.2 ns/op	       0 B/op	       0 allocs/op
PASS
*/
func Benchmark_Str2Bytes(b *testing.B) {
	rawStr := RandomStringWithLength(1024)
	rawBytes := []byte(rawStr)
	b.Run("normal_str2bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = []byte(rawStr)
		}
	})
	b.Run("normal_bytes2str", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = string(rawBytes)
		}
	})
	b.Run("unsafe_str2bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Str2Bytes(rawStr)
		}
	})
	b.Run("unsafe_bytes2str", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Bytes2Str(rawBytes)
		}
	})
}

// func Test_ConvertMap(t *testing.T) {
// 	{
// 		input := map[any]string{"123": "23"}
// 		got := ConvertMap(input)
// 		t.Log(got)
// 		require.True(t, reflect.DeepEqual(map[string]any{"123": "23"}, got))
// 	}

// 	{
// 		input := map[any]int{"123": 23}
// 		got := ConvertMap(input)
// 		t.Log(got)
// 		require.True(t, reflect.DeepEqual(map[string]any{"123": 23}, got))
// 	}

// 	{
// 		input := map[any]uint{"123": 23}
// 		got := ConvertMap(input)
// 		t.Log(got)
// 		require.True(t, reflect.DeepEqual(map[string]any{"123": uint(23)}, got))
// 	}

// 	{
// 		input := map[string]int{"123": 23}
// 		got := ConvertMap(input)
// 		t.Log(got)
// 		require.True(t, reflect.DeepEqual(map[string]any{"123": 23}, got))
// 	}

// }

func TestConvert2Map(t *testing.T) {
	type args struct {
		inputMap any
	}
	tests := []struct {
		name string
		args args
		want map[string]any
	}{
		{"0", args{map[any]string{"123": "23"}}, map[string]any{"123": "23"}},
		{"1", args{map[any]int{"123": 23}}, map[string]any{"123": 23}},
		{"2", args{map[any]uint{"123": 23}}, map[string]any{"123": uint(23)}},
		{"3", args{map[string]uint{"123": 23}}, map[string]any{"123": uint(23)}},
		{"4", args{map[int]uint{123: 23}}, map[string]any{"123": uint(23)}},
		{"5", args{map[float32]string{float32(123): "23"}}, map[string]any{"123": "23"}},
		{"6", args{map[float32]int{float32(123): 23}}, map[string]any{"123": 23}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertMap2StringKey(tt.args.inputMap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStopSignal(t *testing.T) {
	stopCh := StopSignal(WithStopSignalCloseSignals(os.Interrupt, syscall.SIGTERM))
	select {
	case <-stopCh:
		t.Fatal("should not be closed")
	default:
	}

	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	require.NoError(t, err)

	_, ok := <-stopCh
	require.False(t, ok)

	// case: panic
	{
		ok := IsPanic(func() {
			_ = StopSignal(WithStopSignalCloseSignals())
		})
		require.True(t, ok)
	}
}

func TestBytes2Str(t *testing.T) {
	rawStr := RandomStringWithLength(1024)
	rawBytes := []byte(rawStr)
	str := Bytes2Str(rawBytes)
	require.Equal(t, rawStr, str)

	// case: bytes should changed by string
	{
		rawBytes[0] = '@'
		rawBytes[1] = 'a'
		rawBytes[2] = 'b'
		rawBytes[3] = 'c'
		require.Equal(t, string(rawBytes), str)
	}

	// case: Str2Bytes should return the same bytes struct
	{
		newBytes := Str2Bytes(str)
		require.Equal(t, fmt.Sprintf("%x", newBytes), fmt.Sprintf("%x", rawBytes))
	}
}

func Benchmark_slice(b *testing.B) {
	type foo struct {
		val string
	}
	payload := RandomStringWithLength(128)

	b.Run("[]struct append", func(b *testing.B) {
		var data []foo
		for i := 0; i < b.N; i++ {
			data = append(data, foo{val: payload})
		}

		b.Log(len(data))
	})

	b.Run("[]*struct append", func(b *testing.B) {
		var data []*foo
		for i := 0; i < b.N; i++ {
			data = append(data, &foo{val: payload})
		}

		b.Log(len(data))
	})

	b.Run("[]struct with prealloc", func(b *testing.B) {
		data := make([]foo, 100)
		for i := 0; i < b.N; i++ {
			data[i%100] = foo{val: payload}
		}
	})

	b.Run("[]*struct with prealloc", func(b *testing.B) {
		data := make([]*foo, 100)
		for i := 0; i < b.N; i++ {
			data[i%100] = &foo{val: payload}
		}
	})
}

func TestJSONMd5(t *testing.T) {
	type args struct {
		data any
	}
	type foo struct {
		Name string `json:"name"`
	}
	var nilArgs *foo
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"0", args{nil}, "", true},
		{"1", args{nilArgs}, "", true},
		{"2", args{foo{}}, "555dfa90763bd852d5dd9144887eed97", false},
		{"3", args{new(foo)}, "555dfa90763bd852d5dd9144887eed97", false},
		{"4", args{foo{""}}, "555dfa90763bd852d5dd9144887eed97", false},
		{"5", args{foo{Name: "a"}}, "88148e411b9b424a2e0ddf108cb02baa", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MD5JSON(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("MD5JSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MD5JSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNilInterface(t *testing.T) {
	type foo struct{}
	var f *foo
	var v any
	var tf foo

	v = f
	require.NotEqual(t, v, nil)
	require.True(t, NilInterface(v))
	require.False(t, NilInterface(tf))
	require.False(t, NilInterface(123))
	require.True(t, NilInterface(nil))
}

func TestPanicIfErr(t *testing.T) {
	PanicIfErr(nil)

	err := errors.New("yo")
	defer func() {
		deferErr := recover()
		require.Equal(t, err, deferErr)
	}()
	PanicIfErr(err)
}

func TestDedent(t *testing.T) {
	// t.Run("normal", func(t *testing.T) {
	// 	v := `
	// 	123
	// 	234
	// 	 345
	// 		222
	// 	`

	// 	dedent := Dedent(v, WithReplaceTabBySpaces(4))
	// 	require.Equal(t, "123\n234\n 345\n    222", dedent)
	// })

	t.Run("normal with blank lines", func(t *testing.T) {
		v := `
		123


		234

		 345
			222
		`

		dedent := Dedent(v, WithReplaceTabBySpaces(4))
		require.Equal(t, "123\n\n\n234\n\n 345\n    222", dedent)
	})

	t.Run("3 blanks", func(t *testing.T) {
		v := `
		123
		234
		 345	2
			222
		`

		dedent := Dedent(v, WithReplaceTabBySpaces(3))
		require.Equal(t, "123\n234\n 345\t2\n   222", dedent)
	})

	t.Run("shrink", func(t *testing.T) {
		v := `
		123
	   234
		`

		dedent := Dedent(v)
		require.Equal(t, " 123\n234", dedent)
	})

	t.Run("shrink with blank line", func(t *testing.T) {
		v := `
		123

	   234
		`

		dedent := Dedent(v)
		require.Equal(t, " 123\n\n234", dedent)
	})

}

func TestDeepClone(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		inner := []int{4, 5, 6}
		src := [][]int{inner}
		dst := DeepClone(src)

		inner[1] = 100
		require.NotEqual(t, src[0][1], dst[0][1])
	})

	type bar struct {
		A []string
	}

	type foo struct {
		A int
		B *bar
	}

	t.Run("struct", func(t *testing.T) {
		src := foo{
			A: 1,
			B: &bar{
				A: []string{"1", "2", "3"},
			},
		}
		dst := DeepClone(src)

		src.B.A[1] = "100"
		require.NotEqual(t, src.B.A[1], dst.B.A[1])
	})

	t.Run("*struct", func(t *testing.T) {
		src := &foo{
			A: 1,
			B: &bar{
				A: []string{"1", "2", "3"},
			},
		}
		dst := DeepClone(src)

		src.B.A[1] = "100"
		require.NotEqual(t, src.B.A[1], dst.B.A[1])
	})
}

type testCloseQuitlyStruct struct{}

func (f *testCloseQuitlyStruct) Close() error {
	return nil
}

func TestSilentClose(t *testing.T) {

	f := new(testCloseQuitlyStruct)
	SilentClose(f)
}

func TestContains(t *testing.T) {
	require.True(t, Contains([]string{"1", "2", "3"}, "2"))
	require.False(t, Contains([]string{"1", "2", "3"}, "4"))
	require.True(t, Contains([]int{1, 2, 3}, 2))
	require.False(t, Contains([]int{1, 2, 3}, 4))
}

func TestCtxKey(t *testing.T) {
	// Warning: should not use empty type as context key
	t.Run("empty type as key", func(t *testing.T) {
		type ctxKey struct{}

		var (
			keya, keyb ctxKey
		)

		ctx := context.Background()
		ctx = context.WithValue(ctx, keya, 123)

		require.Equal(t, 123, ctx.Value(keyb)) // <- this is incorrect
		require.Equal(t, 123, ctx.Value(keya))
	})

	t.Run("string as key", func(t *testing.T) {
		type ctxKey string

		var (
			keya ctxKey = "a"
			keyb ctxKey = "b"
		)

		ctx := context.Background()
		ctx = context.WithValue(ctx, keya, 123)

		require.Nil(t, ctx.Value(keyb))
		require.Equal(t, 123, ctx.Value(keya))
	})

	// different type with same value will not overwrite each other
	t.Run("different type string as key", func(t *testing.T) {
		type ctxKeyA string
		type ctxKeyB string

		var (
			keya ctxKeyA = "a"
			keyb ctxKeyB = "a"
		)

		ctx := context.Background()
		ctx = context.WithValue(ctx, keya, 123)

		require.Nil(t, ctx.Value(keyb))
		require.Equal(t, 123, ctx.Value(keya))

		ctx = context.WithValue(ctx, keyb, 321)
		require.Equal(t, 123, ctx.Value(keya))
	})
}

func TestStructFieldRequired(t *testing.T) {
	v := struct {
		A  string
		AP *string
		B  int
		BB float64
	}{
		A: "123",
		B: 123,
	}

	require.NoError(t, NotEmpty(v.A, "A"))
	require.NoError(t, NotEmpty(&v.A, "*A"))
	require.ErrorContains(t, NotEmpty(v.AP, "AP"), "is empty pointer")

	emptyString := ""
	v.AP = &emptyString
	require.ErrorContains(t, NotEmpty(v.AP, "AP"), "is point to empty elem")
	require.ErrorContains(t, NotEmpty(*v.AP, "*AP"), "is empty elem")

	require.NoError(t, NotEmpty(v.B, "B"))
	require.ErrorContains(t, NotEmpty(v.BB, "BB"), "is empty elem")
}

func TestOptionalVal(t *testing.T) {
	v := struct {
		A  string
		AP *string
		B  int
		BB float64
	}{
		A: "123",
		B: 123,
	}

	optStr := "laisky"
	optInt := 123
	optFloat64 := float64(123)

	v.A = OptionalVal(&v.A, optStr)
	require.Equal(t, v.A, "123")

	v.AP = OptionalVal(&v.AP, &optStr)
	require.Equal(t, v.AP, &optStr)

	v.B = OptionalVal(&v.B, optInt)
	require.Equal(t, v.B, optInt)

	v.BB = OptionalVal(&v.BB, optFloat64)
	require.Equal(t, v.BB, optFloat64)
}

func TestRunCMDWithEnv(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx  context.Context
		app  string
		args []string
		envs []string
	}
	tests := []struct {
		name       string
		args       args
		wantStdout []byte
		wantErr    bool
	}{
		// {"", args{ctx, `/bin/env`, nil, []string{"FOO=BAR"}}, []byte("BAR"), false},
		{"", args{ctx, `/bin/bash`, []string{"-c", "echo $FOO"}, []string{"FOO=BAR"}}, []byte("BAR\n"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStdout, err := RunCMDWithEnv(tt.args.ctx, tt.args.app, tt.args.args, tt.args.envs)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCMDWithEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotStdout, tt.wantStdout) {
				t.Errorf("RunCMDWithEnv() = %q, want %q", string(gotStdout), string(tt.wantStdout))
			}
		})
	}
}

func TestCostSecs(t *testing.T) {
	d := time.Millisecond * 351
	v := CostSecs(d)
	require.Equal(t, "0.35s", v)
}

func TestPipeline(t *testing.T) {
	f1 := func(v *int) error { (*v)++; return nil }
	f2 := func(v *int) error { (*v) += 2; return nil }
	v := 0

	gotv, err := Pipeline([]func(*int) error{f1, f2}, &v)
	require.NoError(t, err)
	require.Equal(t, 3, v)
	require.Equal(t, 3, *gotv)
}

func Test_singleflight(t *testing.T) {
	var n int
	key := RandomStringWithLength(10)
	internalSFG.Do(key, func() (interface{}, error) { n++; return nil, nil })
	internalSFG.Do(key, func() (interface{}, error) { n++; return nil, nil })
	internalSFG.Do(key, func() (interface{}, error) { n++; return nil, nil })
	require.Equal(t, 3, n)
}

func TestUUID1(t *testing.T) {
	t.Run("goroutine", func(t *testing.T) {
		var (
			mu   sync.Mutex
			uids []string
		)

		var pool errgroup.Group
		for i := 0; i < 10000; i++ {
			pool.Go(func() error {
				uid := UUID1()

				mu.Lock()
				uids = append(uids, uid)
				mu.Unlock()
				return nil
			})
		}

		require.NoError(t, pool.Wait())
		require.Len(t, uids, 10000)

		st := mapset.NewSet(uids...)
		require.Equal(t, len(uids), st.Cardinality())
	})

	t.Run("monotonically time", func(t *testing.T) {
		var uids []string
		for i := 0; i < 10000; i++ {
			uid := UUID1()
			uids = append(uids, uid)
		}

		var lastTime time.Time
		for i := range uids {
			uid, err := uuid.Parse(uids[i])
			require.NoError(t, err)

			ct := time.Unix(uid.Time().UnixTime())
			if lastTime.IsZero() {
				lastTime = ct
				continue
			}

			require.GreaterOrEqual(t, ct, lastTime)
		}
	})
}

func TestDelayer_Wait(t *testing.T) {
	startAt := time.Now()
	delay := 10 * time.Millisecond

	func() {
		defer NewDelay(delay).Wait()

		require.Less(t, time.Since(startAt), time.Millisecond)
	}()
	require.GreaterOrEqual(t, time.Since(startAt), delay)
}

func ExampleNewDelay() {
	startAt := time.Now()
	delay := 10 * time.Millisecond

	func() {
		defer NewDelay(delay).Wait()
	}()

	fmt.Println(time.Since(startAt) >= delay)
	// Output: true
}

func Test_FileHashSharding(t *testing.T) {
	type args struct {
		fname string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"0", args{"0"}, "b6/58/0"},
		{"1", args{"1"}, "35/6a/1"},
		{"2", args{"2"}, "da/4b/2"},
		{"3", args{"fwlfjlwefjjew.txt"}, "65/21/fwlfjlwefjjew.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileHashSharding(tt.args.fname); got != tt.want {
				t.Errorf("fileHashSharding() = %v, cccccbedrjejgvblfehhlfldkkcjucdhifutrjiffurbcccccbedrjejdufvlcdvhdevurcgcdkhfrrvkkuvictgwant %v", got, tt.want)
			}
		})
	}
}

func Test_Sum(t *testing.T) {
	r1 := []byte("a")
	r2 := []byte("b")
	r3 := []byte("c")

	t.Run("sum", func(t *testing.T) {
		hasher := sha256.New()
		hasher.Sum(r1)
		hasher.Sum(r2)
		hasher.Sum(r3)
		got := hasher.Sum(nil)
		require.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hex.EncodeToString(got))
	})

	t.Run("write", func(t *testing.T) {
		hasher := sha256.New()
		hasher.Write(r1)
		hasher.Write(r2)
		hasher.Write(r3)
		got := hasher.Sum(nil)
		require.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", hex.EncodeToString(got))
	})

	// sum will not change the state of the hasher
	t.Run("write & sum", func(t *testing.T) {
		hasher := sha256.New()
		hasher.Write(r1)
		hasher.Sum(r1)
		hasher.Write(r2)
		hasher.Sum(r2)
		hasher.Write(r3)
		hasher.Sum(r3)
		got := hasher.Sum(nil)
		require.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", hex.EncodeToString(got))
	})

}

type testlog struct {
	content string
}

// Error test error
func (l *testlog) Error(msg string, _ ...zap.Field) {
	l.content = msg
}

type tt struct{}

// Close test close with error
func (t *tt) Close() error {
	return errors.Errorf("close error")
}

// Flush test flush with error
func (t *tt) Flush() error {
	return errors.Errorf("flush error")
}

func TestCloseWithLog(t *testing.T) {
	logger := new(testlog)
	tc := new(tt)

	CloseWithLog(tc, nil)
	CloseWithLog(tc, logger)
	require.Equal(t, "close ins", logger.content)

	FlushWithLog(tc, nil)
	FlushWithLog(tc, logger)
	require.Equal(t, "flush ins", logger.content)
}

func TestRunCMD2(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "TestRunCMD2-*")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// write shell file
	execFile := filepath.Join(dir, "test.sh")
	err = os.WriteFile(execFile, []byte(Dedent(
		`#!/bin/bash

		while true; do
			echo "hello"
			sleep 0.1
		done`)), 0755)
	require.NoError(t, err)

	// run shell file
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		stdoutMu sync.Mutex
		stdout   []string
	)
	stdoutHandler := func(msg string) {
		stdoutMu.Lock()
		defer stdoutMu.Unlock()
		stdout = append(stdout, msg)
	}
	go RunCMD2(ctx, "/bin/bash", []string{execFile}, nil, stdoutHandler, nil)
	time.Sleep(time.Second)
	cancel()

	stdoutMu.Lock()
	require.Greater(t, len(stdout), 5)
	require.Contains(t, stdout[0], "hello")
	stdoutMu.Unlock()
}

func TestIsPanic2(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		panicMsg := "test panic"
		f := func() {
			panic(panicMsg)
		}
		err := IsPanic2(f)
		if err == nil {
			t.Fatal("expected an error, but got nil")
		}
		if !strings.Contains(err.Error(), panicMsg) {
			t.Fatalf("expected error message to contain %q, but got %q", panicMsg, err.Error())
		}
	})

	t.Run("no panic", func(t *testing.T) {
		f := func() {}
		err := IsPanic2(f)
		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}
	})
}
func TestUUID4(t *testing.T) {
	t.Parallel()
	val := UUID4()
	_, err := uuid.Parse(val)
	require.NoError(t, err)

	t.Run("unique", func(t *testing.T) {
		t.Parallel()

		var (
			mu      sync.Mutex
			pool    errgroup.Group
			uuidSet = map[string]struct{}{}
		)
		for i := 0; i < 10; i++ {
			pool.Go(func() error {
				for i := 0; i < 1000; i++ {
					uuid := UUID4()
					mu.Lock()
					uuidSet[uuid] = struct{}{}
					mu.Unlock()
				}

				return nil
			})
		}

		require.NoError(t, pool.Wait())
		require.Len(t, uuidSet, 10000)
	})
}

func TestUUID7(t *testing.T) {
	t.Parallel()
	var (
		mu      sync.Mutex
		pool    errgroup.Group
		uuidSet = map[string]struct{}{}
	)
	for i := 0; i < 10; i++ {
		pool.Go(func() error {
			for i := 0; i < 1000; i++ {
				uuid := UUID7()
				mu.Lock()
				uuidSet[uuid] = struct{}{}
				mu.Unlock()
			}

			return nil
		})
	}

	require.NoError(t, pool.Wait())
	require.Len(t, uuidSet, 10000)

	for val := range uuidSet {
		_, err := ParseUUID7(val)
		require.NoError(t, err)
	}

	t.Run("order", func(t *testing.T) {
		t.Parallel()

		v1 := UUID7()
		time.Sleep(time.Millisecond)
		v2 := UUID7()
		require.Greater(t, v2, v1)
	})
}

func TestCopy(t *testing.T) {
	raw := []byte("hello, world")
	raw = raw[: len(raw)-1 : len(raw)]

	padded1 := make([]byte, 16)
	copy(padded1, raw)

	padded2 := append(raw, bytes.Repeat([]byte{0x00}, 16-len(raw))...)
	padded3 := append(raw, make([]byte, 16-len(raw))...)

	require.Len(t, padded1, 16)
	require.Equal(t, padded1, padded2)
	require.Equal(t, padded1, padded3)
}
func TestReverseSlice(t *testing.T) {
	t.Parallel()

	// Test cases
	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name:     "Empty Slice",
			input:    []interface{}{},
			expected: []interface{}{},
		},
		{
			name:     "Slice with Even Number of Elements",
			input:    []interface{}{1, 2, 3, 4},
			expected: []interface{}{4, 3, 2, 1},
		},
		{
			name:     "Slice with Odd Number of Elements",
			input:    []interface{}{"a", "b", "c", "d", "e"},
			expected: []interface{}{"e", "d", "c", "b", "a"},
		},
	}

	// Run tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Make a copy of the input slice
			input := make([]interface{}, len(test.input))
			copy(input, test.input)

			// Reverse the slice
			ReverseSlice(input)

			// Check if the reversed slice matches the expected result
			if !reflect.DeepEqual(input, test.expected) {
				t.Errorf("unexpected result, got: %v, want: %v", input, test.expected)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vs   []string
		want []string
	}{
		{
			name: "Empty input",
			vs:   []string{},
			want: []string{},
		},
		{
			name: "No duplicates",
			vs:   []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "Duplicates",
			vs:   []string{"a", "b", "a", "c", "b"},
			want: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueStrings(tt.vs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("random", func(t *testing.T) {
		t.Parallel()

		orig := []string{}
		for i := 0; i < 100000; i++ {
			orig = append(orig, RandomStringWithLength(2))
		}
		t.Logf("generate length : %d", len(orig))
		orig = UniqueStrings(orig)
		t.Logf("after unique length : %d", len(orig))
		m := map[string]bool{}
		var ok bool
		for _, v := range orig {
			if _, ok = m[v]; ok {
				t.Fatalf("duplicate: %v", v)
			} else {
				m[v] = ok
			}
		}
	})
}

// cpu: Intel(R) Xeon(R) Gold 5320 CPU @ 2.20GHz
// Benchmark_UniqueStrings
// Benchmark_UniqueStrings/100000
// Benchmark_UniqueStrings/100000-104         	1000000000	         0.003633 ns/op	       0 B/op	       0 allocs/op
func Benchmark_UniqueStrings(b *testing.B) {
	orig := []string{}
	for i := 0; i < b.N; i++ {
		for i := 0; i < 100000; i++ {
			orig = append(orig, RandomStringWithLength(2))
		}

		b.ResetTimer()
	}

	b.Run("100000", func(b *testing.B) {
		orig = UniqueStrings(orig)
	})
}

func TestRemoveEmptyVal(t *testing.T) {
	t.Parallel()

	t.Run("Empty map", func(t *testing.T) {
		m1 := map[string]any{}
		want1 := map[string]any{}
		got1 := RemoveEmptyVal(m1)
		if !reflect.DeepEqual(got1, want1) {
			t.Errorf("Test case 1 failed: got %v, want %v", got1, want1)
		}
	})

	t.Run("Map with non-empty values", func(t *testing.T) {
		m2 := map[string]any{
			"a": 1,
			"b": "hello",
			"c": map[string]any{
				"d": 2,
				"e": "",
			},
		}
		want2 := map[string]any{
			"a": 1,
			"b": "hello",
			"c": map[string]any{
				"d": 2,
			},
		}
		got2 := RemoveEmptyVal(m2)
		if !reflect.DeepEqual(got2, want2) {
			t.Errorf("Test case 2 failed: got %v, want %v", got2, want2)
		}
	})

	t.Run("Map with nested empty maps", func(t *testing.T) {
		m3 := map[string]any{
			"a": map[string]any{},
			"b": map[string]any{
				"c": map[string]any{},
			},
		}
		want3 := map[string]any{}
		got3 := RemoveEmptyVal(m3)
		if !reflect.DeepEqual(got3, want3) {
			t.Errorf("Test case 3 failed: got %v, want %v", got3, want3)
		}
	})

	t.Run("Map with nested non-empty maps", func(t *testing.T) {
		m3 := map[string]any{
			"a": map[string]any{},
			"b": map[string]any{
				"c": map[string]any{},
				"d": 123,
			},
			"e": map[string]string{},
			"f": map[string]string{
				"g": "123",
			},
		}
		want3 := map[string]any{
			"b": map[string]any{
				"d": 123,
			},
			"f": map[string]string{
				"g": "123",
			},
		}
		got3 := RemoveEmptyVal(m3)
		if !reflect.DeepEqual(got3, want3) {
			t.Errorf("Test case 3 failed: got %v, want %v", got3, want3)
		}
	})

	t.Run("map with empty slice", func(t *testing.T) {
		m4 := map[string]any{
			"a": []string{},
			"b": 123,
		}
		want4 := map[string]any{"b": 123}
		got4 := RemoveEmptyVal(m4)
		if !reflect.DeepEqual(got4, want4) {
			t.Errorf("Test case 4 failed: got %v, want %v", got4, want4)
		}
	})

	t.Run("map with nil", func(t *testing.T) {
		m5 := map[string]any{
			"a": nil,
		}
		want5 := map[string]any{}
		got5 := RemoveEmptyVal(m5)
		if !reflect.DeepEqual(got5, want5) {
			t.Errorf("Test case 5 failed: got %v, want %v", got5, want5)
		}
	})
}

func TestSanitizeCMDArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expected    []string
		expectedErr error
	}{
		{
			name:        "0",
			args:        []string{"arg1", "arg2", "arg3"},
			expected:    []string{"arg1", "arg2", "arg3"},
			expectedErr: nil,
		},
		{
			name:        "1",
			args:        []string{"arg1", "arg2$", "arg3"},
			expected:    []string{"arg1", "arg2$", "arg3"},
			expectedErr: nil,
		},
		{
			name:        "2",
			args:        []string{"arg1", "arg2$(echo hello)", "arg3"},
			expected:    nil,
			expectedErr: errors.New("invalid command substitution in args"),
		},
		{
			name:        "3",
			args:        []string{"  arg1  ", "arg2  ", "  arg3"},
			expected:    []string{"arg1", "arg2", "arg3"},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sanitizedArgs, err := SanitizeCMDArgs(test.args)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
				return
			}

			isDeepEqual := reflect.DeepEqual(sanitizedArgs, test.expected)
			require.True(t, isDeepEqual, "sanitizedArgs: %v, expected: %v",
				sanitizedArgs, test.expected)
		})
	}
}

func TestCombineSortedChain(t *testing.T) {
	t.Parallel()

	t.Run("Empty input channels", func(t *testing.T) {
		t.Parallel()
		_, err := CombineSortedChain[int](common.SortOrderAsc)
		require.Error(t, err)
		require.Equal(t, "chans cannot be empty", err.Error())
	})

	t.Run("single input channel", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int)
		go func() {
			defer close(ch)
			ch <- 1
		}()

		result, err := CombineSortedChain(common.SortOrderAsc, ch)
		require.NoError(t, err)
		require.Equal(t, 1, <-result)
	})

	t.Run("multiple input channels", func(t *testing.T) {
		t.Parallel()
		ch1 := make(chan int)
		ch2 := make(chan int)
		ch3 := make(chan int)

		go func() {
			defer close(ch1)
			ch1 <- 1
		}()

		go func() {
			defer close(ch2)
			ch2 <- 3
		}()

		go func() {
			defer close(ch3)
			ch3 <- 2
		}()

		result, err := CombineSortedChain(common.SortOrderAsc, ch1, ch2, ch3)
		require.NoError(t, err)
		require.Equal(t, 1, <-result)
		require.Equal(t, 2, <-result)
		require.Equal(t, 3, <-result)
	})

	t.Run("multiple asc input channels with huge difference", func(t *testing.T) {
		t.Parallel()
		ch1 := make(chan int)
		ch2 := make(chan int)
		ch3 := make(chan int)

		go func() {
			defer close(ch1)
			for i := 0; i < 1000; i++ {
				ch1 <- i
			}
		}()

		go func() {
			defer close(ch2)
			for i := 0; i < 1000; i++ {
				ch2 <- i + 10000
			}
		}()

		go func() {
			defer close(ch3)
			for i := 0; i < 1000; i++ {
				ch3 <- i + 20000
			}
		}()

		result, err := CombineSortedChain(common.SortOrderAsc, ch1, ch2, ch3)
		require.NoError(t, err)
		latest, ok := <-result
		require.True(t, ok)

		for {
			cur, ok := <-result
			if !ok {
				return
			}

			require.Greater(t, cur, latest)
			latest = cur
		}
	})

	t.Run("multiple asc input channels with overlay", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch1 := make(chan int)
		ch2 := make(chan int)
		ch3 := make(chan int)

		go func() {
			defer close(ch1)
			i := rand.Intn(10000)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				ch1 <- i
				i += rand.Intn(1000)
			}
		}()

		go func() {
			defer close(ch2)
			i := rand.Intn(10000)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				ch2 <- i
				i += rand.Intn(1000)
			}
		}()

		go func() {
			defer close(ch3)
			i := rand.Intn(10000)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				ch3 <- i
				i += rand.Intn(1000)
			}
		}()

		result, err := CombineSortedChain(common.SortOrderAsc, ch1, ch2, ch3)
		require.NoError(t, err)

		latest, ok := <-result
		require.True(t, ok)

		for i := 0; i < 100000; i++ {
			cur, ok := <-result
			if !ok {
				return
			}

			require.LessOrEqualf(t, latest, cur, "%d", i)
			latest = cur
		}
	})

	t.Run("multiple desc input channels with huge difference", func(t *testing.T) {
		t.Parallel()
		ch1 := make(chan int)
		ch2 := make(chan int)
		ch3 := make(chan int)

		go func() {
			defer close(ch1)
			for i := 0; i < 1000; i++ {
				ch1 <- 10000 - i
			}
		}()

		go func() {
			defer close(ch2)
			for i := 0; i < 1000; i++ {
				ch2 <- 20000 - i
			}
		}()

		go func() {
			defer close(ch3)
			for i := 0; i < 1000; i++ {
				ch3 <- 30000 - i
			}
		}()

		result, err := CombineSortedChain(common.SortOrderDesc, ch1, ch2, ch3)
		require.NoError(t, err)
		latest, ok := <-result
		require.True(t, ok)

		for {
			cur, ok := <-result
			if !ok {
				return
			}

			require.Less(t, cur, latest)
			latest = cur
		}
	})
}

// cpu: Intel(R) Xeon(R) Gold 5320 CPU @ 2.20GHz
// Benchmark_CombineSortedChain
// Benchmark_CombineSortedChain/CombineSortedChain
// Benchmark_CombineSortedChain/CombineSortedChain-104         	   52828	     23706 ns/op	      56 B/op	       3 allocs/op
func Benchmark_CombineSortedChain(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	go func() {
		defer close(ch1)
		i := rand.Intn(10000)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			ch1 <- i
			i += rand.Intn(1000)
		}
	}()

	go func() {
		defer close(ch2)
		i := rand.Intn(10000)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			ch2 <- i
			i += rand.Intn(1000)
		}
	}()

	go func() {
		defer close(ch3)
		i := rand.Intn(10000)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			ch3 <- i
			i += rand.Intn(1000)
		}
	}()

	result, err := CombineSortedChain(common.SortOrderAsc, ch1, ch2, ch3)
	require.NoError(b, err)

	b.ResetTimer()
	b.Run("CombineSortedChain", func(b *testing.B) {
		latest, ok := <-result
		require.True(b, ok)

		for i := 0; i < b.N; i++ {
			cur, ok := <-result
			if !ok {
				return
			}

			require.LessOrEqual(b, latest, cur)
			latest = cur
		}
	})
}

func TestFilterSlice(t *testing.T) {
	t.Parallel()

	// Test case 1: Filter even numbers
	s1 := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	f1 := func(v int) bool {
		return v%2 == 0
	}
	expected1 := []int{2, 4, 6, 8, 10}
	result1 := FilterSlice(s1, f1)
	if !reflect.DeepEqual(result1, expected1) {
		t.Errorf("FilterSlice failed for test case 1, expected: %v, got: %v", expected1, result1)
	}

	// Test case 2: Filter strings with length greater than 3
	s2 := []string{"apple", "banana", "cat", "dog", "elephant"}
	f2 := func(v string) bool {
		return len(v) > 3
	}
	expected2 := []string{"apple", "banana", "elephant"}
	result2 := FilterSlice(s2, f2)
	if !reflect.DeepEqual(result2, expected2) {
		t.Errorf("FilterSlice failed for test case 2, expected: %v, got: %v", expected2, result2)
	}

	// Test case 3: Filter structs based on a condition
	type Person struct {
		Name   string
		Age    int
		Gender string
	}
	s3 := []Person{
		{Name: "Alice", Age: 25, Gender: "Female"},
		{Name: "Bob", Age: 30, Gender: "Male"},
		{Name: "Charlie", Age: 20, Gender: "Male"},
		{Name: "Diana", Age: 35, Gender: "Female"},
	}
	f3 := func(v Person) bool {
		return v.Age > 25 && v.Gender == "Female"
	}
	expected3 := []Person{
		{Name: "Diana", Age: 35, Gender: "Female"},
	}
	result3 := FilterSlice(s3, f3)
	if !reflect.DeepEqual(result3, expected3) {
		t.Errorf("FilterSlice failed for test case 3, expected: %v, got: %v", expected3, result3)
	}
}

func TestPrettyBuildInfo(t *testing.T) {
	t.Parallel()

	t.Run("no deps", func(t *testing.T) {
		ret := PrettyBuildInfo()
		require.Contains(t, ret, `"GoVersion"`)
		require.Contains(t, ret, `"Main":`)
		require.Contains(t, ret, `"Deps": null`)
	})

	t.Run("with deps", func(t *testing.T) {
		ret := PrettyBuildInfo(
			WithPrettyBuildInfoDeps(),
		)
		require.Contains(t, ret, `"GoVersion"`)
		require.Contains(t, ret, `"Main":`)
		require.Contains(t, ret, `"Deps":`)
	})
}

func TestGetEnvInsensitive(t *testing.T) {
	t.Parallel()

	// Set up test environment variables
	os.Setenv("KEY1", "value1")
	os.Setenv("key2", "value2")
	os.Setenv("Key3", "value3")
	os.Setenv("KEY4", "value4")
	os.Setenv("key4", "value5")
	os.Setenv("Key4", "value6")

	require.True(t, strings.EqualFold("key1", "KEY1"))

	// Test case: Key not found
	expected1 := []string{}
	result1 := GetEnvInsensitive("key")
	require.ElementsMatch(t, expected1, result1)

	// Test case: Case-sensitive key match
	expected2 := []string{"value4", "value5", "value6"}
	result2 := GetEnvInsensitive("KEY4")
	require.ElementsMatch(t, expected2, result2)

	expected3 := []string{}
	result3 := GetEnvInsensitive("nonexistent")
	require.ElementsMatch(t, expected3, result3)
}

func TestParseObjectIdentifier(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input string
		want  asn1.ObjectIdentifier
		err   string
	}{
		{
			input: "1.2.3",
			want:  asn1.ObjectIdentifier{1, 2, 3},
			err:   "",
		},
		{
			input: "1.2.3.4.5",
			want:  asn1.ObjectIdentifier{1, 2, 3, 4, 5},
			err:   "",
		},
		{
			input: "1.2.3.4.55555",
			want:  asn1.ObjectIdentifier{1, 2, 3, 4, 55555},
			err:   "",
		},
		{
			input: "1.2.3.4.55555.2",
			want:  asn1.ObjectIdentifier{1, 2, 3, 4, 55555, 2},
			err:   "",
		},
		{
			input: "1.2.3.4.55555.-2",
			want:  nil,
			err:   fmt.Sprintf("invalid oid format"),
		},
		{
			input: "1.2.a",
			want:  nil,
			err:   "invalid oid format",
		},
		{
			input: "1.2.3.",
			want:  nil,
			err:   "invalid oid format",
		},
		{
			input: "1.2.3.4.5.",
			want:  nil,
			err:   "invalid oid format",
		},
	}

	for _, tc := range testCases {
		got, err := ParseObjectIdentifier(tc.input)
		if tc.err == "" {
			require.Equalf(t, tc.want, got, "input: %q", tc.input)
			continue
		}

		require.ErrorContainsf(t, err, tc.err, "input: %q", tc.input)
	}
}

func TestNewHasPrefixWithMagic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix []byte
		input  []byte
		want   bool
	}{
		{
			name:   "8-byte prefix match",
			prefix: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			input:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			want:   true,
		},
		{
			name:   "8-byte prefix no match",
			prefix: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			input:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x09},
			want:   false,
		},
		{
			name:   "4-byte prefix match",
			prefix: []byte{0x01, 0x02, 0x03, 0x04},
			input:  []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			want:   true,
		},
		{
			name:   "4-byte prefix no match",
			prefix: []byte{0x01, 0x02, 0x03, 0x04},
			input:  []byte{0x01, 0x02, 0x03, 0x05},
			want:   false,
		},
		{
			name:   "2-byte prefix match",
			prefix: []byte{0x01, 0x02},
			input:  []byte{0x01, 0x02, 0x03},
			want:   true,
		},
		{
			name:   "2-byte prefix no match",
			prefix: []byte{0x01, 0x02},
			input:  []byte{0x01, 0x03},
			want:   false,
		},
		{
			name:   "empty prefix",
			prefix: []byte{},
			input:  []byte{0x01, 0x02},
			want:   true,
		},
		{
			name:   "non-matching prefix",
			prefix: []byte{0x01, 0x02, 0x03},
			input:  []byte{0x04, 0x05, 0x06},
			want:   false,
		},
		{
			name:   "longer prefix",
			prefix: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			input:  []byte{0x01, 0x02, 0x03, 0x04},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasPrefix := NewHasPrefixWithMagic(tt.prefix)
			if got := hasPrefix(tt.input); got != tt.want {
				t.Errorf("input: %x, prefix: %x, want: %v, got: %v",
					tt.input, tt.prefix, tt.want, got)
			}
		})
	}
}

// cpu: AMD Ryzen 7 5700G with Radeon Graphics
// Benchmark_HasPrefix/std-8         	404345066	         3.031 ns/op	       0 B/op	       0 allocs/op
// Benchmark_HasPrefix/custom-8      	562408310	         2.133 ns/op	       0 B/op	       0 allocs/op
// PASS
func Benchmark_HasPrefix(b *testing.B) {
	val := []byte("hello, world")
	prefix := []byte("hell")

	b.Run("std", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bytes.HasPrefix(val, prefix)
		}
	})

	hasprefix := NewHasPrefixWithMagic(prefix)
	b.Run("custom", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hasprefix(val)
		}
	})
}
