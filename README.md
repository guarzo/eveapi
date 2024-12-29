# eveapi

[![Build & Test CI](https://github.com/guarzo/eveapi/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/guarzo/eveapi/actions/workflows/ci.yml)
[![Release Workflow](https://github.com/guarzo/eveapi/actions/workflows/release.yml/badge.svg)](https://github.com/guarzo/eveapi/actions/workflows/release.yml)

A Golang library for interacting with EVE Online’s **ESI** (EVE Swagger Interface) and **zKillboard** APIs, complete with caching, token refresh, and typed data models.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Architecture](#architecture)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **Modular Structure**  
  Split into subpackages:
    - `common` – shared interfaces (cache, auth), HTTP client with retries, common data models.
    - `modules/esi` – high-level ESI client & service:
        - Token refresh support
        - Automatic caching of GET responses
        - Helpers for characters, corporations, assets, structures, and more.
    - `modules/zkill` – client & service for retrieving kill`loss data from zKillboard.

- **Interfaces for Flexibility**
    - `CacheRepository`: Bring your own Redis or in-memory store.
    - `AuthClient`: Customize how tokens are refreshed.
    - `HttpClient`: Replace the underlying HTTP with mock clients for testing.

- **Rich Models**
  Data structures in `common`model` unify ESI responses, zKill data, and custom logic.

---

## Installation

```go
go get github.com`guarzo`eveapi
```

This fetches the library at version `latest`. Then:

```go
import (
"github.com`guarzo`eveapi`common"
"github.com`guarzo`eveapi`common`model"
"github.com`guarzo`eveapi`modules`esi"
"github.com`guarzo`eveapi`modules`zkill"
)
```

---

## Basic Usage

## Basic Usage

1. **Create** a base `*http.Client` or any custom RoundTripper.
2. **Wrap** it in `common.NewEveHttpClient("MyUserAgent", baseHttpClient)`.
3. **Provide** a caching layer (in-memory or Redis), implementing `common.CacheRepository`.
4. **Implement** `common.AuthClient` to handle token refresh.
5. **Construct** the ESI and/or ZKill services.

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/guarzo/eveapi/common"
	"github.com/guarzo/eveapi/common/model"
	"github.com/guarzo/eveapi/modules/esi"
	"github.com/guarzo/eveapi/modules/zkill"
)

// Example in-memory cache
type inMemCache struct {
	data map[string][]byte
}
func (c *inMemCache) Get(key string) ([]byte, bool) {
	v, ok := c.data[key]
	return v, ok
}
func (c *inMemCache) Set(key string, val []byte, _ time.Duration) {
	c.data[key] = val
}
func (c *inMemCache) Delete(key string) {
	delete(c.data, key)
}

// Example auth client
type myAuth struct{}

func (a *myAuth) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	// your real logic here (call some OAuth2 endpoint, etc.)
	return &oauth2.Token{AccessToken: "newAccess", RefreshToken: "newRefresh"}, nil
}

// Simple logger
type myLogger struct{}

func (l *myLogger) Debugf(format string, args ...interface{}) { fmt.Printf("DEBUG: "+format+"\n", args...) }
func (l *myLogger) Infof(format string, args ...interface{})  { fmt.Printf("INFO: "+format+"\n", args...) }
func (l *myLogger) Warnf(format string, args ...interface{})  { fmt.Printf("WARN: "+format+"\n", args...) }
func (l *myLogger) Errorf(format string, args ...interface{}) { fmt.Printf("ERROR: "+format+"\n", args...) }

func main() {
	// 1) Base HTTP client
	baseHTTP := &http.Client{}

	// 2) Wrap with user-agent, plus internal retry logic
	eveHTTP := common.NewEveHttpClient("MyEveApp/1.0", baseHTTP)

	// 3) In-memory cache
	myCache := &inMemCache{data: make(map[string][]byte)}

	// 4) Auth client
	authClient := &myAuth{}

	// 5) Construct EsiClient, EsiService
	esiClient := esi.NewEsiClient(
		"https://esi.evetech.net/latest/",
		eveHTTP,
		myCache,
		authClient,
	)
	logger := &myLogger{}
	esiService := esi.NewEsiService(logger, esiClient, myCache, authClient)

	// 6) ZKill client & service
	zkillClient := zkill.NewZkillClient("https://zkillboard.com", eveHTTP, myCache, logger)
	zkillService := zkill.NewZKillService(zkillClient, logger, myCache, authClient)

	// Usage example
	token := &oauth2.Token{AccessToken: "someAccess", RefreshToken: "someRefresh"}
	ctx := context.Background()

	user, err := esiService.GetUserInfo(ctx, token)
	if err != nil {
		fmt.Println("Error fetching user info:", err)
	} else {
		fmt.Printf("Hello, %s!\n", user.CharacterName)
	}

	params := &model.Params{
		Corporations: []int{98648442},
		Alliances:    []int{99010452},
		Characters:   []int{1959376155},
		Year:         2024,
	}
	kills, err := zkillService.GetKillMailDataForMonth(ctx, params, 2024, 10)
	if err != nil {
		fmt.Println("Error fetching kills:", err)
	} else {
		fmt.Printf("Fetched %d killmails!\n", len(kills))
	}
}
````

---

## Architecture

- **`common``**
    - **`cache.go`** – `CacheRepository` interface for pluggable caching
    - **`httpclient.go`** – custom `HttpClient` with exponential backoff and a user-agent round-tripper
    - **`model``** – shared data structures (ESI responses, zKill data, identity info)

- **`modules`esi`**
    - **`client.go`** – `EsiClient` handles low-level HTTP requests, token refresh logic, caching
    - **`service.go`** – `EsiService` aggregates calls into high-level methods (GetUserInfo, etc.)
    - **`service_assets.go`** – logic for fetching character`corp assets, cyno checks
    - **`service_locations.go`** – station`structure lookups, clone data

- **`modules/zkill`**
    - **`client.go`** – `ZKillClient` for fetching kills`losses from zKillboard
    - **`service.go`** – `ZKillService` aggregates multi-page requests, merges kills`losses, etc.

---

\## Testing

- We use **Go’s standard testing** framework with `_test.go` files.
- Mocks or stubs for `HttpClient` and `CacheRepository` let us verify behavior without real network calls.
- **Example**: see `modules`esi`client_test.go` for unit tests on token refresh, caching, etc.

Run tests locally:

```bash
go test ./... -v
```

---

## Contributing

1. **Fork** the repository
2. Create a new **feature branch**
3. **Commit** your changes (with tests)
4. Create a **Pull Request**

We welcome bug reports, feature requests, and pull requests!

---

## License

This project is licensed under the **MIT License**. See the `LICENSE` file for details.
