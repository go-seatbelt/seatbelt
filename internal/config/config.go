// Package config contains the schema of Seatbetl's config.yml file, as well
// as the logic to read from it.
package config

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"text/template"

	"github.com/go-seatbelt/seatbelt/internal/trace"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultDatabaseName is the default name of the Postgres database to
	// connnect to.
	DefaultDatabaseName = "seatbelt_development"

	// DefaultHTTPAddr is the default port to start the web server on.
	DefaultHTTPAddr = ":3000"
)

var (
	// RootPath is the absolute path to the root of this project.
	RootPath string

	// Database is the name of the Postgres database the application will
	// connect to.
	Database string

	// Username is the Postgres username of the user who is connecting to the
	// database.
	Username string

	// Password is the password used to authenticate with Postgres when
	// connectimng to the database.
	Password string

	// HTTPAddr is the port to start the web server on.
	HTTPAddr string

	// IsTest is true when the application is started with `go test`.
	IsTest bool
)

type config struct {
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	HTTPAddr string `yaml:"http_addr"`
}

type configContainer struct {
	Development *config `yaml:"development"`
	Staging     *config `yaml:"staging"`
	Test        *config `yaml:"test"`
	Production  *config `yaml:"production"`
}

func init() {
	// Check to see if we're running within a `go test` invocation.
	if flag.Lookup("test.v") != nil {
		IsTest = true
		logrus.Infoln("Application started with go test")
	}

	// Assume the user's root path is the directory that the execuatble is
	// running in.
	bin, err := os.Executable()
	if err != nil {
		panic(err)
	}
	RootPath = filepath.Dir(bin)
	logrus.Infoln("Root path is", RootPath)

	cfg := readConfig()

	// Set the globals based on the values read from the configuration file.
	if cfg.Database == "" {
		logrus.Warnf(`%s: Value for "database" is unset in config/application.yml. Falling back to default %s`, trace.Getfl(), DefaultDatabaseName)
		Database = DefaultDatabaseName
	} else {
		Database = cfg.Database
	}

	if cfg.Username == "" {
		// If the username is unset, try to infer from the shell.
		username, err := user.Current()
		if err != nil {
			logrus.Fatalf(`%s: Value for "username" is unset in config/application.yml, and the name cannot be inferred from the user's shell due to the error: %+v`, trace.Getfl(), err)
		}

		logrus.Warnf(`%s: Value for "username" is unset in config/application.yml. Falling back to the username inferred from the shell: %s`, trace.Getfl(), username.Username)
		Username = username.Username
	} else {
		Username = cfg.Username
	}

	Password = cfg.Password

	if cfg.HTTPAddr == "" {
		logrus.Warnf(`%s: Value for "http_addr is unset in config/application.yml. Falling back to default %s`, trace.Getfl(), DefaultHTTPAddr)
		HTTPAddr = DefaultHTTPAddr
	} else {
		HTTPAddr = cfg.HTTPAddr
	}
}

// readConfig reads and preporcesses the config file, returning the config for
// the currently selected environment.
func readConfig() *config {
	configFilePath := filepath.Join(RootPath, "config", "application.yml")
	f, err := os.OpenFile(configFilePath, os.O_RDONLY, 0664)
	if err != nil {
		logrus.Fatalf("%s: Failed to open config/application.yml: %+v", trace.Getfl(), err)
	}
	defer f.Close()

	cfgContainer := &configContainer{}
	if err := yaml.NewDecoder(preprocess(f)).Decode(cfgContainer); err != nil {
		logrus.Fatalf("%s: Failed to read config/application.yml: %+v", trace.Getfl(), err)
	}

	if IsTest {
		return cfgContainer.Test
	}
	return cfgContainer.Development
}

// preprocess is a helper used for preprocessing the application.yml config
// file. Because this file is allowed contain Go template blocks, we first
// evaluate those blocks, and then return the valid yaml file.
func preprocess(r io.Reader) io.Reader {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		logrus.Fatalf("%s: failed to read from reader: %+v", trace.Getfl(), err)
	}

	tpl, err := template.New("config").Funcs(template.FuncMap{
		"env": func(key string) string {
			return os.Getenv(key)
		},
	}).Parse(string(b))
	if err != nil {
		logrus.Fatalf("%s: failed to parse template: %+v", trace.Getfl(), err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, nil); err != nil {
		logrus.Fatalf("%s: failed to execute template: %+v", trace.Getfl(), err)
	}

	return buf
}
