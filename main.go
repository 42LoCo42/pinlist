package main

import (
	"embed"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-faster/errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	g "github.com/maragudk/gomponents"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
	"gorm.io/gorm"
)

type Entry struct {
	ID   uint
	Item string
	Time time.Time
}

//go:embed static
var staticFS embed.FS

func getItem(c echo.Context) (string, error) {
	raw, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, errors.Wrap(err, "could not read body"))
	}

	item := strings.TrimSpace(string(raw))
	if item == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "item must not be empty")
	}

	return item, nil
}

func main() {
	db, err := gorm.Open(sqlite.Open("db/pinlist.db"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not open database"))
	}

	if err := db.AutoMigrate(&Entry{}); err != nil {
		log.Fatal(errors.Wrap(err, "database migration failed"))
	}

	e := echo.New()
	e.Use(
		middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format:           "${time_custom} ${remote_ip} - ${method} ${host}${uri} - ${status} ${error}\n",
			CustomTimeFormat: "2006/01/02 15:04:05",
		}),

		middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "static",
			Filesystem: http.FS(staticFS),
		}),
	)

	e.GET("/", func(c echo.Context) error {
		entries := []Entry{}
		if err := db.Order("time").Find(&entries).Error; err != nil {
			return errors.Wrap(err, "could not list DB")
		}

		items := []string{}
		for _, item := range entries {
			item := html.EscapeString(item.Item)
			if errors.Must(regexp.MatchString("^https?://", item)) {
				item = fmt.Sprintf(`<a href="%v">%v</a>`, item, item)
			}

			items = append(items, item)
		}

		return Page(items).Render(c.Response())
	})

	e.POST("/add", func(c echo.Context) error {
		item, err := getItem(c)
		if err != nil {
			return err
		}

		if err := db.Create(&Entry{
			Item: item,
			Time: time.Now(),
		}).Error; err != nil {
			return errors.Wrap(err, "could not insert entry")
		}
		return nil
	})

	e.POST("/del", func(c echo.Context) error {
		item, err := getItem(c)
		if err != nil {
			return err
		}

		if err := db.Where("item = ?", item).Delete(&Entry{}).Error; err != nil {
			return errors.Wrap(err, "could not delete item")
		}
		return nil
	})

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}

func Page(items []string) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    "Pinlist",
		Language: "en",
		Head: []g.Node{
			Script(Src("/lib.js")),
		},
		Body: []g.Node{
			H1(g.Text("Pinlist")),
			Form(Action("javascript:"),
				Input(Type("submit"), g.Attr("onclick", "add(this)"), Style("display: none")),
				Table(
					g.Map(items, func(item string) g.Node {
						return Tr(
							Td(Input(Type("submit"), Value("🗑️"), g.Attr("onclick", "del(this)"))),
							Td(ID("item"), g.Raw(item)),
						)
					})...,
				),
				Input(Type("text"), ID("newItem"), g.Attr("autofocus", "")),
				Input(Type("submit"), Value("➕"), g.Attr("onclick", "add(this)")),
			),
		},
	})
}
