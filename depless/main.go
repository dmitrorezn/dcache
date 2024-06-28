package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/caarlos0/env/v9"
	"github.com/dmitrorezn/depless/pkg/renderer"
	"github.com/dmitrorezn/depless/repository"
	"github.com/dmitrorezn/depless/repository/storage"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type App struct {
	db         *repository.MongoStorage
	serverHTTP *http.Server
}

type Server struct {
}

func init() {
	if _, err := os.Lstat(".env"); err == nil {
		if err = godotenv.Load(".env"); err != nil {
			log.Fatal(err)
		}
	}
}

type Cfg struct {
	MongoHost   string `env:"MONGO_HOST"`
	MongoUser   string `env:"MONGO_USER"`
	MongoPass   string `env:"MONGO_PASS"`
	MongoDBName string `env:"MONGO_DB_APP_NAME"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	var cfg Cfg
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	app := &App{
		db: repository.New(ctx, repository.Cfg{
			Host: cfg.MongoHost,
			User: cfg.MongoUser,
			Pass: cfg.MongoPass,
			Name: cfg.MongoDBName,
		}),
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		app.runHTTP()
	}()
	go func() {
		defer wg.Done()

		app.runHTTP()
	}()

}
func (app *App) stopHTTP() {
	app.serverHTTP.Shutdown(context.Background())
}
func newRouter() *echo.Echo {
	router := echo.New()
	router.Debug = true
	router.HTTPErrorHandler = HTTPErrorHandler
	router.Renderer = renderer.NewRendererFS(view)

	return router
}
func (app *App) runHTTP() {
	router := newRouter()

	router.GET("/index", indexHandler())
	router.POST("/app/deploy/create", createDeploymentHandler(
		storage.New(app.db.Database),
	))
	router.PUT("/app/deploy", deployAppHandler(
		storage.New(app.db.Database),
	))
	router.POST("/app/create", createApp(
		storage.New(app.db.Database),
	))
	router.GET("/live/:appID", runApp(
		storage.New(app.db.Database),
	))

	app.serverHTTP = &http.Server{
		Addr:    ":10001",
		Handler: router,
	}
}

type Storage interface {
	CreateApp(ctx context.Context, app *storage.App) error
	GetAppByID(ctx context.Context, id uuid.UUID) (storage.App, error)
	CreateDeployment(ctx context.Context, deployment *storage.Deployment) error
	GetDeploymentByID(ctx context.Context, id uuid.UUID) (storage.Deployment, error)
	UpdateApp(ctx context.Context, id uuid.UUID, upd func(app *storage.App) error) error
	UpdateDeployment(ctx context.Context, id uuid.UUID, upd func(deployment *storage.Deployment) error) error
}

//go:embed view
var view embed.FS

func indexHandler() echo.HandlerFunc {
	return func(c echo.Context) error {

		return c.HTML(http.StatusOK, "index")
	}
}
func runApp(store Storage) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		appID, err := uuid.Parse(c.Param("appID"))
		if err != nil {
			return err
		}
		app, err := store.GetAppByID(ctx, appID)
		if err != nil {
			return err
		}
		log.Println("app", app)

		deployment, err := store.GetDeploymentByID(ctx, app.CurrentDeployID)
		if err != nil {
			return err
		}

		log.Println("deployment", deployment)

		return c.JSON(200, nil)
	}
}

type Runtime struct {
	compiled         *cache[uuid.UUID, wazero.CompiledModule]
	runtime          wazero.Runtime
	closer           api.Closer
	compilationCache wazero.CompilationCache
}

func NewRuntime(ctx context.Context, compilationCache wazero.CompilationCache) *Runtime {
	runtime := wazero.NewRuntimeWithConfig(
		ctx,
		wazero.NewRuntimeConfig().
			WithCompilationCache(compilationCache),
	)
	closer, err := wasi_snapshot_preview1.Instantiate(ctx, runtime)
	if err != nil {
		return nil
	}
	return &Runtime{
		closer:           closer,
		runtime:          runtime,
		compilationCache: wazero.NewCompilationCache(),
	}
}
func (r *Runtime) Close(ctx context.Context) error {
	return r.closer.Close(ctx)
}

type Request struct {
	Method string
	Body   []byte
}
type WasmServer struct {
	runtime *Runtime
	storage endpointsStorage
}

func NewWasmServer(storage endpointsStorage, runtime *Runtime) *WasmServer {
	return &WasmServer{
		runtime: runtime,
		storage: storage,
	}
}

func (ws *WasmServer) HandlerRequest(ctx context.Context, id uuid.UUID, r Request) (io.Reader, error) {
	endpoint, err := ws.storage.GetEndpoint(ctx, id)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	stdin := bytes.NewReader(b)

	resp, err := ws.runtime.Invoke(ctx, id, stdin, endpoint)

	return resp, err
}

type Endpoint struct {
	ID   uuid.UUID
	Blob []byte
	Env  map[string]string
}

type endpointsStorage interface {
	GetEndpoint(ctx context.Context, id uuid.UUID) (Endpoint, error)
	SetEndpoint(ctx context.Context, id uuid.UUID) (Endpoint, error)
}

func NewCache[K comparable, V any]() *cache[K, V] {
	return new(cache[K, V])
}

type cache[K comparable, V any] struct {
	m sync.Map
}

var ErrNotFound = errors.New("not found")
var ErrAssert = errors.New("err assert")

func (c *cache[K, V]) Get(key K) (v V, err error) {
	val, ok := c.m.Load(key)
	if !ok {
		return v, ErrNotFound
	}
	if v, ok = val.(V); !ok {
		return v, ErrAssert
	}

	return v, nil
}
func (c *cache[K, V]) Set(key K, v V) error {
	c.m.Store(key, v)

	return nil
}

func (r *Runtime) Invoke(ctx context.Context, id uuid.UUID, stdin io.Reader, endpoint Endpoint) (respReader io.Reader, err error) {
	buf := bytes.NewBuffer(nil)

	mod, err := r.getMod(ctx, id, endpoint)
	if err != nil {
		return nil, err
	}

	modConf := wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(buf)

	for k, v := range endpoint.Env {
		modConf.WithEnv(k, v)
	}
	module, err := r.runtime.InstantiateModule(ctx, mod, modConf)
	if err != nil {
		return nil, err
	}
	_ = module

	return buf, nil

}

func (r *Runtime) getMod(ctx context.Context, id uuid.UUID, endpoint Endpoint) (mod wazero.CompiledModule, err error) {
	if mod, err = r.compiled.Get(id); err == nil {
		return mod, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return mod, err
	}
	if mod, err = r.runtime.CompileModule(ctx, endpoint.Blob); err != nil {
		return nil, err
	}
	if err = r.compiled.Set(id, mod); err != nil {
		return nil, err
	}

	return mod, nil
}

type DeployRequest struct {
	AppID        string `json:"app_id"`
	DeploymentID string `json:"deployment_id"`
}

func deployAppHandler(store Storage) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		ctx := c.Request().Context()
		var request DeployRequest
		if err = c.Bind(&request); err != nil {
			return err
		}
		var (
			deploymentID uuid.UUID
			appID        uuid.UUID
		)
		if deploymentID, err = uuid.Parse(request.DeploymentID); err != nil {
			return err
		}
		if appID, err = uuid.Parse(request.AppID); err != nil {
			return err
		}
		dep, err := store.GetDeploymentByID(ctx, deploymentID)
		if err != nil {
			return err
		}
		var updatedApp *storage.App
		if err = store.UpdateApp(ctx, appID, func(app *storage.App) error {
			app.Deploy(&dep)
			app.SetURL(fmt.Sprintf("https://localhost:10001/live/%s", dep.ID))
			updatedApp = app
			return nil
		}); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, updatedApp)
	}
}

type CreateDeployment struct {
	AppID       string `bson:"app_id"`
	Version     string `bson:"version"`
	BuildBytes  []byte `bson:"build_bytes"`
	ArchivePath string `bson:"archive_path"`
}

func createDeploymentHandler(store Storage) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		ctx := c.Request().Context()
		var request CreateDeployment
		if err = c.Bind(&request); err != nil {
			return err
		}
		appID, err := uuid.Parse(request.AppID)
		if err != nil {
			return err
		}
		if _, err = store.GetAppByID(ctx, appID); err != nil {
			return err
		}
		dep := storage.NewDeployment(
			appID,
			request.Version,
			request.BuildBytes,
			request.ArchivePath,
			"",
		)
		if err = store.CreateDeployment(ctx, dep); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, dep)
	}
}

type CreateApp struct {
	Name string
}

func createApp(store Storage) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		ctx := c.Request().Context()
		var request CreateApp
		if err = c.Bind(&request); err != nil {
			return err
		}
		var user = storage.NewUser("bob")
		app := storage.NewApp(user, request.Name)

		if err := store.CreateApp(ctx, app); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, app)
	}
}
