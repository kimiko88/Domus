// Package sqlite provides SQLite persistence for household pantry inventory, recipes, and meal plans.
package sqlite

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"

	domain "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/google/uuid"
)

// PantryRepository implements domain.Repository using SQLite.
type PantryRepository struct {
	db *sql.DB
}

// NewPantryRepository constructs a new PantryRepository.
func NewPantryRepository(db *sql.DB) *PantryRepository {
	return &PantryRepository{db: db}
}

// --- PANTRY ITEMS ---

func (r *PantryRepository) CreatePantryItem(ctx context.Context, item *domain.PantryItem) error {
	if item.ID == "" {
		item.ID = "pantry_" + uuid.New().String()
	}
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	query := `
		INSERT INTO pantry_items (
			id, name, category, quantity, unit, min_threshold,
			expiration_date, location, barcode, cost, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		item.ID,
		item.Name,
		string(item.Category),
		item.Quantity,
		string(item.Unit),
		item.MinThreshold,
		item.ExpirationDate,
		string(item.Location),
		item.Barcode,
		item.Cost,
		item.CreatedAt.Format(time.RFC3339),
		item.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *PantryRepository) GetPantryItem(ctx context.Context, id string) (*domain.PantryItem, error) {
	query := `
		SELECT id, name, category, quantity, unit, min_threshold,
		       expiration_date, location, barcode, cost, created_at, updated_at
		FROM pantry_items
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanPantryItem(row)
}

func (r *PantryRepository) GetPantryItemByName(ctx context.Context, name string) (*domain.PantryItem, error) {
	query := `
		SELECT id, name, category, quantity, unit, min_threshold,
		       expiration_date, location, barcode, cost, created_at, updated_at
		FROM pantry_items
		WHERE LOWER(TRIM(name)) = LOWER(TRIM(?))
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, name)
	return scanPantryItem(row)
}

func (r *PantryRepository) UpdatePantryItem(ctx context.Context, item *domain.PantryItem) error {
	item.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE pantry_items
		SET name = ?, category = ?, quantity = ?, unit = ?, min_threshold = ?,
		    expiration_date = ?, location = ?, barcode = ?, cost = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		item.Name,
		string(item.Category),
		item.Quantity,
		string(item.Unit),
		item.MinThreshold,
		item.ExpirationDate,
		string(item.Location),
		item.Barcode,
		item.Cost,
		item.UpdatedAt.Format(time.RFC3339),
		item.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}

func (r *PantryRepository) DeletePantryItem(ctx context.Context, id string) error {
	query := `DELETE FROM pantry_items WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}

func (r *PantryRepository) ListPantryItems(ctx context.Context, category domain.Category, location domain.Location) ([]domain.PantryItem, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT id, name, category, quantity, unit, min_threshold,
		       expiration_date, location, barcode, cost, created_at, updated_at
		FROM pantry_items
		WHERE 1=1
	`)
	var args []any

	if category != "" {
		sb.WriteString(" AND category = ?")
		args = append(args, string(category))
	}
	if location != "" {
		sb.WriteString(" AND location = ?")
		args = append(args, string(location))
	}

	sb.WriteString(" ORDER BY name ASC")

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PantryItem
	for rows.Next() {
		item, err := scanPantryItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PantryRepository) ListExpiringPantryItems(ctx context.Context, days int) ([]domain.PantryItem, error) {
	deadline := time.Now().UTC().AddDate(0, 0, days).Format("2006-01-02")
	query := `
		SELECT id, name, category, quantity, unit, min_threshold,
		       expiration_date, location, barcode, cost, created_at, updated_at
		FROM pantry_items
		WHERE expiration_date IS NOT NULL
		  AND expiration_date <= ?
		ORDER BY expiration_date ASC
	`
	rows, err := r.db.QueryContext(ctx, query, deadline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PantryItem
	for rows.Next() {
		item, err := scanPantryItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PantryRepository) ListLowStockPantryItems(ctx context.Context) ([]domain.PantryItem, error) {
	query := `
		SELECT id, name, category, quantity, unit, min_threshold,
		       expiration_date, location, barcode, cost, created_at, updated_at
		FROM pantry_items
		WHERE quantity <= min_threshold
		ORDER BY quantity ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PantryItem
	for rows.Next() {
		item, err := scanPantryItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PantryRepository) AdjustPantryQuantity(ctx context.Context, id string, delta float64) (*domain.PantryItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, name, category, quantity, unit, min_threshold,
		       expiration_date, location, barcode, cost, created_at, updated_at
		FROM pantry_items
		WHERE id = ?
	`
	row := tx.QueryRowContext(ctx, query, id)
	item, err := scanPantryItem(row)
	if err != nil {
		return nil, err
	}

	newQty := math.Max(0, item.Quantity+delta)
	item.Quantity = math.Round(newQty*100) / 100
	item.UpdatedAt = time.Now().UTC()

	updateQuery := `UPDATE pantry_items SET quantity = ?, updated_at = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, updateQuery, item.Quantity, item.UpdatedAt.Format(time.RFC3339), item.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return item, nil
}

// --- RECIPES ---

func (r *PantryRepository) CreateRecipe(ctx context.Context, recipe *domain.Recipe) error {
	if recipe.ID == "" {
		recipe.ID = "recipe_" + uuid.New().String()
	}
	now := time.Now().UTC()
	if recipe.CreatedAt.IsZero() {
		recipe.CreatedAt = now
	}
	recipe.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO recipes (
			id, title, description, category, portions_base,
			prep_time_minutes, cook_time_minutes, instructions, image_url,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, query,
		recipe.ID,
		recipe.Title,
		recipe.Description,
		string(recipe.Category),
		recipe.PortionsBase,
		recipe.PrepTimeMinutes,
		recipe.CookTimeMinutes,
		recipe.Instructions,
		recipe.ImageURL,
		recipe.CreatedAt.Format(time.RFC3339),
		recipe.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}

	for _, ing := range recipe.Ingredients {
		if ing.ID == "" {
			ing.ID = "ing_" + uuid.New().String()
		}
		ingQuery := `
			INSERT INTO recipe_ingredients (
				id, recipe_id, pantry_item_id, name, quantity, unit, is_optional
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		opt := 0
		if ing.IsOptional {
			opt = 1
		}
		_, err = tx.ExecContext(ctx, ingQuery,
			ing.ID,
			recipe.ID,
			ing.PantryItemID,
			ing.Name,
			ing.Quantity,
			string(ing.Unit),
			opt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PantryRepository) GetRecipe(ctx context.Context, id string) (*domain.Recipe, error) {
	query := `
		SELECT id, title, description, category, portions_base,
		       prep_time_minutes, cook_time_minutes, instructions, image_url,
		       created_at, updated_at
		FROM recipes
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	recipe, err := scanRecipe(row)
	if err != nil {
		return nil, err
	}

	ingredients, err := r.loadRecipeIngredients(ctx, recipe.ID)
	if err != nil {
		return nil, err
	}
	recipe.Ingredients = ingredients
	return recipe, nil
}

func (r *PantryRepository) UpdateRecipe(ctx context.Context, recipe *domain.Recipe) error {
	recipe.UpdatedAt = time.Now().UTC()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE recipes
		SET title = ?, description = ?, category = ?, portions_base = ?,
		    prep_time_minutes = ?, cook_time_minutes = ?, instructions = ?,
		    image_url = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := tx.ExecContext(ctx, query,
		recipe.Title,
		recipe.Description,
		string(recipe.Category),
		recipe.PortionsBase,
		recipe.PrepTimeMinutes,
		recipe.CookTimeMinutes,
		recipe.Instructions,
		recipe.ImageURL,
		recipe.UpdatedAt.Format(time.RFC3339),
		recipe.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrRecipeNotFound
	}

	// Replace ingredients
	if _, err := tx.ExecContext(ctx, "DELETE FROM recipe_ingredients WHERE recipe_id = ?", recipe.ID); err != nil {
		return err
	}

	for _, ing := range recipe.Ingredients {
		if ing.ID == "" {
			ing.ID = "ing_" + uuid.New().String()
		}
		opt := 0
		if ing.IsOptional {
			opt = 1
		}
		ingQuery := `
			INSERT INTO recipe_ingredients (
				id, recipe_id, pantry_item_id, name, quantity, unit, is_optional
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err = tx.ExecContext(ctx, ingQuery,
			ing.ID,
			recipe.ID,
			ing.PantryItemID,
			ing.Name,
			ing.Quantity,
			string(ing.Unit),
			opt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PantryRepository) DeleteRecipe(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM recipes WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrRecipeNotFound
	}
	return nil
}

func (r *PantryRepository) ListRecipes(ctx context.Context, category domain.RecipeCategory) ([]domain.Recipe, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT id, title, description, category, portions_base,
		       prep_time_minutes, cook_time_minutes, instructions, image_url,
		       created_at, updated_at
		FROM recipes
		WHERE 1=1
	`)
	var args []any
	if category != "" {
		sb.WriteString(" AND category = ?")
		args = append(args, string(category))
	}
	sb.WriteString(" ORDER BY title ASC")

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []domain.Recipe
	for rows.Next() {
		rec, err := scanRecipe(rows)
		if err != nil {
			return nil, err
		}
		recipes = append(recipes, *rec)
	}

	// Load ingredients for each recipe
	for i := range recipes {
		ing, err := r.loadRecipeIngredients(ctx, recipes[i].ID)
		if err == nil {
			recipes[i].Ingredients = ing
		}
	}

	return recipes, rows.Err()
}

func (r *PantryRepository) loadRecipeIngredients(ctx context.Context, recipeID string) ([]domain.RecipeIngredient, error) {
	query := `
		SELECT id, recipe_id, pantry_item_id, name, quantity, unit, is_optional
		FROM recipe_ingredients
		WHERE recipe_id = ?
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.RecipeIngredient
	for rows.Next() {
		var ing domain.RecipeIngredient
		var optInt int
		var unitStr string
		err := rows.Scan(
			&ing.ID,
			&ing.RecipeID,
			&ing.PantryItemID,
			&ing.Name,
			&ing.Quantity,
			&unitStr,
			&optInt,
		)
		if err != nil {
			return nil, err
		}
		ing.Unit = domain.Unit(unitStr)
		ing.IsOptional = optInt == 1
		list = append(list, ing)
	}
	return list, rows.Err()
}

// --- MEAL PLANS ---

func (r *PantryRepository) CreateMealPlan(ctx context.Context, plan *domain.MealPlan) error {
	if plan.ID == "" {
		plan.ID = "meal_" + uuid.New().String()
	}
	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now

	query := `
		INSERT INTO meal_plans (
			id, plan_date, meal_type, recipe_id, custom_title,
			persons_count, notes, status, consumed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var consumedAtStr *string
	if plan.ConsumedAt != nil {
		s := plan.ConsumedAt.Format(time.RFC3339)
		consumedAtStr = &s
	}

	_, err := r.db.ExecContext(ctx, query,
		plan.ID,
		plan.PlanDate,
		string(plan.MealType),
		plan.RecipeID,
		plan.CustomTitle,
		plan.PersonsCount,
		plan.Notes,
		string(plan.Status),
		consumedAtStr,
		plan.CreatedAt.Format(time.RFC3339),
		plan.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *PantryRepository) GetMealPlan(ctx context.Context, id string) (*domain.MealPlan, error) {
	query := `
		SELECT id, plan_date, meal_type, recipe_id, custom_title,
		       persons_count, notes, status, consumed_at, created_at, updated_at
		FROM meal_plans
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanMealPlan(row)
}

func (r *PantryRepository) UpdateMealPlan(ctx context.Context, plan *domain.MealPlan) error {
	plan.UpdatedAt = time.Now().UTC()
	var consumedAtStr *string
	if plan.ConsumedAt != nil {
		s := plan.ConsumedAt.Format(time.RFC3339)
		consumedAtStr = &s
	}

	query := `
		UPDATE meal_plans
		SET plan_date = ?, meal_type = ?, recipe_id = ?, custom_title = ?,
		    persons_count = ?, notes = ?, status = ?, consumed_at = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		plan.PlanDate,
		string(plan.MealType),
		plan.RecipeID,
		plan.CustomTitle,
		plan.PersonsCount,
		plan.Notes,
		string(plan.Status),
		consumedAtStr,
		plan.UpdatedAt.Format(time.RFC3339),
		plan.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrMealPlanNotFound
	}
	return nil
}

func (r *PantryRepository) DeleteMealPlan(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM meal_plans WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrMealPlanNotFound
	}
	return nil
}

func (r *PantryRepository) ListMealPlans(ctx context.Context, startDate, endDate string) ([]domain.MealPlan, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT id, plan_date, meal_type, recipe_id, custom_title,
		       persons_count, notes, status, consumed_at, created_at, updated_at
		FROM meal_plans
		WHERE 1=1
	`)
	var args []any
	if startDate != "" {
		sb.WriteString(" AND plan_date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		sb.WriteString(" AND plan_date <= ?")
		args = append(args, endDate)
	}
	sb.WriteString(" ORDER BY plan_date ASC, meal_type ASC")

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.MealPlan
	for rows.Next() {
		plan, err := scanMealPlan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *plan)
	}

	// Populate recipe objects if referenced
	for i := range list {
		if list[i].RecipeID != nil && *list[i].RecipeID != "" {
			if rec, err := r.GetRecipe(ctx, *list[i].RecipeID); err == nil {
				list[i].Recipe = rec
			}
		}
	}

	return list, rows.Err()
}

func (r *PantryRepository) MarkMealPlanConsumed(ctx context.Context, id string, consumedAt time.Time) error {
	query := `
		UPDATE meal_plans
		SET status = 'consumed', consumed_at = ?, updated_at = ?
		WHERE id = ?
	`
	nowStr := consumedAt.Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, query, nowStr, nowStr, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrMealPlanNotFound
	}
	return nil
}

// --- SCAN HELPERS ---

func scanPantryItem(s scannable) (*domain.PantryItem, error) {
	var item domain.PantryItem
	var catStr, unitStr, locStr string
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&item.ID,
		&item.Name,
		&catStr,
		&item.Quantity,
		&unitStr,
		&item.MinThreshold,
		&item.ExpirationDate,
		&locStr,
		&item.Barcode,
		&item.Cost,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrItemNotFound
		}
		return nil, err
	}

	item.Category = domain.Category(catStr)
	item.Unit = domain.Unit(unitStr)
	item.Location = domain.Location(locStr)
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &item, nil
}

func scanRecipe(s scannable) (*domain.Recipe, error) {
	var rec domain.Recipe
	var catStr, createdAtStr, updatedAtStr string

	err := s.Scan(
		&rec.ID,
		&rec.Title,
		&rec.Description,
		&catStr,
		&rec.PortionsBase,
		&rec.PrepTimeMinutes,
		&rec.CookTimeMinutes,
		&rec.Instructions,
		&rec.ImageURL,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrRecipeNotFound
		}
		return nil, err
	}

	rec.Category = domain.RecipeCategory(catStr)
	rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &rec, nil
}

func scanMealPlan(s scannable) (*domain.MealPlan, error) {
	var plan domain.MealPlan
	var typeStr, statusStr string
	var consumedAtStr *string
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&plan.ID,
		&plan.PlanDate,
		&typeStr,
		&plan.RecipeID,
		&plan.CustomTitle,
		&plan.PersonsCount,
		&plan.Notes,
		&statusStr,
		&consumedAtStr,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrMealPlanNotFound
		}
		return nil, err
	}

	plan.MealType = domain.MealType(typeStr)
	plan.Status = domain.MealStatus(statusStr)
	plan.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	plan.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if consumedAtStr != nil && *consumedAtStr != "" {
		t, err := time.Parse(time.RFC3339, *consumedAtStr)
		if err == nil {
			plan.ConsumedAt = &t
		}
	}
	return &plan, nil
}
