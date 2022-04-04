package commands

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/mitchellh/cli"
	"github.com/sirupsen/logrus"
)

// InitCommand contains the CLI UI that provides feedback to the user.
type InitCommand struct {
	// ui is used to provide feedback to the user.
	ui *cli.BasicUi
}

// Help prints the help text of this command.
func (c *InitCommand) Help() string {
	return "Initializes a new Seatbelt project."
}

// Run executes this command.
func (c *InitCommand) Run(args []string) int {
	if len(args) == 0 {
		c.ui.Error("Error: You must give your project a name.")
		return 1
	}

	name := args[0]
	paths := []string{
		filepath.Join("assets", "css"),
		filepath.Join("assets", "js", "controllers"),
		"config",
		"controllers",
		"models",
		filepath.Join("public", "css"),
		filepath.Join("public", "js"),
		filepath.Join("public", "images"),
		"server",
		"services",
		"views",
	}

	c.ui.Info(`Buckle up! Seatbelt is initializing a new project in the folder "` + name + `".`)

	for _, path := range paths {
		fullpath := filepath.Join(name, path)

		if err := os.MkdirAll(fullpath, os.FileMode(0777)); err != nil {
			c.ui.Error(`Error: The directory "` + fullpath + `" could not be created: ` + err.Error())
			return 1
		}
	}

	files := []string{
		filepath.Join("assets", "css", "application.scss"),
		filepath.Join("assets", "js", "application.js"),
		filepath.Join("assets", "js", "controllers", ".gitkeep"),
		filepath.Join("config", "application.yml"),
		filepath.Join("controllers", "application_controller.go"),
		filepath.Join("models", ".gitkeep"),
		filepath.Join("public", "css", "style.css"),
		filepath.Join("public", "js", "main.js"),
		filepath.Join("public", "images", ".gitkeep"),
		filepath.Join("server", "routes.go"),
		filepath.Join("server", "server.go"),
		filepath.Join("services", ".gitkeep"),
		filepath.Join("views", "layout.html"),
		filepath.Join("views", "index.html"),
		".babelrc",
		"main.go",
		"package.json",
		"README.md",
		"webpack.config.js",
	}

	for _, f := range files {
		fullpath := filepath.Join(name, f)

		if _, err := os.Create(fullpath); err != nil {
			c.ui.Error(`Error: The file "` + fullpath + `" could not be created: ` + err.Error())
			return 1
		}
	}

	// Populate the files with the horrible defaults :(
	for filename, contents := range templates {
		fullpath := filepath.Join(name, filepath.Clean(filename))

		f, err := os.OpenFile(fullpath, os.O_RDWR, 0777)
		if err != nil {
			logrus.Errorf("Error opening file %s: %+v", fullpath, err)
			continue
		}

		// Parse the template and execute it with the project data.
		t, err := template.New(fullpath).Delims("{%", "%}").Parse(contents)
		if err != nil {
			logrus.Errorf("Error parsing content template %s: %+v", fullpath, err)
			continue
		}

		if err := t.ExecuteTemplate(f, fullpath, map[string]string{
			"Name":       name,
			"ImportPath": "github.com/bentranter/" + name,
		}); err != nil {
			logrus.Errorf("Error writing to file %s: %+v", fullpath, err)
		}

		if err := f.Close(); err != nil {
			logrus.Errorf("Error closing file %s: %+v", fullpath, err)
			continue
		}
	}

	c.ui.Info(`Your application is ready! Type "cd ` + name + `" to see your new project.`)
	return 0
}

// Synopsis prints the synopsis of this command.
func (c *InitCommand) Synopsis() string {
	return "Init creates and initializes a new Seatbelt project in the given directory."
}

var templates = map[string]string{
	"config/application.yml": `default: &default
  http_addr: :3000

development:
  <<: *default
  database: {% .Name %}_development
  username: postgres

test:
  <<: *default
  username: {{ env "POSTGRES_USER" }}
  database: {% .Name %}_test

staging:
  <<: *default
  username: {{ env "POSTGRES_USER" }}
  database: {% .Name %}_staging

production:
  <<: *default
  database: {% .Name %}_production
  username: {% .Name %}
  password: {{ env "{% .Name %}_DATABASE_PASSWORD" }}
`,

	"controllers/application_controller.go": `package controllers

import (
	"net/http"

	"github.com/bentranter/seatbelt/controllers"
)

// Index renders the HTML template at views/index.html.
func Index(w http.ResponseWriter, r *http.Request) {
	controllers.HTML(w, r, "index", nil)
}
`,

	"server/routes.go": `package server

import (
	"{% .ImportPath %}/controllers"
	"github.com/go-chi/chi"
)

func Routes(r *chi.Mux) {
	r.Get("/", controllers.Index)
}
`,

	"server/server.go": `package server

import "github.com/bentranter/seatbelt/server"

func Start() {
	server.Start(Routes)
}
`,

	"views/layout.html": `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta http-equiv="X-UA-Compatible" content="ie=edge">
  <title>{% .Name %}</title>
  <link href="/public/css/style.css" rel="stylesheet"/>
</head>
<body>
  {{ yield }}
  <script src="/public/js/main.js"></script>
</body>
</html>
`,

	"views/index.html": `<h1>{% .Name %}<h1>
`,

	".babelrc": `{
    "presets": ["@babel/preset-env"],
    "plugins": ["@babel/plugin-proposal-class-properties"]
}
`,

	"main.go": `package main

import (
	"{% .ImportPath %}/server"
)

func main() {
	server.Start()
}
`,

	"package.json": `{
  "name": "{% .Name %}",
  "version": "0.0.1",
  "description": "{% .Name %}",
  "main": "assets/js/application.js",
  "scripts": {
    "start": "./node_modules/.bin/webpack --mode=development --watch",
    "build": "./node_modules/.bin/webpack --mode=production",
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "keywords": [
    "{% .Name %}"
  ],
  "author": "%s",
  "license": "ISC",
  "dependencies": {
    "stimulus": "^1.1.1",
	"turbolinks": "^5.2.0"
  },
  "devDependencies": {
    "@babel/core": "^7.2.2",
    "@babel/plugin-proposal-class-properties": "^7.2.3",
    "@babel/preset-env": "^7.2.3",
    "babel-loader": "^8.0.5",
    "webpack": "^4.29.0",
    "webpack-cli": "^3.2.1"
  }
}
`,

	"webpack.config.js": `const path = require("path")

module.exports = {
  entry: path.resolve(__dirname, "assets", "application.js"),
  output: {
	filename: "main.js",
	path: path.resolve(__dirname, "public", "js")
  },
  module: {
	rules: [
	  {
		test: /\.js$/,
		exclude: /node_modules/,
		use: {
		  loader: "babel-loader"
		}
	  }
	]
  }
}
`,
}
