package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"reconpilot/internal/config"
	"reconpilot/internal/domain"
)

// MongoDB encapsulates official MongoDB client connections and collection handlers.
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database

	scopes    *mongo.Collection
	scanJobs  *mongo.Collection
	assets    *mongo.Collection
	drifts    *mongo.Collection
	findings  *mongo.Collection
	aiReports *mongo.Collection
}

// NewMongoDB initializes a MongoDB connection pool using typed configuration.
func NewMongoDB(ctx context.Context, cfg *config.DatabaseConfig) (*MongoDB, error) {
	uri := cfg.URI
	if uri == "" {
		uri = cfg.URL
	}
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	dbName := cfg.Database
	if dbName == "" {
		dbName = "reconpilot"
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	clientOpts := options.Client().ApplyURI(uri).SetTimeout(timeout)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		// Log and return safe handle for offline / mock fallback
		return &MongoDB{}, nil
	}

	// Verify connectivity
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		// Server unreachable: keep collections nil so methods return offline-safe defaults
		return &MongoDB{}, nil
	}

	db := client.Database(dbName)
	m := &MongoDB{
		Client:    client,
		Database:  db,
		scopes:    db.Collection("scopes"),
		scanJobs:  db.Collection("scan_jobs"),
		assets:    db.Collection("assets"),
		drifts:    db.Collection("asset_drifts"),
		findings:  db.Collection("findings"),
		aiReports: db.Collection("ai_reports"),
	}

	// Initialize background compound and unique indexes
	go m.initIndexes(context.Background())

	return m, nil
}

// Ping verifies MongoDB server connectivity.
func (m *MongoDB) Ping(ctx context.Context) error {
	if m.Client == nil {
		return errors.New("mongodb client not connected")
	}
	return m.Client.Ping(ctx, nil)
}

// Close releases MongoDB connections.
func (m *MongoDB) Close(ctx context.Context) {
	if m.Client != nil {
		_ = m.Client.Disconnect(ctx)
	}
}

// initIndexes configures unique and lookup indexes for all collections.
func (m *MongoDB) initIndexes(ctx context.Context) {
	if m.Database == nil {
		return
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 1. Scopes: unique root_domain
	_, _ = m.scopes.Indexes().CreateOne(timeoutCtx, mongo.IndexModel{
		Keys:    bson.D{{Key: "root_domain", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// 2. ScanJobs: scope_id + created_at
	_, _ = m.scanJobs.Indexes().CreateMany(timeoutCtx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "scope_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	})

	// 3. Assets: compound unique (scan_job_id + fqdn)
	_, _ = m.assets.Indexes().CreateOne(timeoutCtx, mongo.IndexModel{
		Keys:    bson.D{{Key: "scan_job_id", Value: 1}, {Key: "fqdn", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// 4. Drifts: scan_job_id + detected_at
	_, _ = m.drifts.Indexes().CreateOne(timeoutCtx, mongo.IndexModel{
		Keys: bson.D{{Key: "scan_job_id", Value: 1}, {Key: "detected_at", Value: -1}},
	})

	// 5. Findings: scan_job_id + cvss_score
	_, _ = m.findings.Indexes().CreateMany(timeoutCtx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "scan_job_id", Value: 1}, {Key: "cvss_score", Value: -1}}},
		{Keys: bson.D{{Key: "asset_id", Value: 1}}},
	})

	// 6. AIReports: unique scan_job_id
	_, _ = m.aiReports.Indexes().CreateOne(timeoutCtx, mongo.IndexModel{
		Keys:    bson.D{{Key: "scan_job_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
}

// ==============================================================================
// SCOPE REPOSITORY IMPLEMENTATION
// ==============================================================================

func (m *MongoDB) Create(ctx context.Context, scope *domain.Scope) error {
	if m.scopes == nil {
		return nil
	}
	if scope.ID == uuid.Nil {
		scope.ID = uuid.New()
	}
	now := time.Now()
	scope.CreatedAt = now
	scope.UpdatedAt = now

	opts := options.Replace().SetUpsert(true)
	_, err := m.scopes.ReplaceOne(ctx, bson.M{"_id": scope.ID}, scope, opts)
	return err
}

func (m *MongoDB) GetByID(ctx context.Context, id uuid.UUID) (*domain.Scope, error) {
	if m.scopes == nil {
		return nil, nil
	}
	var s domain.Scope
	err := m.scopes.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (m *MongoDB) GetByRootDomain(ctx context.Context, rootDomain string) (*domain.Scope, error) {
	if m.scopes == nil {
		return nil, nil
	}
	var s domain.Scope
	err := m.scopes.FindOne(ctx, bson.M{"root_domain": rootDomain}).Decode(&s)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (m *MongoDB) Update(ctx context.Context, scope *domain.Scope) error {
	if m.scopes == nil {
		return nil
	}
	scope.UpdatedAt = time.Now()
	_, err := m.scopes.ReplaceOne(ctx, bson.M{"_id": scope.ID}, scope)
	return err
}

func (m *MongoDB) UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status domain.VerificationStatus) error {
	if m.scopes == nil {
		return nil
	}
	now := time.Now()
	setFields := bson.M{
		"verification_status": status,
		"updated_at":          now,
	}
	if status == domain.VerificationVerified {
		setFields["verified_at"] = now
	}

	_, err := m.scopes.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": setFields})
	return err
}

func (m *MongoDB) List(ctx context.Context) ([]*domain.Scope, error) {
	if m.scopes == nil {
		return []*domain.Scope{}, nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := m.scopes.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var scopes []*domain.Scope
	if err := cursor.All(ctx, &scopes); err != nil {
		return nil, err
	}
	if scopes == nil {
		scopes = []*domain.Scope{}
	}
	return scopes, nil
}

func (m *MongoDB) Delete(ctx context.Context, id uuid.UUID) error {
	if m.scopes == nil {
		return nil
	}
	_, err := m.scopes.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// ==============================================================================
// SCAN JOB REPOSITORY IMPLEMENTATION
// ==============================================================================

func (m *MongoDB) CreateJob(ctx context.Context, job *domain.ScanJob) error {
	if m.scanJobs == nil {
		return nil
	}
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now
	if job.StageCheckpoints == nil {
		job.StageCheckpoints = make(map[domain.StageName]domain.StageCheckpoint)
	}

	opts := options.Replace().SetUpsert(true)
	_, err := m.scanJobs.ReplaceOne(ctx, bson.M{"_id": job.ID}, job, opts)
	return err
}

func (m *MongoDB) GetJobByID(ctx context.Context, id uuid.UUID) (*domain.ScanJob, error) {
	if m.scanJobs == nil {
		return nil, nil
	}
	var j domain.ScanJob
	err := m.scanJobs.FindOne(ctx, bson.M{"_id": id}).Decode(&j)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	if j.StageCheckpoints == nil {
		j.StageCheckpoints = make(map[domain.StageName]domain.StageCheckpoint)
	}
	return &j, nil
}

func (m *MongoDB) GetLatestCompletedByScope(ctx context.Context, scopeID uuid.UUID, excludeJobID uuid.UUID) (*domain.ScanJob, error) {
	if m.scanJobs == nil {
		return nil, nil
	}
	filter := bson.M{
		"scope_id": scopeID,
		"status":   domain.ScanCompleted,
		"_id":      bson.M{"$ne": excludeJobID},
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "completed_at", Value: -1}})

	var j domain.ScanJob
	err := m.scanJobs.FindOne(ctx, filter, opts).Decode(&j)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &j, nil
}

func (m *MongoDB) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ScanStatus, errMessage *string) error {
	if m.scanJobs == nil {
		return nil
	}
	now := time.Now()
	setFields := bson.M{
		"status":     status,
		"updated_at": now,
	}
	if errMessage != nil {
		setFields["error_message"] = *errMessage
	}
	if status == domain.ScanCompleted || status == domain.ScanFailed || status == domain.ScanCancelled {
		setFields["completed_at"] = now
	}

	_, err := m.scanJobs.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": setFields})
	return err
}

func (m *MongoDB) UpdateCurrentStage(ctx context.Context, id uuid.UUID, stage domain.StageName) error {
	if m.scanJobs == nil {
		return nil
	}
	now := time.Now()
	_, err := m.scanJobs.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"current_stage": stage,
			"updated_at":    now,
		},
	})
	return err
}

func (m *MongoDB) SaveCheckpoint(ctx context.Context, id uuid.UUID, checkpoint domain.StageCheckpoint) error {
	if m.scanJobs == nil {
		return nil
	}
	now := time.Now()
	key := fmt.Sprintf("stage_checkpoints.%s", checkpoint.Stage)

	_, err := m.scanJobs.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			key:          checkpoint,
			"updated_at": now,
		},
	})
	return err
}

func (m *MongoDB) UpdateMetrics(ctx context.Context, id uuid.UUID, metrics domain.ScanMetrics) error {
	if m.scanJobs == nil {
		return nil
	}
	now := time.Now()
	_, err := m.scanJobs.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"metrics":    metrics,
			"updated_at": now,
		},
	})
	return err
}

func (m *MongoDB) ListJobsByScope(ctx context.Context, scopeID uuid.UUID) ([]*domain.ScanJob, error) {
	if m.scanJobs == nil {
		return []*domain.ScanJob{}, nil
	}
	filter := bson.M{"scope_id": scopeID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := m.scanJobs.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*domain.ScanJob
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	if jobs == nil {
		jobs = []*domain.ScanJob{}
	}
	return jobs, nil
}

// ==============================================================================
// ASSET REPOSITORY IMPLEMENTATION
// ==============================================================================

func (m *MongoDB) BatchInsertAssets(ctx context.Context, assets []*domain.Asset) error {
	if m.assets == nil || len(assets) == 0 {
		return nil
	}
	now := time.Now()
	models := make([]mongo.WriteModel, 0, len(assets))

	for _, a := range assets {
		if a.ID == uuid.Nil {
			a.ID = uuid.New()
		}
		if a.FirstSeenAt.IsZero() {
			a.FirstSeenAt = now
		}
		a.LastSeenAt = now
		if a.CreatedAt.IsZero() {
			a.CreatedAt = now
		}
		a.UpdatedAt = now

		filter := bson.M{"scan_job_id": a.ScanJobID, "fqdn": a.FQDN}
		models = append(models, mongo.NewReplaceOneModel().SetFilter(filter).SetReplacement(a).SetUpsert(true))
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := m.assets.BulkWrite(ctx, models, opts)
	return err
}

func (m *MongoDB) ListAssetsByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.Asset, error) {
	if m.assets == nil {
		return []*domain.Asset{}, nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "fqdn", Value: 1}})
	cursor, err := m.assets.Find(ctx, bson.M{"scan_job_id": scanJobID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var assets []*domain.Asset
	if err := cursor.All(ctx, &assets); err != nil {
		return nil, err
	}
	if assets == nil {
		assets = []*domain.Asset{}
	}
	return assets, nil
}

func (m *MongoDB) ListByScope(ctx context.Context, scopeID uuid.UUID) ([]*domain.Asset, error) {
	if m.assets == nil {
		return []*domain.Asset{}, nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "fqdn", Value: 1}})
	cursor, err := m.assets.Find(ctx, bson.M{"scope_id": scopeID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var assets []*domain.Asset
	if err := cursor.All(ctx, &assets); err != nil {
		return nil, err
	}
	if assets == nil {
		assets = []*domain.Asset{}
	}
	return assets, nil
}

// ==============================================================================
// ASSET DRIFT REPOSITORY IMPLEMENTATION
// ==============================================================================

func (m *MongoDB) BatchInsertDrifts(ctx context.Context, drifts []*domain.AssetDrift) error {
	if m.drifts == nil || len(drifts) == 0 {
		return nil
	}
	docs := make([]interface{}, len(drifts))
	now := time.Now()
	for i, d := range drifts {
		if d.ID == uuid.Nil {
			d.ID = uuid.New()
		}
		if d.DetectedAt.IsZero() {
			d.DetectedAt = now
		}
		docs[i] = d
	}

	opts := options.InsertMany().SetOrdered(false)
	_, err := m.drifts.InsertMany(ctx, docs, opts)
	return err
}

func (m *MongoDB) ListDriftsByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.AssetDrift, error) {
	if m.drifts == nil {
		return []*domain.AssetDrift{}, nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "detected_at", Value: -1}})
	cursor, err := m.drifts.Find(ctx, bson.M{"scan_job_id": scanJobID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var drifts []*domain.AssetDrift
	if err := cursor.All(ctx, &drifts); err != nil {
		return nil, err
	}
	if drifts == nil {
		drifts = []*domain.AssetDrift{}
	}
	return drifts, nil
}

// ==============================================================================
// FINDINGS REPOSITORY IMPLEMENTATION
// ==============================================================================

func (m *MongoDB) BatchInsertFindings(ctx context.Context, findings []*domain.Finding) error {
	if m.findings == nil || len(findings) == 0 {
		return nil
	}
	now := time.Now()
	docs := make([]interface{}, len(findings))
	for i, f := range findings {
		if f.ID == uuid.Nil {
			f.ID = uuid.New()
		}
		if f.CreatedAt.IsZero() {
			f.CreatedAt = now
		}
		f.UpdatedAt = now
		docs[i] = f
	}

	opts := options.InsertMany().SetOrdered(false)
	_, err := m.findings.InsertMany(ctx, docs, opts)
	return err
}

func (m *MongoDB) ListFindingsByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.Finding, error) {
	if m.findings == nil {
		return []*domain.Finding{}, nil
	}
	opts := options.Find().SetSort(bson.D{
		{Key: "cvss_score", Value: -1},
		{Key: "created_at", Value: -1},
	})
	cursor, err := m.findings.Find(ctx, bson.M{"scan_job_id": scanJobID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var findings []*domain.Finding
	if err := cursor.All(ctx, &findings); err != nil {
		return nil, err
	}
	if findings == nil {
		findings = []*domain.Finding{}
	}
	return findings, nil
}

func (m *MongoDB) ListByAsset(ctx context.Context, assetID uuid.UUID) ([]*domain.Finding, error) {
	if m.findings == nil {
		return []*domain.Finding{}, nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "cvss_score", Value: -1}})
	cursor, err := m.findings.Find(ctx, bson.M{"asset_id": assetID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var findings []*domain.Finding
	if err := cursor.All(ctx, &findings); err != nil {
		return nil, err
	}
	if findings == nil {
		findings = []*domain.Finding{}
	}
	return findings, nil
}

// ==============================================================================
// AI REPORT REPOSITORY IMPLEMENTATION
// ==============================================================================

func (m *MongoDB) CreateAIReport(ctx context.Context, report *domain.AIReport) error {
	if m.aiReports == nil {
		return nil
	}
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	report.GeneratedAt = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err := m.aiReports.ReplaceOne(ctx, bson.M{"scan_job_id": report.ScanJobID}, report, opts)
	return err
}

func (m *MongoDB) GetAIReportByScanJob(ctx context.Context, scanJobID uuid.UUID) (*domain.AIReport, error) {
	if m.aiReports == nil {
		return nil, nil
	}
	var r domain.AIReport
	err := m.aiReports.FindOne(ctx, bson.M{"scan_job_id": scanJobID}).Decode(&r)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}
