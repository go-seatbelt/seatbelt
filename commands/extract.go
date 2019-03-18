package commands

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v2"
)

// ExtractCommand contains the CLI UI that provides feedback to the user.
type ExtractCommand struct {
	// ui is used to provide feedback to the user.
	ui *cli.BasicUi
}

// Help prints the help text of this command.
func (c *ExtractCommand) Help() string {
	return `Extract extracts translateable strings from HTML files. Translateable strings
must be enclosed in HTML tags. For example,

  <p>{{ t "greetings.hello" }}</p>

will find the key "greetings.hello", whereas,

  {{ t "greetings.hello" }}

will find nothing.`
}

// Run executes this command.
func (c *ExtractCommand) Run(args []string) int {
	wd, err := os.Getwd()
	if err != nil {
		c.ui.Error("Failed to get working directory: " + err.Error())
		return 1
	}

	// Parse the config file, if it exists.
	translationFile := filepath.Join(wd, "config", "locales", "active.en.yml")
	f, err := os.OpenFile(translationFile, os.O_RDWR, 0644)
	if err != nil {
		c.ui.Error("There is no translation file at " + translationFile + ". Please create it, and run the command again.")
		return 1
	}
	defer f.Close()

	translationMap := make(map[string]interface{})
	if err := yaml.NewDecoder(f).Decode(&translationMap); err != nil {
		c.ui.Error("Failed to decode yaml file at " + translationFile + " with error " + err.Error())
		return 1
	}

	// Extract all the translation keys from the current working directory.
	translationKeys := make([]string, 0)

	viewsPath := filepath.Join(wd, "views")
	if _, err := os.Stat(viewsPath); err != nil {
		if os.IsNotExist(err) {
			c.ui.Error(`The directory "views" doesn't exist at ` + viewsPath + `. Please create it, and run the command again.`)
		}
	}

	filepath.Walk(viewsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info == nil || info.IsDir() {
			return nil
		}

		ext := ""
		if strings.Index(path, ".") != -1 {
			ext = filepath.Ext(path)
		}

		if ext == ".html" {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			tokenizer := html.NewTokenizer(f)

			for {
				tokenType := tokenizer.Next()

				switch tokenType {
				case html.ErrorToken:
					return nil

				case html.TextToken:
					token := strings.TrimSpace(string(tokenizer.Text()))
					if token != "" {
						if translationKey := extractTranslationKey(token); translationKey != "" {
							translationKeys = append(translationKeys, translationKey)
						}
					}
				}
			}
		}
		return nil
	})

	// Walk through each key in the translation file, checking if a given key
	// exists. If it does not, add it to a new map so that we can append new
	// translation keys to the file.
	translationMapAdditions := make(map[string]interface{})
	for _, key := range translationKeys {
		if _, ok := translationMap[key]; !ok {
			translationMapAdditions[key] = map[string]string{
				"other": "MISSING",
			}
		}
	}

	// Write out the updated map file, if we've added anything to it.
	if len(translationMapAdditions) > 0 {
		if err := yaml.NewEncoder(f).Encode(&translationMapAdditions); err != nil {
			c.ui.Error("Failed to encode updated yaml file at " + translationFile + " with error " + err.Error())
			return 1
		}
		c.ui.Info("Updated translation file with " + strconv.Itoa(len(translationMapAdditions)) + " addition(s).")
	}

	return 0
}

// Synopsis prints the synopsis of this command.
func (c *ExtractCommand) Synopsis() string {
	return "Extract translateable strings from HTML files."
}

const (
	tokenLeftDelim         = "{{"
	tokenRightDelim        = "}}"
	tokenTranslateFunction = "t"
)

// extractTranslationKey extracts a translation key from Go HTML template
// source files.
func extractTranslationKey(text string) string {
	// Remove the leading `{{`.
	if !strings.HasPrefix(text, tokenLeftDelim) {
		return ""
	}
	text = strings.TrimPrefix(text, tokenLeftDelim)

	// Remove the traling `}}`.
	if !strings.HasSuffix(text, tokenRightDelim) {
		return ""
	}
	text = strings.TrimSuffix(text, tokenRightDelim)

	// Remove any leading or trailing whitespace.
	text = strings.TrimSpace(text)

	// Remove the leading function token, but only if the token is an exact
	// match. To check this, we'll make sure the token immediately after the
	// function token is either a space, or a double quote.
	if !strings.HasPrefix(text, tokenTranslateFunction) {
		return ""
	}
	text = strings.TrimPrefix(text, tokenTranslateFunction)

	// Check that the next token from the left is either a space or a
	// doublequote, to make sure that the token is in face 't' instead of the
	// start of the word "template".
	if !strings.HasPrefix(text, " ") {
		if !strings.HasPrefix(text, `"`) {
			return ""
		}
	}

	// remove any leading or trailing whitespace
	text = strings.TrimSpace(text)

	// read each char from the leading `"` to the final `"`
	stateReading := false
	translationKey := ""

	for _, c := range text {
		if stateReading {
			if c == '"' {
				stateReading = false
				return translationKey
			}

			translationKey += string(c)
		}

		if c == '"' {
			stateReading = true
		}
	}

	return translationKey
}
