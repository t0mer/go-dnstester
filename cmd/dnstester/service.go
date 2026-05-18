package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	ossvc "github.com/kardianos/service"
)

func svcConfig(port int) *ossvc.Config {
	cfg := &ossvc.Config{
		Name:        "dnstester",
		DisplayName: "DNS Tester",
		Description: "DNS performance testing service — web UI available on the configured port.",
	}
	if port != 7020 {
		cfg.Arguments = []string{fmt.Sprintf("--port=%d", port)}
	}
	return cfg
}

func handleServiceCmd() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: dnstester service <install|uninstall> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags for install:\n")
		fmt.Fprintf(os.Stderr, "  --port int   port to listen on (default 7020)\n")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("service", flag.ExitOnError)
	port := fs.Int("port", 7020, "port to listen on")
	fs.Parse(os.Args[3:]) //nolint:errcheck

	switch os.Args[2] {
	case "install":
		installService(*port)
	case "uninstall":
		uninstallService()
	default:
		fmt.Fprintf(os.Stderr, "unknown service subcommand: %q\n", os.Args[2])
		fmt.Fprintf(os.Stderr, "valid subcommands: install, uninstall\n")
		os.Exit(1)
	}
}

func installService(port int) {
	prg := &program{port: port}
	s, err := ossvc.New(prg, svcConfig(port))
	if err != nil {
		log.Fatalf("service: %v", err)
	}

	if err := s.Install(); err != nil {
		log.Fatalf("install: %v", err)
	}
	fmt.Println("Service installed.")

	if err := s.Start(); err != nil {
		fmt.Printf("Warning: service installed but could not be started: %v\n", err)
		fmt.Println("Start it manually with the commands below.")
	} else {
		fmt.Println("Service started.")
	}

	printManageHints()
}

func uninstallService() {
	prg := &program{}
	s, err := ossvc.New(prg, svcConfig(7020))
	if err != nil {
		log.Fatalf("service: %v", err)
	}

	// Best-effort stop before uninstall; ignore errors (may already be stopped).
	_ = s.Stop()

	if err := s.Uninstall(); err != nil {
		log.Fatalf("uninstall: %v", err)
	}
	fmt.Println("Service uninstalled.")
}

func printManageHints() {
	switch runtime.GOOS {
	case "linux":
		fmt.Println("\nManage with systemctl:")
		fmt.Println("  systemctl status  dnstester")
		fmt.Println("  systemctl stop    dnstester")
		fmt.Println("  systemctl start   dnstester")
		fmt.Println("  systemctl restart dnstester")
		fmt.Println("  journalctl -u dnstester -f")
	case "windows":
		fmt.Println("\nManage with sc or Services MMC:")
		fmt.Println("  sc query  dnstester")
		fmt.Println("  sc stop   dnstester")
		fmt.Println("  sc start  dnstester")
	}
}
