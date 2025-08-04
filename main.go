package main

import (
	"context"
	"embed"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	// HTTP
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	// HTML
	g "github.com/maragudk/gomponents"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"

	// DB
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	// OIDC
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	// misc
	"github.com/go-faster/errors"
	"github.com/google/uuid"
)

type Entry struct {
	ID   uint
	Item string
	Time time.Time
}

//go:embed static
var staticFS embed.FS

var sessionCookieName string = "__Host-session"

var oidc_issuer = os.Getenv("PINLIST_OIDC_ISSUER")
var oidc_client_id = os.Getenv("PINLIST_OIDC_CLIENT_ID")
var oidc_client_secret = os.Getenv("PINLIST_OIDC_CLIENT_SECRET")

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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	key := securecookie.GenerateRandomKey(32)

	provider, err := rp.NewRelyingPartyOIDC(
		ctx,
		oidc_issuer,
		oidc_client_id,
		oidc_client_secret,
		"",
		[]string{"openid"},
		rp.WithPKCE(httphelper.NewCookieHandler(key, key)),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create OIDC provider")
	}

	db, err := gorm.Open(sqlite.Open(os.Args[1]))
	if err != nil {
		return errors.Wrap(err, "could not open database")
	}

	if err := db.AutoMigrate(&Entry{}); err != nil {
		return errors.Wrap(err, "database migration failed")
	}

	e := echo.New()

	e.Use(
		middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format:           "${time_custom} ${remote_ip} - ${method} ${uri} - ${status} ${error}\n",
			CustomTimeFormat: "2006/01/02 15:04:05",
		}),

		middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "static",
			Filesystem: http.FS(staticFS),
		}),

		session.Middleware(sessions.NewCookieStore(key)),
	)

	authed := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fail := func() error {
				rp.AuthURLHandler(
					func() string {
						return uuid.NewString()
					},
					provider,
				)(c.Response(), c.Request())
				return nil
			}

			sess, err := session.Get(sessionCookieName, c)
			if err != nil {
				return fail()
			}

			login, ok := sess.Values["login"].(bool)
			if !(ok && login) {
				return fail()
			}

			return next(c)
		}
	}

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
	}, authed)

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
	}, authed)

	e.POST("/del", func(c echo.Context) error {
		item, err := getItem(c)
		if err != nil {
			return err
		}

		if err := db.Where("item = ?", item).Delete(&Entry{}).Error; err != nil {
			return errors.Wrap(err, "could not delete item")
		}
		return nil
	}, authed)

	e.GET("/oauth2/callback", func(c echo.Context) error {
		var err error = nil
		rp.CodeExchangeHandler(
			rp.UserinfoCallback(func(
				w http.ResponseWriter,
				r *http.Request,
				tokens *oidc.Tokens[*oidc.IDTokenClaims],
				state string,
				rp rp.RelyingParty,
				info *oidc.UserInfo,
			) {
				err = func() error {
					sess, err := session.Get(sessionCookieName, c)
					if err != nil {
						return errors.Wrap(err, "failed to get session")
					}

					sess.Options = &sessions.Options{
						Path:     "/",
						MaxAge:   86400,
						Secure:   true,
						HttpOnly: true,
						SameSite: http.SameSiteStrictMode,
					}

					sess.Values["login"] = true
					if err := sess.Save(c.Request(), c.Response()); err != nil {
						return errors.Wrap(err, "failed to save session")
					}

					// this is stupid
					return c.HTML(http.StatusOK, "<script>location = '/'</script>")
				}()
			}),
			provider,
		)(c.Response(), c.Request())
		return err
	})

	return e.Start(":8080")
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
