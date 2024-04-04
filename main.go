package main

import (
	"embed"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/42LoCo42/pin/jade"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-faster/errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Entry struct {
	item string
	time time.Time
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
	authKey := strings.TrimSpace(os.Getenv("PIN_AUTH"))
	if authKey == "" {
		log.Fatal("No PIN_AUTH set!")
	}

	db, err := badger.Open(badger.DefaultOptions("db"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not open DB"))
	}
	defer db.Close()

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

		if err := db.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				if err := it.Item().Value(func(val []byte) error {
					var time time.Time
					if err := time.UnmarshalBinary(val); err != nil {
						return errors.Wrap(err, "could not unmarshal timestamp")
					}

					entries = append(entries, Entry{
						item: string(it.Item().Key()),
						time: time,
					})
					return nil
				}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				errors.Wrap(err, "db iteration failure"),
			)
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].time.UnixMilli() < entries[j].time.UnixMilli()
		})

		items := []string{}
		for _, item := range entries {
			items = append(items, item.item)
		}

		jade.Jade_index(items, c.Response())
		return nil
	})

	e.POST("/add", func(c echo.Context) error {
		item, err := getItem(c)
		if err != nil {
			return err
		}

		if err := db.Update(func(txn *badger.Txn) error {
			val, err := time.Now().MarshalBinary()
			if err != nil {
				return errors.Wrap(err, "could not marshal timestamp")
			}

			return txn.Set([]byte(item), val)
		}); err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				errors.Wrap(err, "db transaction failure"),
			)
		}
		return nil
	})

	e.POST("/del", func(c echo.Context) error {
		item, err := getItem(c)
		if err != nil {
			return err
		}

		if err := db.Update(func(txn *badger.Txn) error {
			return txn.Delete([]byte(item))
		}); err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				errors.Wrap(err, "db transaction failure"),
			)
		}

		return nil
	})

	if err := e.Start(":8000"); err != nil {
		log.Fatal(err)
	}
}
