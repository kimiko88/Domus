package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/domus-hub/domus/internal/domain/medicine"
	"github.com/google/uuid"
)

// MedicineRepository implements domain.Repository using SQLite.
type MedicineRepository struct {
	db *sql.DB
}

// NewMedicineRepository creates a new SQLite MedicineRepository.
func NewMedicineRepository(db *sql.DB) *MedicineRepository {
	return &MedicineRepository{db: db}
}

func (r *MedicineRepository) Create(ctx context.Context, item *domain.Medicine) error {
	if item.ID == "" {
		item.ID = "med_" + uuid.New().String()
	}
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	coldChainInt := 0
	if item.RequiresColdChain {
		coldChainInt = 1
	}
	rxInt := 0
	if item.PrescriptionRequired {
		rxInt = 1
	}

	query := `
		INSERT INTO medicines (
			id, name, active_ingredient, dosage, form,
			aic, barcode, lot_number, expiration_date, pao_days,
			opened_at, effective_expiration_date, location,
			requires_cold_chain, quantity, unit, min_threshold,
			prescription_required, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		item.ID,
		item.Name,
		item.ActiveIngredient,
		item.Dosage,
		item.Form,
		item.AIC,
		item.Barcode,
		item.LotNumber,
		item.ExpirationDate,
		item.PaoDays,
		item.OpenedAt,
		item.EffectiveExpirationDate,
		string(item.Location),
		coldChainInt,
		item.Quantity,
		string(item.Unit),
		item.MinThreshold,
		rxInt,
		item.Notes,
		item.CreatedAt.Format(time.RFC3339),
		item.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *MedicineRepository) GetByID(ctx context.Context, id string) (*domain.Medicine, error) {
	query := `
		SELECT id, name, active_ingredient, dosage, form,
		       aic, barcode, lot_number, expiration_date, pao_days,
		       opened_at, effective_expiration_date, location,
		       requires_cold_chain, quantity, unit, min_threshold,
		       prescription_required, notes, created_at, updated_at
		FROM medicines
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanMedicine(row)
}

func (r *MedicineRepository) Update(ctx context.Context, item *domain.Medicine) error {
	item.UpdatedAt = time.Now().UTC()

	coldChainInt := 0
	if item.RequiresColdChain {
		coldChainInt = 1
	}
	rxInt := 0
	if item.PrescriptionRequired {
		rxInt = 1
	}

	query := `
		UPDATE medicines SET
			name = ?, active_ingredient = ?, dosage = ?, form = ?,
			aic = ?, barcode = ?, lot_number = ?, expiration_date = ?, pao_days = ?,
			opened_at = ?, effective_expiration_date = ?, location = ?,
			requires_cold_chain = ?, quantity = ?, unit = ?, min_threshold = ?,
			prescription_required = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`

	res, err := r.db.ExecContext(ctx, query,
		item.Name,
		item.ActiveIngredient,
		item.Dosage,
		item.Form,
		item.AIC,
		item.Barcode,
		item.LotNumber,
		item.ExpirationDate,
		item.PaoDays,
		item.OpenedAt,
		item.EffectiveExpirationDate,
		string(item.Location),
		coldChainInt,
		item.Quantity,
		string(item.Unit),
		item.MinThreshold,
		rxInt,
		item.Notes,
		item.UpdatedAt.Format(time.RFC3339),
		item.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMedicineNotFound
	}
	return nil
}

func (r *MedicineRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM medicines WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMedicineNotFound
	}
	return nil
}

func (r *MedicineRepository) List(ctx context.Context, filter domain.Filter) ([]*domain.Medicine, error) {
	var sb strings.Builder
	var args []any

	sb.WriteString(`
		SELECT id, name, active_ingredient, dosage, form,
		       aic, barcode, lot_number, expiration_date, pao_days,
		       opened_at, effective_expiration_date, location,
		       requires_cold_chain, quantity, unit, min_threshold,
		       prescription_required, notes, created_at, updated_at
		FROM medicines
		WHERE 1=1
	`)

	if filter.Location != "" {
		sb.WriteString(" AND location = ?")
		args = append(args, string(filter.Location))
	}

	if filter.Search != "" {
		pattern := "%" + strings.ToLower(filter.Search) + "%"
		sb.WriteString(" AND (LOWER(name) LIKE ? OR LOWER(active_ingredient) LIKE ? OR aic LIKE ? OR barcode LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}

	if filter.ColdChain != nil {
		if *filter.ColdChain {
			sb.WriteString(" AND requires_cold_chain = 1")
		} else {
			sb.WriteString(" AND requires_cold_chain = 0")
		}
	}

	if filter.LowStock != nil && *filter.LowStock {
		sb.WriteString(" AND min_threshold > 0 AND quantity <= min_threshold")
	}

	nowStr := time.Now().UTC().Format("2006-01-02")
	if filter.IsExpired != nil && *filter.IsExpired {
		sb.WriteString(" AND effective_expiration_date < ?")
		args = append(args, nowStr)
	}

	if filter.ExpiringDays != nil && *filter.ExpiringDays > 0 {
		targetDate := time.Now().UTC().AddDate(0, 0, *filter.ExpiringDays).Format("2006-01-02")
		sb.WriteString(" AND effective_expiration_date >= ? AND effective_expiration_date <= ?")
		args = append(args, nowStr, targetDate)
	}

	sb.WriteString(" ORDER BY effective_expiration_date ASC, name ASC")

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list medicines failed: %w", err)
	}
	defer rows.Close()

	var items []*domain.Medicine
	for rows.Next() {
		m, err := r.scanMedicineRow(rows)
		if err != nil {
			return nil, fmt.Errorf("sqlite: scan medicine failed: %w", err)
		}
		items = append(items, m)
	}
	if items == nil {
		items = []*domain.Medicine{}
	}
	return items, nil
}

func (r *MedicineRepository) scanMedicine(s scanner) (*domain.Medicine, error) {
	return r.scanMedicineRow(s)
}

func (r *MedicineRepository) scanMedicineRow(s scanner) (*domain.Medicine, error) {
	var m domain.Medicine
	var activeIngredient, dosage, form, aic, barcode, lotNumber, openedAt, notes sql.NullString
	var coldChainInt, rxInt int
	var locStr, unitStr string
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&m.ID,
		&m.Name,
		&activeIngredient,
		&dosage,
		&form,
		&aic,
		&barcode,
		&lotNumber,
		&m.ExpirationDate,
		&m.PaoDays,
		&openedAt,
		&m.EffectiveExpirationDate,
		&locStr,
		&coldChainInt,
		&m.Quantity,
		&unitStr,
		&m.MinThreshold,
		&rxInt,
		&notes,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMedicineNotFound
		}
		return nil, err
	}

	m.ActiveIngredient = activeIngredient.String
	m.Dosage = dosage.String
	m.Form = form.String
	m.AIC = aic.String
	m.Barcode = barcode.String
	m.LotNumber = lotNumber.String
	if openedAt.Valid && openedAt.String != "" {
		val := openedAt.String
		m.OpenedAt = &val
	}
	m.Notes = notes.String
	m.Location = domain.Location(locStr)
	m.RequiresColdChain = (coldChainInt == 1)
	m.Unit = domain.Unit(unitStr)
	m.PrescriptionRequired = (rxInt == 1)

	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		m.CreatedAt = t
	} else if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
		m.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		m.UpdatedAt = t
	} else if t, err := time.Parse("2006-01-02 15:04:05", updatedAtStr); err == nil {
		m.UpdatedAt = t
	}

	return &m, nil
}
