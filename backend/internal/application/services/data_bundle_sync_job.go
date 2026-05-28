package services

// data_bundle_sync_job.go — Periodic VTPass bundle catalog sync.
//
// Mirrors the RechargeMax DataBundleSyncJob pattern:
//   - Runs immediately on startup, then every DataBundleSyncInterval (≈4h48m).
//   - Upserts bundles into network_data_bundles via NetworkBundleService so that
//     GetBundles reads from DB rather than calling VTPass on every user page load.
//   - In sandbox mode (VTPASS_SANDBOX=true) sync failures are WARN-level.

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loyalty-nexus/internal/infrastructure/external"
)

// DataBundleSyncInterval is the time between VTPass bundle catalog refreshes
// (5 times per day, ≈ every 4h48m).
const DataBundleSyncInterval = (24 * time.Hour) / 5

// activeSyncNetworks are the VTPass network codes synced on each interval.
var activeSyncNetworks = []string{"MTN", "GLO", "AIRTEL", "9MOBILE"}

// networkDataBundleRow is the GORM model for network_data_bundles.
// Lives here to keep the sync logic self-contained; external.NetworkBundleService
// has its own private copy for reads.
type networkDataBundleRow struct {
	ID            string     `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	NetworkCode   string     `gorm:"column:network_code;not null"`
	VariationCode string     `gorm:"column:variation_code;not null"`
	Name          string     `gorm:"column:name;not null"`
	Price         float64    `gorm:"column:price;not null;default:0"`
	DataSize      string     `gorm:"column:data_size;not null;default:''"`
	IsActive      bool       `gorm:"column:is_active;not null;default:true"`
	LastSyncedAt  *time.Time `gorm:"column:last_synced_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (networkDataBundleRow) TableName() string { return "network_data_bundles" }

// DataBundleSyncJob periodically refreshes network_data_bundles from VTPass so
// that the frontend always serves bundles from the DB (fast, restart-safe).
type DataBundleSyncJob struct {
	db         *gorm.DB
	vtpass     *external.VTPassHTTPClient
	bundleSvc  *external.NetworkBundleService
	isSandbox  bool
	stopCh     chan struct{}
}

// NewDataBundleSyncJob creates the job.
// bundleSvc.InvalidateCache is called after each sync so the in-memory layer
// re-reads from DB on the next request.
func NewDataBundleSyncJob(
	db *gorm.DB,
	vtpass *external.VTPassHTTPClient,
	bundleSvc *external.NetworkBundleService,
) *DataBundleSyncJob {
	isSandbox := strings.EqualFold(os.Getenv("VTPASS_SANDBOX"), "true")
	return &DataBundleSyncJob{
		db:        db,
		vtpass:    vtpass,
		bundleSvc: bundleSvc,
		isSandbox: isSandbox,
		stopCh:    make(chan struct{}),
	}
}

// Start launches the sync goroutine. Runs once immediately, then on each tick.
func (j *DataBundleSyncJob) Start(ctx context.Context) {
	log.Printf("[DataBundleSyncJob] started: interval=%s networks=%v sandbox=%v",
		DataBundleSyncInterval, activeSyncNetworks, j.isSandbox)
	if j.isSandbox {
		log.Printf("[DataBundleSyncJob] sandbox mode — VTPass variations may not be available; sync errors are non-fatal")
	}
	go func() {
		// Immediate run on startup so bundles are in the DB before the first request.
		j.runOnce(ctx)

		ticker := time.NewTicker(DataBundleSyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				j.runOnce(ctx)
			case <-j.stopCh:
				log.Printf("[DataBundleSyncJob] stopped")
				return
			}
		}
	}()
}

// Stop signals the goroutine to exit.
func (j *DataBundleSyncJob) Stop() {
	select {
	case <-j.stopCh:
	default:
		close(j.stopCh)
	}
}

// ForceSync triggers an immediate out-of-schedule refresh (e.g. admin action).
func (j *DataBundleSyncJob) ForceSync(ctx context.Context) {
	go j.runOnce(ctx)
}

func (j *DataBundleSyncJob) runOnce(ctx context.Context) {
	total, errs := 0, 0
	for _, network := range activeSyncNetworks {
		n, err := j.syncNetwork(ctx, network)
		if err != nil {
			if j.isSandbox {
				log.Printf("[DataBundleSyncJob] WARN (sandbox) sync failed: network=%s err=%v", network, err)
			} else {
				log.Printf("[DataBundleSyncJob] ERROR sync failed: network=%s err=%v", network, err)
			}
			errs++
			continue
		}
		total += n
		// Flush in-memory cache so the next GetBundles call reads fresh DB data.
		if j.bundleSvc != nil {
			j.bundleSvc.InvalidateCache(network)
		}
	}
	log.Printf("[DataBundleSyncJob] sync complete: upserted=%d network_errors=%d", total, errs)
}

func (j *DataBundleSyncJob) syncNetwork(ctx context.Context, network string) (int, error) {
	if j.vtpass == nil {
		return 0, fmt.Errorf("VTPass client not available")
	}

	variations, err := j.vtpass.GetVariations(ctx, network)
	if err != nil {
		return 0, fmt.Errorf("GetVariations(%s): %w", network, err)
	}
	if len(variations) == 0 {
		log.Printf("[DataBundleSyncJob] WARN no variations returned: network=%s", network)
		return 0, nil
	}

	now := time.Now()
	upserted := 0

	for _, v := range variations {
		code := strings.TrimSpace(v.Code)
		if code == "" {
			continue
		}
		row := networkDataBundleRow{
			NetworkCode:   network,
			VariationCode: code,
			Name:          v.Name,
			Price:         v.Amount,
			DataSize:      extractSyncDataSize(v.Name),
			IsActive:      true,
			LastSyncedAt:  &now,
		}

		res := j.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "network_code"},
					{Name: "variation_code"},
				},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"name":           row.Name,
					"price":          row.Price,
					"data_size":      row.DataSize,
					"is_active":      true,
					"last_synced_at": now,
					"updated_at":     now,
				}),
			}).
			Create(&row)

		if res.Error != nil {
			log.Printf("[DataBundleSyncJob] WARN upsert failed: network=%s code=%s err=%v",
				network, code, res.Error)
			continue
		}
		upserted++
	}

	// Mark plans not seen in this sync as inactive (VTPass retired them).
	seenCodes := make([]string, 0, len(variations))
	for _, v := range variations {
		if v.Code != "" {
			seenCodes = append(seenCodes, v.Code)
		}
	}
	j.db.WithContext(ctx).Model(&networkDataBundleRow{}).
		Where("network_code = ? AND variation_code NOT IN ? AND is_active = true", network, seenCodes).
		Updates(map[string]interface{}{"is_active": false, "updated_at": now})

	log.Printf("[DataBundleSyncJob] network synced: network=%s upserted=%d total=%d",
		network, upserted, len(variations))
	return upserted, nil
}

// extractSyncDataSize parses "500MB", "1GB" etc. from a bundle name.
func extractSyncDataSize(name string) string {
	upper := strings.ToUpper(name)
	for _, unit := range []string{"TB", "GB", "MB", "KB"} {
		if idx := strings.Index(upper, unit); idx > 0 {
			start := idx - 1
			for start > 0 && (upper[start-1] >= '0' && upper[start-1] <= '9' || upper[start-1] == '.') {
				start--
			}
			return strings.TrimSpace(name[start : idx+len(unit)])
		}
	}
	return ""
}
