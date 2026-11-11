package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/domain"
	"reconpilot/internal/ingestion"
	"reconpilot/internal/matching"
	"reconpilot/internal/repository"
	"reconpilot/internal/seed"
	"reconpilot/internal/service"
	transporthttp "reconpilot/internal/transport/http"
	"reconpilot/internal/transport/stream"
)

var (
	ordersFilePath string
	pgFilePath     string
	bankFilePath   string
	exportCSVPath  string
	exportJSONPath string
	seedOutputDir  string
	serverPort     int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "reconpilot",
		Short: "ReconPilot: High-Throughput 3-Way Financial Reconciliation Engine",
		Long:  `ReconPilot is an institutional-grade 3-Way Financial Reconciliation & AI Discrepancy Platform built in Go.`,
	}

	// 1. Reconcile command
	reconcileCmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Run 3-way financial reconciliation on input CSV files",
		Run:   runReconciliationCLI,
	}
	reconcileCmd.Flags().StringVar(&ordersFilePath, "orders", "", "Path to Internal Orders CSV file")
	reconcileCmd.Flags().StringVar(&pgFilePath, "pg", "", "Path to Payment Gateway Settlements CSV file")
	reconcileCmd.Flags().StringVar(&bankFilePath, "bank", "", "Path to Bank Statements CSV file")
	reconcileCmd.Flags().StringVar(&exportCSVPath, "export-csv", "", "Optional path to save reconciled CSV trace output")
	reconcileCmd.Flags().StringVar(&exportJSONPath, "export-json", "", "Optional path to save reconciled JSON audit output")

	// 2. Seed demo CSVs command
	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Generate sample CSV datasets (Orders, PG Settlements, Bank Statement) for demo testing",
		Run:   runSeedCSVCLI,
	}
	seedCmd.Flags().StringVar(&seedOutputDir, "output-dir", "./samples", "Output directory to generate sample CSVs")

	// 3. Serve daemon command
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP & SSE API server daemon",
		Run:   runServeCLI,
	}
	serveCmd.Flags().IntVar(&serverPort, "port", 8080, "HTTP server listening port")

	// 4. Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print ReconPilot engine version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ReconPilot Engine v1.0.0 (Go 1.23+, PostgreSQL, Gemini 2.5 Flash)")
		},
	}

	rootCmd.AddCommand(reconcileCmd, seedCmd, serveCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runReconciliationCLI(cmd *cobra.Command, args []string) {
	if ordersFilePath == "" || pgFilePath == "" || bankFilePath == "" {
		log.Fatal("All three input files are required: --orders, --pg, and --bank")
	}

	ctx := context.Background()
	log.Printf("Parsing Orders CSV: %s", ordersFilePath)
	oFile, err := os.Open(ordersFilePath)
	if err != nil {
		log.Fatalf("Failed to open orders file: %v", err)
	}
	defer oFile.Close()
	orders, err := ingestion.ParseOrdersCSV(oFile, false)
	if err != nil {
		log.Fatalf("Parse orders CSV failed: %v", err)
	}

	log.Printf("Parsing PG Settlements CSV: %s", pgFilePath)
	pgFile, err := os.Open(pgFilePath)
	if err != nil {
		log.Fatalf("Failed to open pg file: %v", err)
	}
	defer pgFile.Close()
	settlements, err := ingestion.ParsePGSettlementsCSV(pgFile, false)
	if err != nil {
		log.Fatalf("Parse pg settlements CSV failed: %v", err)
	}

	log.Printf("Parsing Bank Statements CSV: %s", bankFilePath)
	bFile, err := os.Open(bankFilePath)
	if err != nil {
		log.Fatalf("Failed to open bank file: %v", err)
	}
	defer bFile.Close()
	bankStatements, err := ingestion.ParseBankStatementsCSV(bFile, false)
	if err != nil {
		log.Fatalf("Parse bank statements CSV failed: %v", err)
	}

	engine := matching.NewEngine(matching.DefaultEngineConfig())
	res, err := engine.Reconcile(ctx, orders, settlements, bankStatements, func(evt domain.PipelineTelemetryEvent) {
		log.Printf("[%s] (%.0f%%) %s", evt.Stage, evt.ProgressPercentage, evt.Message)
	})
	if err != nil {
		log.Fatalf("Reconciliation failed: %v", err)
	}

	fmt.Println("\n=======================================================")
	fmt.Println("             RECONCILIATION SUMMARY")
	fmt.Println("=======================================================")
	fmt.Printf("Total Orders:              %d\n", res.Summary.TotalOrders)
	fmt.Printf("Total Gross Volume:        ₹%s\n", res.Summary.TotalGrossOrderVolume.StringFixed(2))
	fmt.Printf("Total PG Settlements:      %d\n", res.Summary.TotalPGSettlements)
	fmt.Printf("Total PG Fees:             ₹%s\n", res.Summary.TotalPGFees.StringFixed(2))
	fmt.Printf("Total PG Tax (GST):        ₹%s\n", res.Summary.TotalPGTax.StringFixed(2))
	fmt.Printf("Expected Net Remittance:   ₹%s\n", res.Summary.ExpectedNetPayout.StringFixed(2))
	fmt.Printf("Total Bank Credits:        ₹%s\n", res.Summary.TotalBankCredits.StringFixed(2))
	fmt.Printf("Net Variance:              ₹%s\n", res.Summary.NetDiscrepancyVariance.StringFixed(2))
	fmt.Printf("3-Way Match Rate:          %s%%\n", res.Summary.ThreeWayMatchRate.StringFixed(2))
	fmt.Printf("Detected Discrepancies:    %d\n", len(res.Discrepancies))
	fmt.Println("=======================================================")

	exportSvc := service.NewExportService()
	if exportCSVPath != "" {
		data, err := exportSvc.ExportTracesCSV(res.TraceItems)
		if err == nil {
			_ = os.WriteFile(exportCSVPath, data, 0644)
			log.Printf("Exported 3-way trace CSV to: %s", exportCSVPath)
		}
	}
	if exportJSONPath != "" {
		data, err := exportSvc.ExportJSON(res)
		if err == nil {
			_ = os.WriteFile(exportJSONPath, data, 0644)
			log.Printf("Exported audit JSON to: %s", exportJSONPath)
		}
	}
}

func runSeedCSVCLI(cmd *cobra.Command, args []string) {
	if err := os.MkdirAll(seedOutputDir, 0755); err != nil {
		log.Fatalf("Failed to create seed directory: %v", err)
	}

	dataset := seed.BuildRealisticDataset()

	// Orders CSV
	var ordersBuf strings.Builder
	ordersBuf.WriteString("order_id,customer_id,gross_amount,currency,order_status,created_at\n")
	for _, o := range dataset.Orders {
		ordersBuf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			o.OrderID, o.CustomerID, o.GrossAmount.StringFixed(4), o.Currency, o.OrderStatus, o.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	// PG Settlements CSV
	var pgBuf strings.Builder
	pgBuf.WriteString("pg_txn_id,order_id,gross_amount,pg_fee,tax,net_amount,payout_id,payout_status,settled_at\n")
	for _, s := range dataset.PGSettlements {
		pgBuf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			s.PGTxnID, s.OrderID, s.GrossAmount.StringFixed(4), s.PGFee.StringFixed(4), s.Tax.StringFixed(4),
			s.NetAmount.StringFixed(4), s.PayoutID, s.PayoutStatus, s.SettledAt.Format("2006-01-02 15:04:05")))
	}

	// Bank Statements CSV
	var bankBuf strings.Builder
	bankBuf.WriteString("bank_txn_id,utr_number,credit_amount,description,transaction_date\n")
	for _, b := range dataset.BankStatements {
		bankBuf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			b.BankTxnID, b.UTRNumber, b.CreditAmount.StringFixed(4), b.Description, b.TransactionDate.Format("2006-01-02 15:04:05")))
	}

	_ = os.WriteFile(filepath.Join(seedOutputDir, "orders.csv"), []byte(ordersBuf.String()), 0644)
	_ = os.WriteFile(filepath.Join(seedOutputDir, "pg_settlements.csv"), []byte(pgBuf.String()), 0644)
	_ = os.WriteFile(filepath.Join(seedOutputDir, "bank_statements.csv"), []byte(bankBuf.String()), 0644)

	log.Printf("Sample CSV files generated in %s: %d orders, %d PG settlements, %d bank statements across 6 batches",
		seedOutputDir, len(dataset.Orders), len(dataset.PGSettlements), len(dataset.BankStatements))
}

func runServeCLI(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}
	if serverPort > 0 {
		cfg.Server.Port = serverPort
	}

	repo := repository.NewMockRepository(true)
	aiClient, _ := ai.NewClient(ctx, cfg.AI.GeminiAPIKey, cfg.AI.GeminiModel)
	engine := matching.NewEngine(matching.DefaultEngineConfig())
	hub := stream.NewHub()
	reconSvc := service.NewReconciliationService(repo, engine, aiClient, hub)
	exportSvc := service.NewExportService()

	router := transporthttp.NewRouter(cfg, reconSvc, exportSvc)
	log.Printf("Starting ReconPilot API server on :%d", cfg.Server.Port)
	if err := router.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
