package main

import (
	"embed"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/42LoCo42/pinlist/jade"
	"github.com/go-faster/errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/sqlite"
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

	authKey := strings.TrimSpace(os.Getenv("PIN_AUTH"))
	if authKey == "" {
		log.Fatal("No PIN_AUTH set!")
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

		// yes i know this is cringe, but pinlist doesn't really need to be super secure
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				auth, err := c.Cookie("auth")
				if err != nil {
					c.SetCookie(&http.Cookie{
						Name:     "auth",
						Value:    "0",
						Path:     "/",
						MaxAge:   math.MaxInt32,
						Secure:   true,
						HttpOnly: true,
						SameSite: http.SameSiteStrictMode,
					})

					return echo.NewHTTPError(
						http.StatusForbidden,
						errors.Wrap(err, "could not get auth cookie"),
					)
				}

				if auth.Value != authKey {
					return echo.NewHTTPError(
						http.StatusForbidden,
						"invalid auth cookie",
					)
				}

				return next(c)
			}
		},
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

		jade.Jade_index(items, c.Response())
		return nil
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

	if err := e.Start(":8000"); err != nil {
		log.Fatal(err)
	}
}
