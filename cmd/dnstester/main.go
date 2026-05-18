package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	ossvc "github.com/kardianos/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tomerklein/dnstester/internal/config"
	"github.com/tomerklein/dnstester/internal/handler"
	intmetrics "github.com/tomerklein/dnstester/internal/metrics"
	httpsrv "github.com/tomerklein/dnstester/internal/server"
	intsvc "github.com/tomerklein/dnstester/internal/service"
	"github.com/tomerklein/dnstester/internal/store"
	webembed "github.com/tomerklein/dnstester/web"
)

var (
	version   = "dev"
	buildMode = "dev"
)

type program struct {
	port      int
	db        *sql.DB
	web       *httpsrv.Server
	scheduler *intsvc.SchedulerService
}

func (p *program) Start(_ ossvc.Service) error {
	go func() {
		if err := p.run(); err != nil {
			log.Printf("server exited: %v", err)
		}
	}()
	return nil
}

func (p *program) Stop(_ ossvc.Service) error {
	if p.scheduler != nil {
		p.scheduler.Stop()
	}
	err := p.web.Shutdown(10 * time.Second)
	if p.db != nil {
		p.db.Close()
	}
	return err
}

func (p *program) run() error {
	_ = version
	_ = buildMode

	configDir := configDirectory()

	db, err := store.Open(filepath.Join(configDir, "dnstester.db"))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	p.db = db

	runs := store.NewRunStore(db)
	cfgSvc := config.NewService(configDir)
	dnsSvc := intsvc.NewDNSService()
	pingSvc := intsvc.NewPingService()
	testSvc := intsvc.NewTestService(dnsSvc, pingSvc)

	p.scheduler = intsvc.NewSchedulerService(cfgSvc, testSvc, runs)
	p.scheduler.Start()

	prometheus.MustRegister(intmetrics.NewCollector(testSvc, runs))

	cfgHandler := handler.NewConfigHandler(cfgSvc)
	testHandler := handler.NewTestHandler(cfgSvc, testSvc, runs)
	historyHandler := handler.NewHistoryHandler(runs)
	scheduleHandler := handler.NewScheduleHandler(cfgSvc)

	ui, err := fs.Sub(webembed.Assets, "dist")
	if err != nil {
		return fmt.Errorf("ui assets: %w", err)
	}

	p.web = httpsrv.New(p.port, cfgHandler, testHandler, historyHandler, scheduleHandler, ui)
	return p.web.Run()
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "service" {
		handleServiceCmd()
		return
	}

	port := flag.Int("port", 7020, "port to listen on")
	flag.Parse()

	prg := &program{port: *port}
	s, err := ossvc.New(prg, svcConfig(*port))
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

func configDirectory() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	dir = filepath.Join(dir, "dnstester")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("create config dir: %v", err)
	}
	return dir
}
