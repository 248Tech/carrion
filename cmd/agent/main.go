// Package main is the mg7d agent entrypoint.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mg7d/mg7d/internal/actions"
	"github.com/mg7d/mg7d/internal/api"
	"github.com/mg7d/mg7d/internal/config"
	"github.com/mg7d/mg7d/internal/logtail"
	"github.com/mg7d/mg7d/internal/metrics"
	"github.com/mg7d/mg7d/internal/parser"
	"github.com/mg7d/mg7d/internal/policy"
	"github.com/mg7d/mg7d/internal/state"
	"github.com/mg7d/mg7d/internal/telnet"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Fatal("config load failed", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Single instance for Phase 0-3
	if len(cfg.Instances) == 0 {
		logger.Fatal("no instances in config")
	}
	inst := cfg.Instances[0]
	instanceName := inst.Name

	snapStore := state.NewSnapshotStore()
	auditRing := state.NewAuditRing(1024)
	metricsReg := metrics.NewRegistry(instanceName)
	metricsReg.RegisterCollectors()

	policyEngine := policy.NewEngine(instanceName, inst)

	var applier *actions.Applier
	if inst.Telnet.Host != "" && inst.Telnet.Port > 0 {
		telnetCfg := telnet.Config{
			Host:            inst.Telnet.Host,
			Port:            inst.Telnet.Port,
			Password:        inst.Telnet.Password,
			RateLimitPerSec: inst.Telnet.RateLimitPerSec,
		}
		telnetClient := telnet.NewClient(telnetCfg)
		go telnetClient.Run(ctx)
		applier = actions.NewApplier(telnetClient, auditRing, 32)
		if len(inst.Actions.Baseline) > 0 {
			applier.SetBaseline(inst.Actions.Baseline)
		}
		go applier.Run(ctx)
	}

	tailer, err := logtail.NewTailer(inst.LogPath, logtail.Options{})
	if err != nil {
		logger.Fatal("tailer create failed", zap.String("instance", instanceName), zap.Error(err))
	}
	linesCh := tailer.Lines()
	go func() {
		if err := tailer.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("tailer exited", zap.String("instance", instanceName), zap.Error(err))
		}
	}()

	// Parser goroutine: consume lines -> update snapshot -> metrics -> policy -> applier
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-linesCh:
				if !ok {
					return
				}
				snap, ok, parseErr := parser.ParseTimeLine(line)
				if parseErr != nil {
					logger.Debug("parse error", zap.String("line", line), zap.Error(parseErr))
					continue
				}
				if !ok {
					continue
				}
				snapStore.Update(snap)
				metricsReg.UpdateFromSnapshot(snap)
				if policyActions := policyEngine.Evaluate(snap); applier != nil && len(policyActions) > 0 {
					for _, a := range policyActions {
						if err := applier.Enqueue(ctx, a); err != nil {
							logger.Warn("applier enqueue failed", zap.String("action_id", a.ID()), zap.Error(err))
						}
					}
				}
			}
		}
	}()

	// Metrics HTTP server
	if cfg.Metrics.Enable {
		srv := api.NewMetricsServer(cfg.API.Listen, cfg.Metrics.Path, metricsReg.Handler())
		go func() {
			if err := srv.ListenAndServe(); err != nil && ctx.Err() == nil {
				logger.Error("metrics server failed", zap.Error(err))
			}
		}()
		defer srv.Shutdown(context.Background())
	}

	logger.Info("agent running", zap.String("instance", instanceName), zap.String("log_path", inst.LogPath))
	<-ctx.Done()
	logger.Info("agent shutting down")
}
