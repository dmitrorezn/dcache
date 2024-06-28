package main

import (
	"context"
	"github.com/dmitrorezn/feedero/domain"
	"github.com/dmitrorezn/feedero/pkg/server"
	"github.com/google/uuid"
	"github.com/labstack/echo"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
	"os/signal"
)

type Cfg struct {
	HTTPAddr string
	GRPCAddr string
}
type App struct {
	cfg          Cfg
	httpSrv      *server.HTTPServer
	grpcSrv      *server.GRPCServer
	feedbacksSvc FeedbacksService
}

func NewApp(cfg Cfg) *App {
	return &App{
		cfg: cfg,
	}
}

func (a *App) SetupServer() {
	a.httpSrv = server.NewHTTP(a.cfg.HTTPAddr)
	a.grpcSrv = server.NewGRPC(a.cfg.GRPCAddr)
}
func (a *App) SetupServices() {
	a.feedbacksSvc = FeedbacksService{}
}

type FeedbacksService struct {
	s Storage
}
type GetFeedbacksQuery struct {
}

type FeedbacksResult struct {
	Feedbacks []domain.Feedback
}

func (s *FeedbacksService) List(ctx context.Context, query GetFeedbacksQuery) (FeedbacksResult, error) {
	var feedbacks = make([]domain.Feedback, 0)

	feedbacks = append(feedbacks, domain.NewFeedback(
		domain.NewID(),
		true,
		"Nice product"),
	)

	return FeedbacksResult{
		Feedbacks: feedbacks,
	}, nil
}

type CreateFeedback struct {
	Anonymous bool     `json:"anonymous" `
	Content   string   `json:"content"`
	Media     []string `json:"media"`
	Links     []string `json:"links"`
}

type Storage interface {
	InsertFeedback(ctx context.Context, f domain.Feedback) error
}

func (s *FeedbacksService) Create(ctx context.Context, command CreateFeedback) ([]interface{}, error) {
	uid := domain.NewID()

	f := domain.NewFeedback(uid, command.Anonymous, command.Content)

	if err := s.s.InsertFeedback(ctx, f); err != nil {
		return nil, err
	}

	return []interface{}{}, nil
}

type GetFeedbackQuery struct {
	ID domain.ID
}

func (g GetFeedbackQuery) WithParam(name, val string) (err error) {
	g.ID, err = uuid.Parse(val)

	return err
}

type FeedbackResult struct {
	Feedback domain.Feedback
}

func (s *FeedbacksService) Get(ctx context.Context, query GetFeedbacksQuery) (FeedbacksResult, error) {
	var feedbacks = make([]domain.Feedback, 0)

	feedbacks = append(feedbacks, domain.NewFeedback(
		domain.NewID(),
		true,
		"Nice product"),
	)

	return FeedbacksResult{
		Feedbacks: feedbacks,
	}, nil
}

func (a *App) RegisterHandlers() {
	e := echo.New()
	defer a.httpSrv.Register(e)

	v1 := e.Group("/api/v1")

	feedbacks := v1.Group("/feedbacks")

	feedbacks.GET("/list", NewQueryHandler(a.feedbacksSvc.List).GET())
	feedbacks.GET("/:id", NewQueryHandler(a.feedbacksSvc.Get).GET("id"))
	feedbacks.GET("/:id", NewCommandHandler(a.feedbacksSvc.Get).GET("id"))
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()
	var cfg = Cfg{
		HTTPAddr: ":8080",
		GRPCAddr: ":8081",
	}
	var app = NewApp(cfg)

	var setup = []func(){
		app.SetupServer,
		app.SetupServices,
		app.RegisterHandlers,
	}
	for _, s := range setup {
		s()
	}

	wg := errgroup.Group{}

	wg.Go(app.httpSrv.Run)
	wg.Go(func() error {
		<-ctx.Done()
		return app.httpSrv.Close(context.Background())
	})

	if err := wg.Wait(); err != nil {
		log.Println("Wait", err)
	}
}
