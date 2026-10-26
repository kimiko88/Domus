package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/application/store"
	"github.com/google/uuid"
)

// StoreRepository implements store.Repository for SQLite.
type StoreRepository struct {
	db *sql.DB
}

// NewStoreRepository creates a new SQLite-backed StoreRepository.
func NewStoreRepository(db *sql.DB) *StoreRepository {
	return &StoreRepository{db: db}
}

// SavePackage inserts or updates a community store package.
func (r *StoreRepository) SavePackage(ctx context.Context, pkg *store.Package) error {
	if pkg.ID == "" {
		pkg.ID = "pkg_" + uuid.NewString()[:8]
	}

	installedInt := 0
	if pkg.Installed {
		installedInt = 1
	}

	var installedAtStr *string
	if pkg.InstalledAt != nil {
		s := pkg.InstalledAt.Format(time.RFC3339)
		installedAtStr = &s
	}

	tier := string(pkg.Tier)
	if tier == "" {
		tier = string(store.TierVerified)
	}

	reviewStatus := string(pkg.ReviewStatus)
	if reviewStatus == "" {
		reviewStatus = string(store.ReviewStatusApproved)
	}

	license := pkg.License
	if license == "" {
		license = "Apache-2.0"
	}

	icon := pkg.Icon
	if icon == "" {
		icon = "box"
	}

	capsBytes, _ := json.Marshal(pkg.Capabilities)
	capsJSON := string(capsBytes)
	if capsJSON == "" || capsJSON == "null" {
		capsJSON = "[]"
	}

	query := `
		INSERT INTO store_packages (
			id, package_id, name, type, version, author, description,
			installed, installed_at, tier, review_status, repository_url, license, capabilities_json, icon
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(package_id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			version = excluded.version,
			author = excluded.author,
			description = excluded.description,
			installed = excluded.installed,
			installed_at = excluded.installed_at,
			tier = excluded.tier,
			review_status = excluded.review_status,
			repository_url = excluded.repository_url,
			license = excluded.license,
			capabilities_json = excluded.capabilities_json,
			icon = excluded.icon
	`
	_, err := r.db.ExecContext(
		ctx, query,
		pkg.ID,
		pkg.PackageID,
		pkg.Name,
		string(pkg.Type),
		pkg.Version,
		pkg.Author,
		pkg.Description,
		installedInt,
		installedAtStr,
		tier,
		reviewStatus,
		pkg.RepositoryURL,
		license,
		capsJSON,
		icon,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save store package: %w", err)
	}
	return nil
}

// GetPackage retrieves a package by ID or package_id.
func (r *StoreRepository) GetPackage(ctx context.Context, id string) (*store.Package, error) {
	query := `
		SELECT id, package_id, name, type, version, author, description, installed, installed_at,
		       tier, review_status, repository_url, license, capabilities_json, icon
		FROM store_packages
		WHERE id = ? OR package_id = ?
	`
	var (
		pkg            store.Package
		pkgType        string
		description    sql.NullString
		installedInt   int
		installedAtStr sql.NullString
		tier           sql.NullString
		reviewStatus   sql.NullString
		repoURL        sql.NullString
		license        sql.NullString
		capsJSON       sql.NullString
		icon           sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query, id, id).Scan(
		&pkg.ID,
		&pkg.PackageID,
		&pkg.Name,
		&pkgType,
		&pkg.Version,
		&pkg.Author,
		&description,
		&installedInt,
		&installedAtStr,
		&tier,
		&reviewStatus,
		&repoURL,
		&license,
		&capsJSON,
		&icon,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("package not found: %s", id)
		}
		return nil, fmt.Errorf("sqlite: failed to get store package: %w", err)
	}

	pkg.Type = store.PackageType(pkgType)
	pkg.Description = description.String
	pkg.Installed = installedInt == 1
	if installedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, installedAtStr.String); err == nil {
			pkg.InstalledAt = &t
		}
	}

	if tier.Valid && tier.String != "" {
		pkg.Tier = store.PackageTier(tier.String)
	} else {
		pkg.Tier = store.TierVerified
	}

	if reviewStatus.Valid && reviewStatus.String != "" {
		pkg.ReviewStatus = store.ReviewStatus(reviewStatus.String)
	} else {
		pkg.ReviewStatus = store.ReviewStatusApproved
	}

	pkg.RepositoryURL = repoURL.String
	pkg.License = license.String
	if pkg.License == "" {
		pkg.License = "Apache-2.0"
	}
	pkg.Icon = icon.String
	if pkg.Icon == "" {
		pkg.Icon = "box"
	}

	if capsJSON.Valid && capsJSON.String != "" && capsJSON.String != "[]" {
		_ = json.Unmarshal([]byte(capsJSON.String), &pkg.Capabilities)
	}
	if pkg.Capabilities == nil {
		pkg.Capabilities = []string{}
	}

	return &pkg, nil
}

// ListPackages returns all packages matching filterType.
func (r *StoreRepository) ListPackages(ctx context.Context, filterType string) ([]*store.Package, error) {
	var query string
	var args []any
	if filterType != "" {
		query = `
			SELECT id, package_id, name, type, version, author, description, installed, installed_at,
			       tier, review_status, repository_url, license, capabilities_json, icon
			FROM store_packages
			WHERE type = ?
			ORDER BY name ASC
		`
		args = []any{filterType}
	} else {
		query = `
			SELECT id, package_id, name, type, version, author, description, installed, installed_at,
			       tier, review_status, repository_url, license, capabilities_json, icon
			FROM store_packages
			ORDER BY name ASC
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list store packages: %w", err)
	}
	defer rows.Close()

	var packages []*store.Package
	for rows.Next() {
		var (
			pkg            store.Package
			pkgType        string
			description    sql.NullString
			installedInt   int
			installedAtStr sql.NullString
			tier           sql.NullString
			reviewStatus   sql.NullString
			repoURL        sql.NullString
			license        sql.NullString
			capsJSON       sql.NullString
			icon           sql.NullString
		)
		if err := rows.Scan(
			&pkg.ID,
			&pkg.PackageID,
			&pkg.Name,
			&pkgType,
			&pkg.Version,
			&pkg.Author,
			&description,
			&installedInt,
			&installedAtStr,
			&tier,
			&reviewStatus,
			&repoURL,
			&license,
			&capsJSON,
			&icon,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan store package error: %w", err)
		}
		pkg.Type = store.PackageType(pkgType)
		pkg.Description = description.String
		pkg.Installed = installedInt == 1
		if installedAtStr.Valid {
			if t, err := time.Parse(time.RFC3339, installedAtStr.String); err == nil {
				pkg.InstalledAt = &t
			}
		}

		if tier.Valid && tier.String != "" {
			pkg.Tier = store.PackageTier(tier.String)
		} else {
			pkg.Tier = store.TierVerified
		}

		if reviewStatus.Valid && reviewStatus.String != "" {
			pkg.ReviewStatus = store.ReviewStatus(reviewStatus.String)
		} else {
			pkg.ReviewStatus = store.ReviewStatusApproved
		}

		pkg.RepositoryURL = repoURL.String
		pkg.License = license.String
		if pkg.License == "" {
			pkg.License = "Apache-2.0"
		}
		pkg.Icon = icon.String
		if pkg.Icon == "" {
			pkg.Icon = "box"
		}

		if capsJSON.Valid && capsJSON.String != "" && capsJSON.String != "[]" {
			_ = json.Unmarshal([]byte(capsJSON.String), &pkg.Capabilities)
		}
		if pkg.Capabilities == nil {
			pkg.Capabilities = []string{}
		}

		packages = append(packages, &pkg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: rows error: %w", err)
	}

	if packages == nil {
		packages = []*store.Package{}
	}
	return packages, nil
}

// DeletePackage removes a package record.
func (r *StoreRepository) DeletePackage(ctx context.Context, id string) error {
	query := `DELETE FROM store_packages WHERE id = ? OR package_id = ?`
	_, err := r.db.ExecContext(ctx, query, id, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete store package: %w", err)
	}
	return nil
}

// SeedDefaultPackages populates default curated store packages if the repository is empty.
func (r *StoreRepository) SeedDefaultPackages(ctx context.Context) error {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM store_packages").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	for _, p := range store.DefaultPackages() {
		if err := r.SavePackage(ctx, p); err != nil {
			return err
		}
	}
	return nil
}
