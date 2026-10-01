# unofficial

northのWebクライアントが使用する非公開APIのGoクライアントです。

> [!WARNING]
>
> - 非公開APIは予告なく変更または削除される可能性があります。
> - セッションCookieはアカウントを操作できる認証情報です。ログへの出力や共有、リポジトリへの保存をしないでください。

トレンドなど認証が不要な経路にはCookieなしで接続できます。

```go
client, err := unofficial.NewPublicClient()
if err != nil {
	log.Fatal(err)
}
trends, _, err := client.Trends(ctx)
if err != nil {
	log.Fatal(err)
}
```

## Cookieを使う

Cookieにはブラウザが送信する`Cookie`ヘッダーの値を渡します。

```go
client, err := unofficial.NewClient(os.Getenv("NORTH_SESSION_COOKIE"))
if err != nil {
	log.Fatal(err)
}

notifications, _, err := client.Notifications(
	ctx,
	unofficial.NotificationsAll,
	"",
)
```

### ブラウザから読み取る

[kooky](https://github.com/browserutils/kooky)を使うと、ブラウザの既定プロファイルからCookieを取得できます。

以下はFirefoxの例です。

```go
package main

import (
	"context"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/Hayao0819/go-north/unofficial"
	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/firefox"
)

func main() {
	ctx := context.Background()
	cookies, err := kooky.ReadCookies(
		ctx,
		kooky.Valid,
		kooky.FilterFunc(func(cookie *kooky.Cookie) bool {
			return cookie != nil && strings.EqualFold(
				strings.TrimPrefix(cookie.Domain, "."),
				"north.rip",
			)
		}),
		kooky.FilterFunc(func(cookie *kooky.Cookie) bool {
			return cookie.Browser != nil && cookie.Browser.IsDefaultProfile()
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	origin, err := url.Parse("https://north.rip/api/")
	if err != nil {
		log.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, cookie := range cookies {
		copy := cookie.Cookie
		jar.SetCookies(origin, []*http.Cookie{&copy})
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		origin.String(),
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, cookie := range jar.Cookies(origin) {
		request.AddCookie(cookie)
	}

	client, err := unofficial.NewClient(request.Header.Get("Cookie"))
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = client.Me(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
```

## テスト

`NORTH_SESSION_COOKIE`にCookieヘッダーの値を設定した上で以下を実行してください。

```bash
go test -tags=integration ./unofficial -run '^TestIntegration'
```
