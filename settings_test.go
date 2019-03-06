package utils_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Laisky/go-utils"
	zap "github.com/Laisky/zap"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func ExampleSettings() {
	// read settings from yml file
	pflag.String("config", "/etc/go-ramjet/settings", "config file directory path")
	pflag.Parse()

	// bind pflags to settings
	utils.Settings.BindPFlags(pflag.CommandLine)

	// use
	utils.Settings.Get("xxx")
	utils.Settings.GetString("xxx")
	utils.Settings.GetStringSlice("xxx")
	utils.Settings.GetBool("xxx")

	utils.Settings.Set("name", "val")
}

func TestSettings(t *testing.T) {
	var (
		err error
		st  = []byte(`---
key1: val1
key2: val2
key3: val3`)
	)

	dirName, err := ioutil.TempDir("", "go-utils-test")
	if err != nil {
		t.Fatalf("try to create tmp dir got error: %+v", err)
	}
	fp, err := os.Create(filepath.Join(dirName, "settings.yml"))
	if err != nil {
		t.Fatalf("try to create tmp file got error: %+v", err)
	}
	t.Logf("create file: %v", fp.Name())
	// defer os.RemoveAll(dirName)

	fp.Write(st)
	fp.Close()

	t.Logf("load settings from: %v", dirName)
	if err = utils.Settings.Setup(dirName); err != nil {
		t.Fatalf("setup settings got error: %+v", err)
	}

	t.Logf(">> key1: %+v", viper.Get("key1"))
	fp, err = os.Open(fp.Name())
	defer fp.Close()
	if b, err := ioutil.ReadAll(fp); err != nil {
		t.Fatalf("try to read tmp file got error: %+v", err)
	} else {
		t.Logf("file content: %v", string(b))
	}

	cases := map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}
	var got string
	for k, expect := range cases {
		got = utils.Settings.GetString(k)
		if got != expect {
			t.Errorf("load %v, expect %v, got %v", k, expect, got)
		}
	}
}

// dependends on remote config-s  erver
func TestSetupFromConfigServerWithRawYaml(t *testing.T) {
	fakedata := map[string]interface{}{
		"name":     "app",
		"profiles": []string{"profile"},
		"label":    "label",
		"version":  "12345",
		"propertySources": []map[string]interface{}{
			map[string]interface{}{
				"name": "config name",
				"source": map[string]string{
					"profile": "profile",
					"raw": `
a:
  b: 123
  c: abc
  d:
    - 1
    - 2
  e: true`,
				},
			},
		},
	}

	jb, err := json.Marshal(fakedata)
	if err != nil {
		utils.Logger.Panic("try to marshal fake data got error", zap.Error(err))
	}
	addr := RunMockConfigSrv(jb)

	cfg := &utils.ConfigServerCfg{
		URL:     "http://" + addr,
		Profile: "profile",
		Label:   "label",
		App:     "app",
	}
	if err := utils.Settings.SetupFromConfigServerWithRawYaml(cfg, "raw"); err != nil {
		t.Fatalf("got error: %+v", err)
	}
	for k, vi := range map[string]interface{}{
		"a.b": 123,
		"a.c": "abc",
		"a.d": []string{"1", "2"},
		"a.e": true,
	} {
		switch val := vi.(type) {
		case string:
			if utils.Settings.GetString(k) != val {
				t.Fatalf("`%v` should be `%v`, but got %+v", k, val, utils.Settings.Get(k))
			}
		case int:
			if utils.Settings.GetInt(k) != val {
				t.Fatalf("`%v` should be `%v`, but got %+v", k, val, utils.Settings.Get(k))
			}
		case []string:
			vs := utils.Settings.GetStringSlice(k)
			if len(vs) != 2 ||
				vs[0] != val[0] ||
				vs[1] != val[1] {
				t.Fatalf("`%v` should be `%v`, but got %+v", k, val, utils.Settings.Get(k))
			}
		case bool:
			if utils.Settings.GetBool(k) != val {
				t.Fatalf("`%v` should be `%v`, but got %+v", k, val, utils.Settings.Get(k))
			}
		default:
			t.Fatal("unknown type")
		}
	}
}
