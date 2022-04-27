package main

import (
	"errors"
	"log"

	"github.com/go-seatbelt/seatbelt"
)

func main() {
	app := seatbelt.New(seatbelt.Option{
		TemplateDir: "templates",
		Reload:      true,
	})

	app.Get("/", func(c *seatbelt.Context) error {
		return c.Render("index", nil)
	})
	app.Get("/session", func(c *seatbelt.Context) error {
		return c.Render("session", map[string]interface{}{
			"Session": c.Get("session"),
		})
	})
	app.Post("/session", func(c *seatbelt.Context) error {
		v := &struct {
			Session string
			Flash   string
			Error   string
		}{}
		if err := c.Params(v); err != nil {
			return err
		}

		if v.Error != "" {
			return errors.New("main: " + v.Error)
		}

		if v.Session != "" {
			c.Set("session", v.Session)
		}
		if v.Flash != "" {
			c.Flash("notice", v.Flash)
		}

		return c.Redirect("/session")
	})

	app.Post("/session/reset", func(c *seatbelt.Context) error {
		c.Reset()
		return c.Redirect("/session")
	})

	log.Fatalln(app.Start(":3000"))
}
