package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/domain"
	"reconpilot/internal/orchestrator"
	"reconpilot/internal/repository"
	"reconpilot/internal/security"
	"reconpilot/internal/service"
	"reconpilot/pkg/runner"
	"reconpilot/pkg/validator"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "reconpilot",
		Short: "ReconPilot: Production-Grade External Attack Surface Management & Security Pipeline",
		Long: `ReconPilot is an automated External Attack Surface Management (EASM),
pipeline orchestrator, and AI-powered security synthesis platform.`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .env)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigFile(".env")
	}
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()
}

func main() {
	// 1. Scan Command
	var (
		scanTarget  string
		scanProfile string
		scanDryRun  bool
		scanExport  string
		scanOutput  string
	)

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Execute reconnaissance and vulnerability audit on an authorized target domain",
		Run: func(cmd *cobra.Command, args []string) {
			if scanTarget == "" {
				log.Fatal("\x1b[31m[ERROR]\x1b[0m --target domain is required")
			}

			if err := validator.ValidateFQDN(scanTarget); err != nil {
				log.Fatalf("\x1b[31m[ERROR]\x1b[0m Invalid target FQDN: %v", err)
			}

			cfg, err := config.Load()
			if err != nil {
				log.Fatalf("\x1b[31m[CONFIG_ERROR]\x1b[0m %v", err)
			}

			ctx := context.Background()

			// Initialize dependencies
			db, _ := repository.NewPostgresDB(ctx, &cfg.Database)
			hub := repository.NewRedisTelemetryHub(ctx, &cfg.Redis)
			safeRunner := runner.NewSafeRunner(cfg.Pipeline.StageTimeout, cfg.Pipeline.RingBufferSizeKB)
			orch := orchestrator.NewPipelineOrchestrator(cfg, db, hub, safeRunner)
			analyst := ai.NewGeminiAnalyst(&cfg.AI)
			exporter := service.NewExportService()

			scope := &domain.Scope{
				ID:                 uuid.New(),
				Name:               scanTarget,
				RootDomain:         scanTarget,
				VerificationStatus: domain.VerificationVerified,
				RateLimitRPS:       20,
			}

			job := &domain.ScanJob{
				ID:           uuid.New(),
				ScopeID:      scope.ID,
				Status:       domain.ScanQueued,
				Profile:      domain.ScanProfile(scanProfile),
				CurrentStage: domain.StageIdle,
				DryRun:       scanDryRun,
			}

			// Register job in orchestrator (supports standalone memory execution if DB is offline)
			orch.RegisterStandaloneJob(job, scope)
			if db != nil && db.Pool != nil {
				_ = db.Create(ctx, scope)
				_ = db.CreateJob(ctx, job)
			}

			fmt.Printf("\n\x1b[35;1m=== RECONPILOT ATTACK SURFACE ASSESSMENT ===\x1b[0m\n")
			fmt.Printf("Target:       \x1b[1m%s\x1b[0m\n", scanTarget)
			fmt.Printf("Scan Job ID:  %s\n", job.ID)
			fmt.Printf("Profile:      %s\n", scanProfile)
			fmt.Printf("Dry-Run Mode: %v\n\n", scanDryRun)

			// Subscribe to telemetry and print directly to console
			telemetryChan, unsubscribe := hub.Subscribe(ctx, job.ID)
			defer unsubscribe()

			go func() {
				for ev := range telemetryChan {
					if ev.Type == "log" {
						fmt.Println(ev.Payload)
					}
				}
			}()

			// Run scan in orchestrator
			if err := orch.ExecuteScan(ctx, job.ID); err != nil {
				log.Fatalf("\x1b[31m[SCAN_ERROR]\x1b[0m Pipeline failed: %v", err)
			}

			// Fetch results (from in-memory orchestrator or database)
			assets, findings, drifts := orch.GetJobArtifacts(job.ID)
			if db != nil && db.Pool != nil && len(assets) == 0 {
				assets, _ = db.ListAssetsByScanJob(ctx, job.ID)
				findings, _ = db.ListFindingsByScanJob(ctx, job.ID)
				drifts, _ = db.ListDriftsByScanJob(ctx, job.ID)
			}

			// Synthesize AI Report
			fmt.Printf("\n\x1b[36m[AI_SYNTHESIS]\x1b[0m Generating executive posture briefing and remediation playbook...\n")
			report, err := analyst.GenerateReport(ctx, job.ID, scanTarget, assets, findings, nil)
			if err != nil {
				log.Printf("\x1b[33m[WARN]\x1b[0m AI report synthesis failed: %v", err)
			} else {
				fmt.Printf("\n\x1b[32;1m================ EXECUTIVE SUMMARY ================\x1b[0m\n")
				fmt.Printf("Calculated Posture Score: \x1b[1m%d / 100\x1b[0m\n\n", report.OverallRiskScore)
				fmt.Println(report.ExecutiveSummary)
				fmt.Printf("\n\x1b[32;1m================ PRIORITIZED FINDINGS =============\x1b[0m\n")
				for _, f := range report.PrioritizedFindings {
					fmt.Printf("Rank #%d [%-8s] %-25s: %s\n", f.Rank, strings.ToUpper(string(f.Severity)), f.AffectedAsset, f.Title)
				}
			}

			// Handle export flag
			if scanExport == "markdown" {
				md, _ := exporter.ExportMarkdown(scope, job, assets, drifts, findings, report)
				fileName := scanOutput
				if fileName == "" {
					fileName = fmt.Sprintf("reconpilot-%s.md", scanTarget)
				}
				_ = os.WriteFile(fileName, []byte(md), 0644)
				fmt.Printf("\n\x1b[32m[EXPORT]\x1b[0m Written Markdown report to %s\n", fileName)
			} else if scanExport == "json" {
				jsonBytes, _ := exporter.ExportJSON(scope, job, assets, drifts, findings, report)
				fileName := scanOutput
				if fileName == "" {
					fileName = fmt.Sprintf("reconpilot-%s.json", scanTarget)
				}
				_ = os.WriteFile(fileName, jsonBytes, 0644)
				fmt.Printf("\n\x1b[32m[EXPORT]\x1b[0m Written JSON scan export to %s\n", fileName)
			}
		},
	}

	scanCmd.Flags().StringVarP(&scanTarget, "target", "t", "", "Target root domain to scan (e.g. example.com)")
	scanCmd.Flags().StringVarP(&scanProfile, "profile", "p", "standard", "Scan profile (passive, standard, deep)")
	scanCmd.Flags().BoolVarP(&scanDryRun, "dry-run", "d", false, "Simulate scan execution with realistic mock telemetry")
	scanCmd.Flags().StringVarP(&scanExport, "export", "e", "", "Export report format (markdown, json)")
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Output filepath for exported report")

	// 2. Verify Command
	var verifyTarget string
	var verifyToken string

	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify domain ownership challenge via DNS TXT records",
		Run: func(cmd *cobra.Command, args []string) {
			if verifyTarget == "" {
				log.Fatal("\x1b[31m[ERROR]\x1b[0m --target domain is required")
			}
			verifier := security.NewDNSVerifier(10 * time.Second)

			if verifyToken == "" {
				token, _ := verifier.GenerateToken()
				fmt.Printf("\nGenerated DNS verification challenge token:\n")
				fmt.Printf("  Record Name:  \x1b[1m_reconpilot-challenge.%s\x1b[0m\n", verifyTarget)
				fmt.Printf("  Record Type:  \x1b[1mTXT\x1b[0m\n")
				fmt.Printf("  Record Value: \x1b[32;1m%s\x1b[0m\n\n", token)
				fmt.Printf("Publish this record and verify using:\n  reconpilot verify -t %s --token %s\n", verifyTarget, token)
				return
			}

			verified, details, err := verifier.VerifyOwnership(context.Background(), verifyTarget, verifyToken)
			if err != nil {
				log.Fatalf("\x1b[31m[VERIFY_ERROR]\x1b[0m %v", err)
			}

			if verified {
				fmt.Printf("\x1b[32;1m[VERIFIED]\x1b[0m %s\n", details)
			} else {
				fmt.Printf("\x1b[31m[FAILED]\x1b[0m %s\n", details)
			}
		},
	}

	verifyCmd.Flags().StringVarP(&verifyTarget, "target", "t", "", "Root domain to verify")
	verifyCmd.Flags().StringVarP(&verifyToken, "token", "k", "", "Verification token to check")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(verifyCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

