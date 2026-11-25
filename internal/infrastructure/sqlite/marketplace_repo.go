package sqlite

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/domus-hub/domus/internal/application/marketplace"
)

type MarketplaceRepository struct {
    db *sql.DB
}

func NewMarketplaceRepository(db *sql.DB) *MarketplaceRepository {
    return &MarketplaceRepository{db: db}
}

func (r *MarketplaceRepository) SaveSource(ctx context.Context, source *marketplace.Source) error {
    enabled := 0
    if source.Enabled { enabled = 1 }
    trusted := 0
    if source.Trusted { trusted = 1 }
    var synced *string
    if source.LastSyncedAt != nil {
        value := source.LastSyncedAt.UTC().Format(time.RFC3339Nano)
        synced = &value
    }
    created := source.CreatedAt
    if created.IsZero() { created = time.Now().UTC() }
    updated := source.UpdatedAt
    if updated.IsZero() { updated = time.Now().UTC() }
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO marketplace_repositories (
            id, url, name, owner, description, key_id, trusted, enabled,
            index_sha256, last_synced_at, last_error, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            url = excluded.url,
            name = excluded.name,
            owner = excluded.owner,
            description = excluded.description,
            key_id = excluded.key_id,
            trusted = excluded.trusted,
            enabled = excluded.enabled,
            index_sha256 = excluded.index_sha256,
            last_synced_at = excluded.last_synced_at,
            last_error = excluded.last_error,
            updated_at = excluded.updated_at
    `, source.ID, source.URL, source.Name, source.Owner, source.Description, source.KeyID,
        trusted, enabled, source.IndexSHA256, synced, source.LastError,
        created.UTC().Format(time.RFC3339Nano), updated.UTC().Format(time.RFC3339Nano))
    if err != nil { return fmt.Errorf("sqlite: save marketplace repository: %w", err) }
    return nil
}

func (r *MarketplaceRepository) GetSource(ctx context.Context, id string) (*marketplace.Source, error) {
    row := r.db.QueryRowContext(ctx, `
        SELECT id, url, name, owner, description, key_id, trusted, enabled,
               index_sha256, last_synced_at, last_error, created_at, updated_at
        FROM marketplace_repositories WHERE id = ?
    `, id)
    source, err := scanMarketplaceSource(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) { return nil, marketplace.ErrNotFound }
        return nil, fmt.Errorf("sqlite: get marketplace repository: %w", err)
    }
    return source, nil
}

func (r *MarketplaceRepository) ListSources(ctx context.Context) ([]*marketplace.Source, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, url, name, owner, description, key_id, trusted, enabled,
               index_sha256, last_synced_at, last_error, created_at, updated_at
        FROM marketplace_repositories ORDER BY name ASC, url ASC
    `)
    if err != nil { return nil, fmt.Errorf("sqlite: list marketplace repositories: %w", err) }
    defer rows.Close()
    result := make([]*marketplace.Source, 0)
    for rows.Next() {
        source, err := scanMarketplaceSource(rows)
        if err != nil { return nil, fmt.Errorf("sqlite: scan marketplace repository: %w", err) }
        result = append(result, source)
    }
    if err := rows.Err(); err != nil { return nil, err }
    return result, nil
}

func (r *MarketplaceRepository) DeleteSource(ctx context.Context, id string) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()
    if _, err := tx.ExecContext(ctx, "DELETE FROM marketplace_packages WHERE repository_id = ?", id); err != nil { return err }
    result, err := tx.ExecContext(ctx, "DELETE FROM marketplace_repositories WHERE id = ?", id)
    if err != nil { return err }
    count, _ := result.RowsAffected()
    if count == 0 { return marketplace.ErrNotFound }
    return tx.Commit()
}

func (r *MarketplaceRepository) SavePackage(ctx context.Context, pkg *marketplace.Package) error {
    capabilities, err := json.Marshal(pkg.Capabilities)
    if err != nil { return err }
    trusted, compatible := 0, 0
    if pkg.Trusted { trusted = 1 }
    if pkg.Compatible { compatible = 1 }
    var installedAt *string
    if pkg.InstalledAt != nil {
        value := pkg.InstalledAt.UTC().Format(time.RFC3339Nano)
        installedAt = &value
    }
    updated := pkg.UpdatedAt
    if updated.IsZero() { updated = time.Now().UTC() }
    _, err = r.db.ExecContext(ctx, `
        INSERT INTO marketplace_packages (
            id, repository_id, name, type, version, author, description, license, runtime,
            min_domus_version, max_domus_version, capabilities_json, artifact_url, artifact_sha256,
            artifact_size, trusted, compatible, installed_path, installed_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            repository_id = excluded.repository_id,
            name = excluded.name,
            type = excluded.type,
            version = excluded.version,
            author = excluded.author,
            description = excluded.description,
            license = excluded.license,
            runtime = excluded.runtime,
            min_domus_version = excluded.min_domus_version,
            max_domus_version = excluded.max_domus_version,
            capabilities_json = excluded.capabilities_json,
            artifact_url = excluded.artifact_url,
            artifact_sha256 = excluded.artifact_sha256,
            artifact_size = excluded.artifact_size,
            trusted = excluded.trusted,
            compatible = excluded.compatible,
            installed_path = CASE
                WHEN marketplace_packages.version = excluded.version
                 AND marketplace_packages.artifact_sha256 = excluded.artifact_sha256
                THEN marketplace_packages.installed_path ELSE '' END,
            installed_at = CASE
                WHEN marketplace_packages.version = excluded.version
                 AND marketplace_packages.artifact_sha256 = excluded.artifact_sha256
                THEN marketplace_packages.installed_at ELSE NULL END,
            updated_at = excluded.updated_at
    `, pkg.ID, pkg.RepositoryID, pkg.Name, pkg.Type, pkg.Version, pkg.Author, pkg.Description,
        pkg.License, pkg.Runtime, pkg.MinDomusVersion, pkg.MaxDomusVersion, string(capabilities),
        pkg.Artifact.URL, pkg.Artifact.SHA256, pkg.Artifact.Size, trusted, compatible,
        pkg.InstalledPath, installedAt, updated.UTC().Format(time.RFC3339Nano))
    if err != nil { return fmt.Errorf("sqlite: save marketplace package: %w", err) }
    return nil
}

func (r *MarketplaceRepository) GetPackage(ctx context.Context, id string) (*marketplace.Package, error) {
    row := r.db.QueryRowContext(ctx, marketplacePackageSelect+" WHERE id = ?", id)
    pkg, err := scanMarketplacePackage(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) { return nil, marketplace.ErrNotFound }
        return nil, fmt.Errorf("sqlite: get marketplace package: %w", err)
    }
    return pkg, nil
}

func (r *MarketplaceRepository) ListPackages(ctx context.Context, repositoryID string) ([]*marketplace.Package, error) {
    query := marketplacePackageSelect
    args := []any{}
    if repositoryID != "" {
        query += " WHERE repository_id = ?"
        args = append(args, repositoryID)
    }
    query += " ORDER BY name ASC, id ASC"
    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil { return nil, fmt.Errorf("sqlite: list marketplace packages: %w", err) }
    defer rows.Close()
    result := make([]*marketplace.Package, 0)
    for rows.Next() {
        pkg, err := scanMarketplacePackage(rows)
        if err != nil { return nil, fmt.Errorf("sqlite: scan marketplace package: %w", err) }
        result = append(result, pkg)
    }
    if err := rows.Err(); err != nil { return nil, err }
    return result, nil
}

func (r *MarketplaceRepository) DeletePackage(ctx context.Context, id string) error {
    result, err := r.db.ExecContext(ctx, "DELETE FROM marketplace_packages WHERE id = ?", id)
    if err != nil { return err }
    count, _ := result.RowsAffected()
    if count == 0 { return marketplace.ErrNotFound }
    return nil
}

const marketplacePackageSelect = `
    SELECT id, repository_id, name, type, version, author, description, license, runtime,
           min_domus_version, max_domus_version, capabilities_json, artifact_url, artifact_sha256,
           artifact_size, trusted, compatible, installed_path, installed_at, updated_at
    FROM marketplace_packages`

type marketplaceScanner interface { Scan(dest ...any) error }

func scanMarketplaceSource(row marketplaceScanner) (*marketplace.Source, error) {
    var source marketplace.Source
    var trusted, enabled int
    var synced, created, updated sql.NullString
    if err := row.Scan(
        &source.ID, &source.URL, &source.Name, &source.Owner, &source.Description, &source.KeyID,
        &trusted, &enabled, &source.IndexSHA256, &synced, &source.LastError, &created, &updated,
    ); err != nil { return nil, err }
    source.Trusted, source.Enabled = trusted == 1, enabled == 1
    if synced.Valid { if value, err := time.Parse(time.RFC3339Nano, synced.String); err == nil { source.LastSyncedAt = &value } }
    if created.Valid { source.CreatedAt, _ = time.Parse(time.RFC3339Nano, created.String) }
    if updated.Valid { source.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated.String) }
    return &source, nil
}

func scanMarketplacePackage(row marketplaceScanner) (*marketplace.Package, error) {
    var pkg marketplace.Package
    var capabilities string
    var trusted, compatible int
    var installedAt sql.NullString
    if err := row.Scan(
        &pkg.ID, &pkg.RepositoryID, &pkg.Name, &pkg.Type, &pkg.Version, &pkg.Author,
        &pkg.Description, &pkg.License, &pkg.Runtime, &pkg.MinDomusVersion, &pkg.MaxDomusVersion,
        &capabilities, &pkg.Artifact.URL, &pkg.Artifact.SHA256, &pkg.Artifact.Size,
        &trusted, &compatible, &pkg.InstalledPath, &installedAt, &pkg.UpdatedAt,
    ); err != nil { return nil, err }
    pkg.Trusted, pkg.Compatible = trusted == 1, compatible == 1
    if err := json.Unmarshal([]byte(capabilities), &pkg.Capabilities); err != nil { pkg.Capabilities = []string{} }
    if installedAt.Valid { if value, err := time.Parse(time.RFC3339Nano, installedAt.String); err == nil { pkg.InstalledAt = &value } }
    if pkg.UpdatedAt.IsZero() { pkg.UpdatedAt = time.Now().UTC() }
    return &pkg, nil
}
