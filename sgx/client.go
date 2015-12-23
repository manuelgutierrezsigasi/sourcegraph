package sgx

import (
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/url"
	"os"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"sourcegraph.com/sourcegraph/grpccache"
	"src.sourcegraph.com/sourcegraph/auth/userauth"
	"src.sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"src.sourcegraph.com/sourcegraph/sgx/cli"
	"src.sourcegraph.com/sourcegraph/util/randstring"
)

var ClientContextFuncs []func(context.Context) context.Context

func init() {
	cli.CLI.AddGroup("Client API endpoint", "", &Endpoint)
	cli.CLI.AddGroup("Client authentication", "", &Credentials)
}

var Endpoint EndpointOpts

// EndpointOpts sets the URL to use when contacting the Sourcegraph
// server.
//
// This endpoint may differ from the endpoint that a server wishes to
// advertise externally. For example, if internal API traffic is
// routed through a local network, you may want to use
// "http://10.1.2.3:3080" as the endpoint here, but you may want
// external clients to use "https://example.com:443". To do so, run
// the server with `src --endpoint http://10.1.2.3:3080 serve
// --app-url https://example.com`.
type EndpointOpts struct {
	// URL is the raw endpoint URL. Callers should almost never use
	// this; use the URLOrDefault method instead.
	URL string `short:"u" long:"endpoint" description:"URL to Sourcegraph server" default:"" env:"SRC_URL"`
}

// NewContext sets the server endpoint in the context.
func (c *EndpointOpts) NewContext(ctx context.Context) context.Context {
	return sourcegraph.WithGRPCEndpoint(ctx, c.URLOrDefault())
}

// URLOrDefault returns c.Endpoint as a *url.URL but with various
// modifications (e.g. a sensible default, no path component, etc). It
// is also responsible for erroring out when the user provides a
// garbage endpoint URL. Always use c.URLOrDefault instead of
// c.Endpoint, even when you just need a string form (just call
// URLOrDefault().String()).
func (c *EndpointOpts) URLOrDefault() *url.URL {
	e := c.URL
	if e == "" {
		// The user did not explicitly specify a endpoint URL, so use the default
		// found in the auth file.
		userAuth, err := userauth.Read(Credentials.AuthFile)
		if err != nil {
			log.Fatal(err, "failed to read user auth file (in EndpointOpts.URLOrDefault)")
		}
		e, _ = userAuth.GetDefault()
		if e == "" {
			// Auth file has no default, so just choose a sensible default value
			// instead.
			e = "http://localhost:3080"
		}
	}
	endpoint, err := url.Parse(e)
	if err != nil {
		log.Fatal(err, "invalid endpoint URL specified (in EndpointOpts.URLOrDefault)")
	}

	// This prevents users who might be using e.g. Sourcegraph under a reverse proxy
	// at mycompany.com/sourcegraph (a subdirectory) from logging in -- but this
	// is not a typical case and otherwise users who effectively run:
	//
	//  src --endpoint=http://localhost:3080 login
	//
	// will be unable to authenticate in the event that they add a slash suffix:
	//
	//  src --endpoint=http://localhost:3080/ login
	//
	endpoint.Path = ""

	if endpoint.Scheme == "" {
		log.Fatal("invalid endpoint URL specified, endpoint URL must start with a schema (e.g. http://mydomain.com)")
	}
	return endpoint
}

// CredentialOpts sets the authentication credentials to use when
// contacting the Sourcegraph server's API.
type CredentialOpts struct {
	AuthFile string `long:"auth-file" description:"path to .src-auth" default:"$HOME/.src-auth" env:"SRC_AUTH_FILE"`

	mu sync.RWMutex

	// AccessToken should be accessed via the GetAccessToken and SetAccessToken
	// methods which synchronize access to this value.
	AccessToken string `long:"token" description:"access token (OAuth2)" env:"SRC_TOKEN"`
}

var Credentials CredentialOpts

// WithCredentials sets the HTTP and gRPC credentials in the context.
func (c *CredentialOpts) WithCredentials(ctx context.Context) (context.Context, error) {
	token := c.GetAccessToken()
	if token == "" && c.AuthFile != "" { // AccessToken takes precedence over AuthFile
		userAuth, err := userauth.Read(c.AuthFile)
		if err != nil {
			return nil, err
		}

		ua := userAuth[Endpoint.URLOrDefault().String()]
		if ua != nil {
			c.SetAccessToken(ua.AccessToken)
		}
	}

	token = c.GetAccessToken()
	if token != "" {
		ctx = sourcegraph.WithCredentials(ctx, oauth2.StaticTokenSource(&oauth2.Token{TokenType: "Bearer", AccessToken: token}))
	}

	return ctx, nil
}

// GetAccessToken returns the currently set AccessToken.
//
// It synchronizes access to the token by acquiring a read lock.
func (c *CredentialOpts) GetAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.AccessToken
}

// SetAccessToken sets a new access token.
//
// It synchronizes access to the token by acquiring a write lock.
func (c *CredentialOpts) SetAccessToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.AccessToken = token
}

func init() {
	sourcegraph.Cache = &grpccache.Cache{
		MaxSize: 150 * (1024 * 1024), // 150 MB
		KeyPart: func(ctx context.Context) string {
			// Separate caches based on the authorization level, to
			// avoid cross-user/client leakage.
			//
			// The authorization metadata can be in EITHER:
			//  (a) the client credentials, if the client was created
			//      normally. This is tried first.
			//  (b) the ctx's metadata, if the client was created by the
			//      remote services in a federated request. This is tried
			//      second in order to properly cache federated requests.
			//
			// NOTE: This is important code. If we don't get
			// the right auth data, we could leak sensitive data
			// across authentication boundaries.

			key := sourcegraph.GRPCEndpoint(ctx).String() + ":"

			if cred := sourcegraph.CredentialsFromContext(ctx); cred != nil {
				md, err := (oauth.TokenSource{TokenSource: cred}).GetRequestMetadata(ctx)
				if err != nil {
					log.Printf("Determining cache key with token auth failed: %s. Caching will not be performed.", err)
					// Use a random string to prevent caching.
					return "err#" + randstring.NewLen(64)
				}
				key += md["authorization"]
			} else if md, ok := metadata.FromContext(ctx); ok {
				// if ctx metadata contains auth token, use it in the cache key
				if authMD, ok := md["authorization"]; ok && len(authMD) > 0 {
					key += authMD[len(authMD)-1]
				}
			}

			s := sha256.Sum256([]byte(key))
			return base64.StdEncoding.EncodeToString(s[:])
		},
		Log: os.Getenv("CACHE_LOG") != "",
	}
}

// Client returns a Sourcegraph API client configured to use the
// specified endpoints and authentication info.
func Client() *sourcegraph.Client {
	return sourcegraph.NewClientFromContext(cli.Ctx)
}

func init() {
	cli.CLI.InitFuncs = append(cli.CLI.InitFuncs, func() {
		// The "src version" command does not need a cli context at all.
		if cli.CLI.Active != nil && cli.CLI.Active.Name == "version" {
			return
		}
		cli.Ctx = WithClientContext(context.Background())
	})
}

// WithClientContext returns a copy of parent with client endpoint and
// auth information added.
func WithClientContext(parent context.Context) context.Context {
	// The "src serve" command is the only non-client command; it
	// must not have credentials set (because it is not a client
	// command).
	if cli.CLI.Active != nil && cli.CLI.Active.Name == "serve" {
		Credentials.AuthFile = ""
	}
	ctx := WithClientContextUnauthed(parent)
	ctx, err := Credentials.WithCredentials(ctx)
	if err != nil {
		log.Fatalf("Error constructing API client credentials: %s.", err)
	}
	return ctx
}

// WithClientContextUnauthed returns a copy of parent with client endpoint added.
func WithClientContextUnauthed(parent context.Context) context.Context {
	return Endpoint.NewContext(parent)
}
