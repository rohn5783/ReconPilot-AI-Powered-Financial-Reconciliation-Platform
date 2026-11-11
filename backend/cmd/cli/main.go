package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"reconpilot/internal/ai"
	"reconpilot/internal/config"
	"reconpilot/internal/domain"
	"reconpilot/internal/ingestion"
	"reconpilot/internal/matching"
	"reconpilot/internal/repository"
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

	ordersCSV := `order_id,customer_id,gross_amount,currency,order_status,created_at
ORD-IN-1001,CUST-IN-901,10000.0000,INR,CAPTURED,2026-11-01 10:00:00
ORD-IN-1002,CUST-IN-902,20000.0000,INR,CAPTURED,2026-11-01 11:00:00
ORD-IN-1003,CUST-IN-903,30000.0000,INR,CAPTURED,2026-11-01 12:00:00
ORD-IN-1006,CUST-IN-906,45000.0000,INR,CAPTURED,2026-11-01 13:00:00
ORD-IN-1007,CUST-IN-907,18750.0000,INR,CAPTURED,2026-11-01 14:00:00
ORD-IN-1008,CUST-IN-908,12400.0000,INR,REFUNDED,2026-11-01 15:00:00
ORD-IN-1009,CUST-IN-909,35000.0000,INR,CAPTURED,2026-11-01 16:00:00
`
	pgCSV := `pg_txn_id,order_id,gross_amount,pg_fee,tax,net_amount,payout_id,payout_status,settled_at
pay_rzp_9001,ORD-IN-1001,10000.0000,200.0000,36.0000,9764.0000,PO-IN-2026-001,SETTLED,2026-11-02 10:00:00
pay_rzp_9002,ORD-IN-1002,20000.0000,400.0000,72.0000,19528.0000,PO-IN-2026-001,SETTLED,2026-11-02 11:00:00
pay_rzp_9003,ORD-IN-1003,30000.0000,600.0000,108.0000,29292.0000,PO-IN-2026-001,SETTLED,2026-11-02 12:00:00
pay_rzp_9006,ORD-IN-1006,45000.0000,1260.0000,226.8000,43513.2000,PO-IN-2026-002,SETTLED,2026-11-02 13:00:00
pay_rzp_9008,ORD-IN-1008,12400.0000,248.0000,44.6400,12107.3600,PO-IN-2026-002,SETTLED,2026-11-02 14:00:00
pay_rzp_9009,ORD-IN-1009,31500.0000,630.0000,113.4000,30756.6000,PO-IN-2026-002,SETTLED,2026-11-02 15:00:00
pay_rzp_8101,ORD-IN-1020,100000.0000,2000.0000,360.0000,97640.0000,PO-IN-2026-081,SETTLED,2026-11-02 16:00:00
`
	bankCSV := `bank_txn_id,utr_number,credit_amount,description,transaction_date
BNK-HDFC-001,HDFCR52026110290101,58584.0000,CMS/RAZORPAY SOFTWARE/PO-IN-2026-001/HDFCR52026110290101,2026-11-02 16:00:00
BNK-HDFC-002,HDFCR52026110290102,86377.1600,CMS/RAZORPAY SOFTWARE/PO-IN-2026-002/HDFCR52026110290102,2026-11-02 17:00:00
`
	_ = os.WriteFile(filepath.Join(seedOutputDir, "orders.csv"), []byte(ordersCSV), 0644)
	_ = os.WriteFile(filepath.Join(seedOutputDir, "pg_settlements.csv"), []byte(pgCSV), 0644)
	_ = os.WriteFile(filepath.Join(seedOutputDir, "bank_statements.csv"), []byte(bankCSV), 0644)

	log.Printf("Sample CSV files generated in: %s", seedOutputDir)
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
