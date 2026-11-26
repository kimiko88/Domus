package pantry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	domainFinance "github.com/domus-hub/domus/internal/domain/finance"
	domain "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/google/uuid"
)

// FinanceRecorder abstracts recording expenses into the Home Finance module.
type FinanceRecorder interface {
	RecordExpense(ctx context.Context, title string, amount float64, cat domainFinance.Category, subcat domainFinance.Subcategory, date time.Time, isFixed bool, notes string) (*domainFinance.Expense, error)
}

// ShoppingListAdder abstracts adding items into the household shopping list.
type ShoppingListAdder interface {
	AddItem(ctx context.Context, title string, quantity string, category string, addedBy string) error
}

// Service coordinates pantry inventory, recipes, meal planning, and receipt imports.
var (
	// ErrInvalidReceiptImport identifies user-supplied receipt data that cannot
	// be safely imported.
	ErrInvalidReceiptImport = errors.New("pantry: invalid receipt import")
	// ErrReceiptInventoryConflict indicates that an item cannot be merged into
	// the existing pantry inventory without changing its measurement unit.
	ErrReceiptInventoryConflict = errors.New("pantry: receipt item conflicts with pantry inventory")
	// ErrReceiptFinanceUnavailable is returned instead of silently skipping an
	// expense when the user explicitly requested financial registration.
	ErrReceiptFinanceUnavailable = errors.New("pantry: finance integration is unavailable")
)

type Service struct {
	repo        domain.Repository
	bus         events.Bus
	ocrParser   *ReceiptParser
	financeApp  FinanceRecorder
	shoppingApp ShoppingListAdder
	logger      *slog.Logger
}

// NewService instantiates the pantry application service.
func NewService(
	repo domain.Repository,
	bus events.Bus,
	financeApp FinanceRecorder,
	shoppingApp ShoppingListAdder,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:        repo,
		bus:         bus,
		ocrParser:   NewReceiptParser(),
		financeApp:  financeApp,
		shoppingApp: shoppingApp,
		logger:      logger,
	}
}

// --- PANTRY ITEMS ---

type CreatePantryItemRequest struct {
	Name           string          `json:"name"`
	Category       domain.Category `json:"category"`
	Quantity       float64         `json:"quantity"`
	Unit           domain.Unit     `json:"unit"`
	MinThreshold   float64         `json:"min_threshold"`
	ExpirationDate *string         `json:"expiration_date,omitempty"`
	Location       domain.Location `json:"location"`
	Barcode        *string         `json:"barcode,omitempty"`
	Cost           float64         `json:"cost"`
}

func (s *Service) CreatePantryItem(ctx context.Context, req CreatePantryItemRequest) (*domain.PantryItem, error) {
	now := time.Now().UTC()
	item := &domain.PantryItem{
		ID:             uuid.NewString(),
		Name:           req.Name,
		Category:       req.Category,
		Quantity:       req.Quantity,
		Unit:           req.Unit,
		MinThreshold:   req.MinThreshold,
		ExpirationDate: req.ExpirationDate,
		Location:       req.Location,
		Barcode:        req.Barcode,
		Cost:           req.Cost,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := item.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.CreatePantryItem(ctx, item); err != nil {
		return nil, fmt.Errorf("pantry: failed to create item: %w", err)
	}

	if item.IsLowStock() {
		s.publishLowStockEvent(item)
	}

	return item, nil
}

func (s *Service) GetPantryItem(ctx context.Context, id string) (*domain.PantryItem, error) {
	return s.repo.GetPantryItem(ctx, id)
}

func (s *Service) UpdatePantryItem(ctx context.Context, id string, req CreatePantryItemRequest) (*domain.PantryItem, error) {
	existing, err := s.repo.GetPantryItem(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Category = req.Category
	existing.Quantity = req.Quantity
	existing.Unit = req.Unit
	existing.MinThreshold = req.MinThreshold
	existing.ExpirationDate = req.ExpirationDate
	existing.Location = req.Location
	existing.Barcode = req.Barcode
	existing.Cost = req.Cost
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.UpdatePantryItem(ctx, existing); err != nil {
		return nil, fmt.Errorf("pantry: failed to update item: %w", err)
	}

	if existing.IsLowStock() {
		s.publishLowStockEvent(existing)
	}

	return existing, nil
}

func (s *Service) DeletePantryItem(ctx context.Context, id string) error {
	return s.repo.DeletePantryItem(ctx, id)
}

func (s *Service) ListPantryItems(ctx context.Context, category domain.Category, location domain.Location) ([]domain.PantryItem, error) {
	return s.repo.ListPantryItems(ctx, category, location)
}

func (s *Service) ListExpiringItems(ctx context.Context, days int) ([]domain.PantryItem, error) {
	if days <= 0 {
		days = 3
	}
	return s.repo.ListExpiringPantryItems(ctx, days)
}

func (s *Service) ListLowStockItems(ctx context.Context) ([]domain.PantryItem, error) {
	return s.repo.ListLowStockPantryItems(ctx)
}

func (s *Service) AdjustQuantity(ctx context.Context, id string, delta float64) (*domain.PantryItem, error) {
	item, err := s.repo.AdjustPantryQuantity(ctx, id, delta)
	if err != nil {
		return nil, err
	}
	if item.IsLowStock() {
		s.publishLowStockEvent(item)
	}
	return item, nil
}

// --- RECIPES ---

type RecipeIngredientRequest struct {
	PantryItemID *string     `json:"pantry_item_id,omitempty"`
	Name         string      `json:"name"`
	Quantity     float64     `json:"quantity"`
	Unit         domain.Unit `json:"unit"`
	IsOptional   bool        `json:"is_optional"`
}

type CreateRecipeRequest struct {
	Title           string                    `json:"title"`
	Description     string                    `json:"description,omitempty"`
	Category        domain.RecipeCategory     `json:"category"`
	PortionsBase    int                       `json:"portions_base"`
	PrepTimeMinutes int                       `json:"prep_time_minutes"`
	CookTimeMinutes int                       `json:"cook_time_minutes"`
	Instructions    string                    `json:"instructions"`
	ImageURL        *string                   `json:"image_url,omitempty"`
	Ingredients     []RecipeIngredientRequest `json:"ingredients"`
}

func (s *Service) CreateRecipe(ctx context.Context, req CreateRecipeRequest) (*domain.Recipe, error) {
	now := time.Now().UTC()
	recipeID := uuid.NewString()

	ingredients := make([]domain.RecipeIngredient, len(req.Ingredients))
	for i, ing := range req.Ingredients {
		ingredients[i] = domain.RecipeIngredient{
			ID:           uuid.NewString(),
			RecipeID:     recipeID,
			PantryItemID: ing.PantryItemID,
			Name:         ing.Name,
			Quantity:     ing.Quantity,
			Unit:         ing.Unit,
			IsOptional:   ing.IsOptional,
		}
	}

	recipe := &domain.Recipe{
		ID:              recipeID,
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		PortionsBase:    req.PortionsBase,
		PrepTimeMinutes: req.PrepTimeMinutes,
		CookTimeMinutes: req.CookTimeMinutes,
		Instructions:    req.Instructions,
		ImageURL:        req.ImageURL,
		Ingredients:     ingredients,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := recipe.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.CreateRecipe(ctx, recipe); err != nil {
		return nil, fmt.Errorf("pantry: failed to create recipe: %w", err)
	}

	return recipe, nil
}

func (s *Service) GetRecipe(ctx context.Context, id string) (*domain.Recipe, error) {
	return s.repo.GetRecipe(ctx, id)
}

func (s *Service) UpdateRecipe(ctx context.Context, id string, req CreateRecipeRequest) (*domain.Recipe, error) {
	existing, err := s.repo.GetRecipe(ctx, id)
	if err != nil {
		return nil, err
	}

	ingredients := make([]domain.RecipeIngredient, len(req.Ingredients))
	for i, ing := range req.Ingredients {
		ingredients[i] = domain.RecipeIngredient{
			ID:           uuid.NewString(),
			RecipeID:     id,
			PantryItemID: ing.PantryItemID,
			Name:         ing.Name,
			Quantity:     ing.Quantity,
			Unit:         ing.Unit,
			IsOptional:   ing.IsOptional,
		}
	}

	existing.Title = req.Title
	existing.Description = req.Description
	existing.Category = req.Category
	existing.PortionsBase = req.PortionsBase
	existing.PrepTimeMinutes = req.PrepTimeMinutes
	existing.CookTimeMinutes = req.CookTimeMinutes
	existing.Instructions = req.Instructions
	existing.ImageURL = req.ImageURL
	existing.Ingredients = ingredients
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateRecipe(ctx, existing); err != nil {
		return nil, fmt.Errorf("pantry: failed to update recipe: %w", err)
	}

	return existing, nil
}

func (s *Service) DeleteRecipe(ctx context.Context, id string) error {
	return s.repo.DeleteRecipe(ctx, id)
}

func (s *Service) ListRecipes(ctx context.Context, category domain.RecipeCategory) ([]domain.Recipe, error) {
	return s.repo.ListRecipes(ctx, category)
}

type CookableRecipeResult struct {
	Recipe            domain.Recipe `json:"recipe"`
	IsCookable        bool          `json:"is_cookable"`
	AvailablePercent  float64       `json:"available_percent"`
	MissingIngredient []string      `json:"missing_ingredients"`
}

// FindCookableRecipes checks all recipes against the currently available pantry items.
func (s *Service) FindCookableRecipes(ctx context.Context) ([]CookableRecipeResult, error) {
	recipes, err := s.repo.ListRecipes(ctx, "")
	if err != nil {
		return nil, err
	}

	pantryItems, err := s.repo.ListPantryItems(ctx, "", "")
	if err != nil {
		return nil, err
	}

	// Index pantry items by ID and by lower name
	byID := make(map[string]domain.PantryItem)
	byName := make(map[string]domain.PantryItem)
	for _, it := range pantryItems {
		byID[it.ID] = it
		byName[strings.ToLower(strings.TrimSpace(it.Name))] = it
	}

	results := make([]CookableRecipeResult, 0, len(recipes))
	for _, rec := range recipes {
		if len(rec.Ingredients) == 0 {
			results = append(results, CookableRecipeResult{
				Recipe:           rec,
				IsCookable:       true,
				AvailablePercent: 100,
			})
			continue
		}

		matchedCount := 0
		requiredCount := 0
		var missing []string

		for _, ing := range rec.Ingredients {
			if ing.IsOptional {
				continue
			}
			requiredCount++

			var matchedItem *domain.PantryItem
			if ing.PantryItemID != nil {
				if it, ok := byID[*ing.PantryItemID]; ok {
					matchedItem = &it
				}
			}
			if matchedItem == nil {
				if it, ok := byName[strings.ToLower(strings.TrimSpace(ing.Name))]; ok {
					matchedItem = &it
				}
			}

			if matchedItem != nil && matchedItem.Quantity >= ing.Quantity {
				matchedCount++
			} else {
				missing = append(missing, ing.Name)
			}
		}

		pct := 100.0
		if requiredCount > 0 {
			pct = float64(matchedCount) / float64(requiredCount) * 100.0
		}

		results = append(results, CookableRecipeResult{
			Recipe:            rec,
			IsCookable:        matchedCount == requiredCount,
			AvailablePercent:  pct,
			MissingIngredient: missing,
		})
	}

	return results, nil
}

// --- MEAL PLANNING ---

type CreateMealPlanRequest struct {
	PlanDate     string          `json:"plan_date"`
	MealType     domain.MealType `json:"meal_type"`
	RecipeID     *string         `json:"recipe_id,omitempty"`
	CustomTitle  *string         `json:"custom_title,omitempty"`
	PersonsCount int             `json:"persons_count"`
	Notes        *string         `json:"notes,omitempty"`
}

func (s *Service) CreateMealPlan(ctx context.Context, req CreateMealPlanRequest) (*domain.MealPlan, error) {
	now := time.Now().UTC()
	if req.PersonsCount < 1 {
		req.PersonsCount = 1
	}

	plan := &domain.MealPlan{
		ID:           uuid.NewString(),
		PlanDate:     req.PlanDate,
		MealType:     req.MealType,
		RecipeID:     req.RecipeID,
		CustomTitle:  req.CustomTitle,
		PersonsCount: req.PersonsCount,
		Notes:        req.Notes,
		Status:       domain.MealStatusPlanned,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := plan.Validate(); err != nil {
		return nil, err
	}

	if plan.RecipeID != nil && *plan.RecipeID != "" {
		recipe, err := s.repo.GetRecipe(ctx, *plan.RecipeID)
		if err == nil {
			plan.Recipe = recipe
		}
	}

	if err := s.repo.CreateMealPlan(ctx, plan); err != nil {
		return nil, fmt.Errorf("pantry: failed to create meal plan: %w", err)
	}

	return plan, nil
}

func (s *Service) GetMealPlan(ctx context.Context, id string) (*domain.MealPlan, error) {
	return s.repo.GetMealPlan(ctx, id)
}

func (s *Service) UpdateMealPlan(ctx context.Context, id string, req CreateMealPlanRequest) (*domain.MealPlan, error) {
	existing, err := s.repo.GetMealPlan(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.PlanDate = req.PlanDate
	existing.MealType = req.MealType
	existing.RecipeID = req.RecipeID
	existing.CustomTitle = req.CustomTitle
	existing.PersonsCount = req.PersonsCount
	existing.Notes = req.Notes
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateMealPlan(ctx, existing); err != nil {
		return nil, fmt.Errorf("pantry: failed to update meal plan: %w", err)
	}

	return existing, nil
}

func (s *Service) DeleteMealPlan(ctx context.Context, id string) error {
	return s.repo.DeleteMealPlan(ctx, id)
}

func (s *Service) ListMealPlans(ctx context.Context, startDate, endDate string) ([]domain.MealPlan, error) {
	return s.repo.ListMealPlans(ctx, startDate, endDate)
}

// ConsumeMealPlan marks a meal plan as consumed and automatically scales & decucts ingredients from pantry.
// If an ingredient's remaining quantity falls <= min_threshold, it publishes an alert and adds to the shopping list!
func (s *Service) ConsumeMealPlan(ctx context.Context, id string) (*domain.MealPlan, error) {
	plan, err := s.repo.GetMealPlan(ctx, id)
	if err != nil {
		return nil, err
	}

	if plan.Status == domain.MealStatusConsumed {
		return plan, nil // Already consumed
	}

	// If linked to a recipe, deduct proportionally
	if plan.RecipeID != nil && *plan.RecipeID != "" {
		recipe, err := s.repo.GetRecipe(ctx, *plan.RecipeID)
		if err == nil && recipe != nil {
			plan.Recipe = recipe
			scaledIngredients := recipe.ScaleIngredients(plan.PersonsCount)

			// Load all pantry items for matching
			pantryList, _ := s.repo.ListPantryItems(ctx, "", "")
			byID := make(map[string]domain.PantryItem)
			byName := make(map[string]domain.PantryItem)
			for _, it := range pantryList {
				byID[it.ID] = it
				byName[strings.ToLower(strings.TrimSpace(it.Name))] = it
			}

			for _, ing := range scaledIngredients {
				var targetID string
				if ing.PantryItemID != nil && *ing.PantryItemID != "" {
					targetID = *ing.PantryItemID
				} else if it, found := byName[strings.ToLower(strings.TrimSpace(ing.Name))]; found {
					targetID = it.ID
				}

				if targetID != "" {
					// Deduct stock proportionally
					updated, err := s.repo.AdjustPantryQuantity(ctx, targetID, -ing.Quantity)
					if err == nil && updated != nil {
						if updated.IsLowStock() {
							s.publishLowStockEvent(updated)
						}
					}
				}
			}
		}
	}

	now := time.Now().UTC()
	plan.Status = domain.MealStatusConsumed
	plan.ConsumedAt = &now
	plan.UpdatedAt = now

	if err := s.repo.MarkMealPlanConsumed(ctx, id, now); err != nil {
		return nil, fmt.Errorf("pantry: failed to mark meal plan consumed: %w", err)
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.MealPlanConsumed,
			OccurredAt: now,
			Payload:    plan,
			Source:     "pantry_service",
		})
	}

	return plan, nil
}

// --- RECEIPT OCR & IMPORT ---

// ParseReceipt analyzes raw receipt plain text and returns structured line items, categories, and totals.
func (s *Service) ParseReceipt(ctx context.Context, rawText string) (*domain.Receipt, error) {
	return s.ocrParser.ParseText(rawText)
}

type ImportReceiptRequest struct {
	DocumentType     string               `json:"document_type"`
	StoreName        string               `json:"store_name"`
	Date             string               `json:"date"`
	Items            []domain.ReceiptItem `json:"items"`
	TotalAmount      float64              `json:"total_amount"`
	AutoAddToFinance bool                 `json:"auto_add_to_finance"`
	AutoUpdatePantry bool                 `json:"auto_update_pantry"`
}

// ImportReceipt persists recognized items into the pantry and automatically records the expense in Home Finance!
type receiptInventoryPlan struct {
	before   *domain.PantryItem
	updated  *domain.PantryItem
	created  *domain.PantryItem
}

type receiptInventoryMutation struct {
	before   *domain.PantryItem
	createdID string
}

// ImportReceipt validates the complete request before mutating any state.
// Pantry mutations are compensated if a later pantry write or finance write
// fails, preventing the common retry case from duplicating stock quantities.
func (s *Service) ImportReceipt(ctx context.Context, req ImportReceiptRequest) (*domain.ReceiptImportResult, error) {
	documentType := strings.ToLower(strings.TrimSpace(req.DocumentType))
	if documentType == "" {
		documentType = "receipt"
	}
	if documentType != "receipt" && documentType != "invoice" {
		return nil, fmt.Errorf("%w: unsupported financial document type %q", ErrInvalidReceiptImport, documentType)
	}
	if documentType == "invoice" {
		req.AutoUpdatePantry = false
	}
	if len(req.Items) > 500 {
		return nil, fmt.Errorf("%w: receipt contains too many items", ErrInvalidReceiptImport)
	}
	if math.IsNaN(req.TotalAmount) || math.IsInf(req.TotalAmount, 0) || req.TotalAmount < 0 || req.TotalAmount > 1_000_000 {
		return nil, fmt.Errorf("%w: receipt total amount is invalid", ErrInvalidReceiptImport)
	}
	req.StoreName = strings.TrimSpace(req.StoreName)
	if req.StoreName == "" {
		req.StoreName = "Supermercato"
	}
	if len(req.StoreName) > 150 {
		return nil, fmt.Errorf("%w: store name is too long", ErrInvalidReceiptImport)
	}
	if req.Date != "" {
		if _, err := time.Parse("2006-01-02", req.Date); err != nil {
			return nil, fmt.Errorf("%w: receipt date must use YYYY-MM-DD", ErrInvalidReceiptImport)
		}
	}
	if req.AutoAddToFinance && req.TotalAmount > 0 && s.financeApp == nil {
		return nil, ErrReceiptFinanceUnavailable
	}

	// Normalize and validate every input item before querying or mutating the
	// pantry. This also merges repeated product lines from the same receipt.
	items, err := normalizeReceiptImportItems(req.Items, req.AutoUpdatePantry)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	result := &domain.ReceiptImportResult{ExpenseAmount: req.TotalAmount}
	itemNames := make([]string, 0, len(items))
	for _, item := range items {
		itemNames = append(itemNames, fmt.Sprintf("%s (%.3g %s)", item.Name, item.Quantity, item.Unit))
	}

	var plans []receiptInventoryPlan
	if req.AutoUpdatePantry {
		plans = make([]receiptInventoryPlan, 0, len(items))
		for _, item := range items {
			existing, lookupErr := s.repo.GetPantryItemByName(ctx, item.Name)
			switch {
			case lookupErr == nil && existing != nil:
				if existing.Unit != item.Unit {
					return nil, fmt.Errorf("%w: %q is stored as %q but the receipt uses %q",
						ErrReceiptInventoryConflict, item.Name, existing.Unit, item.Unit)
				}
				before := clonePantryItem(existing)
				updated := clonePantryItem(existing)
				updated.Quantity += item.Quantity
				if math.IsNaN(updated.Quantity) || math.IsInf(updated.Quantity, 0) || updated.Quantity > 1_000_000 {
					return nil, fmt.Errorf("%w: merged quantity for %q exceeds the supported limit", ErrInvalidReceiptImport, item.Name)
				}
				updated.Cost = receiptUnitCost(item)
				updated.UpdatedAt = now
				if err := updated.Validate(); err != nil {
					return nil, fmt.Errorf("%w: merged pantry item %q is invalid: %v",
						ErrInvalidReceiptImport, item.Name, err)
				}
				plans = append(plans, receiptInventoryPlan{before: before, updated: updated})
			case errors.Is(lookupErr, domain.ErrItemNotFound) || (lookupErr == nil && existing == nil):
				newItem := &domain.PantryItem{
					ID:           uuid.NewString(),
					Name:         item.Name,
					Category:     item.Category,
					Quantity:     item.Quantity,
					Unit:         item.Unit,
					MinThreshold: 1.0,
					Location:     domain.LocationPantry,
					Cost:         receiptUnitCost(item),
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				if err := newItem.Validate(); err != nil {
					return nil, fmt.Errorf("%w: pantry item %q is invalid: %v",
						ErrInvalidReceiptImport, item.Name, err)
				}
				plans = append(plans, receiptInventoryPlan{created: newItem})
			default:
				return nil, fmt.Errorf("pantry: failed to find existing receipt item %q: %w", item.Name, lookupErr)
			}
		}
	}

	applied := make([]receiptInventoryMutation, 0, len(plans))
	for _, plan := range plans {
		if plan.updated != nil {
			if err := s.repo.UpdatePantryItem(ctx, plan.updated); err != nil {
				rollbackErr := s.rollbackReceiptInventory(applied)
				return nil, errors.Join(fmt.Errorf("pantry: failed to update receipt item %q: %w", plan.updated.Name, err), rollbackErr)
			}
			applied = append(applied, receiptInventoryMutation{before: plan.before})
			result.ItemsUpdated++
			continue
		}
		if err := s.repo.CreatePantryItem(ctx, plan.created); err != nil {
			rollbackErr := s.rollbackReceiptInventory(applied)
			return nil, errors.Join(fmt.Errorf("pantry: failed to create receipt item %q: %w", plan.created.Name, err), rollbackErr)
		}
		applied = append(applied, receiptInventoryMutation{createdID: plan.created.ID})
		result.ItemsAdded++
	}

	// Financial registration is deliberately last. If it fails, undo the
	// already-applied inventory updates so the user can retry safely.
	if req.AutoAddToFinance && req.TotalAmount > 0 {
		expDate := now
		if parsedDate, err := time.Parse("2006-01-02", req.Date); err == nil {
			expDate = parsedDate
		}

		category := domainFinance.CategoryFood
		subcategory := domainFinance.SubcategoryGroceries
		title := fmt.Sprintf("Spesa %s", req.StoreName)
		documentLabel := "scontrino"
		if documentType == "invoice" {
			category = domainFinance.CategoryOther
			subcategory = ""
			title = fmt.Sprintf("Fattura %s", req.StoreName)
			documentLabel = "fattura"
		}
		notes := fmt.Sprintf("Import da %s: %s", documentLabel, strings.Join(itemNames, ", "))
		exp, err := s.financeApp.RecordExpense(ctx, title, req.TotalAmount, category, subcategory, expDate, false, notes)
		if err != nil {
			rollbackErr := s.rollbackReceiptInventory(applied)
			return nil, errors.Join(fmt.Errorf("pantry: failed to record receipt expense: %w", err), rollbackErr)
		}
		if exp != nil {
			result.ExpenseID = &exp.ID
		}
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.ReceiptProcessed,
			OccurredAt: now,
			Payload:    result,
			Source:     "pantry_service",
		})
	}
	return result, nil
}

func normalizeReceiptImportItems(input []domain.ReceiptItem, forPantry bool) ([]domain.ReceiptItem, error) {
	items := make([]domain.ReceiptItem, 0, len(input))
	positions := make(map[string]int, len(input))
	validUnits := map[domain.Unit]bool{
		domain.UnitPiece: true, domain.UnitGram: true, domain.UnitKilogram: true,
		domain.UnitMillilit: true, domain.UnitLiter: true, domain.UnitPack: true,
	}
	validCategories := make(map[domain.Category]bool, len(domain.ValidCategories))
	for _, category := range domain.ValidCategories {
		validCategories[category] = true
	}

	for index, item := range input {
		item.Name = strings.TrimSpace(item.Name)
		maxNameLength := 200
		if forPantry {
			maxNameLength = 150
		}
		if item.Name == "" || len(item.Name) > maxNameLength {
			return nil, fmt.Errorf("%w: item %d has an invalid name", ErrInvalidReceiptImport, index+1)
		}
		if math.IsNaN(item.Quantity) || math.IsInf(item.Quantity, 0) || item.Quantity <= 0 || item.Quantity > 1_000_000 {
			return nil, fmt.Errorf("%w: item %d has an invalid quantity", ErrInvalidReceiptImport, index+1)
		}
		if math.IsNaN(item.TotalPrice) || math.IsInf(item.TotalPrice, 0) || item.TotalPrice < 0 || item.TotalPrice > 1_000_000 {
			return nil, fmt.Errorf("%w: item %d has an invalid total price", ErrInvalidReceiptImport, index+1)
		}
		if math.IsNaN(item.UnitPrice) || math.IsInf(item.UnitPrice, 0) || item.UnitPrice < 0 || item.UnitPrice > 1_000_000 {
			return nil, fmt.Errorf("%w: item %d has an invalid unit price", ErrInvalidReceiptImport, index+1)
		}
		if item.Unit == "" {
			item.Unit = domain.UnitPiece
		}
		if !validUnits[item.Unit] {
			return nil, fmt.Errorf("%w: item %d has an unsupported measurement unit", ErrInvalidReceiptImport, index+1)
		}
		if item.Category == "" {
			item.Category = domain.CategoryOther
		}
		if !validCategories[item.Category] {
			return nil, fmt.Errorf("%w: item %d has an unsupported category", ErrInvalidReceiptImport, index+1)
		}
		if item.UnitPrice == 0 && item.TotalPrice > 0 {
			item.UnitPrice = item.TotalPrice / item.Quantity
		}

		key := strings.ToLower(item.Name)
		if !forPantry {
			items = append(items, item)
			continue
		}
		if position, exists := positions[key]; exists {
			current := &items[position]
			if current.Unit != item.Unit {
				return nil, fmt.Errorf("%w: duplicate receipt lines for %q use different units", ErrReceiptInventoryConflict, item.Name)
			}
			current.Quantity += item.Quantity
			current.TotalPrice += item.TotalPrice
			if current.Category == domain.CategoryOther && item.Category != domain.CategoryOther {
				current.Category = item.Category
			}
			current.UnitPrice = current.TotalPrice / current.Quantity
			if current.Quantity > 1_000_000 || current.TotalPrice > 1_000_000 {
				return nil, fmt.Errorf("%w: combined item %q exceeds the supported limit", ErrInvalidReceiptImport, item.Name)
			}
			continue
		}
		positions[key] = len(items)
		items = append(items, item)
	}
	return items, nil
}

func receiptUnitCost(item domain.ReceiptItem) float64 {
	if item.UnitPrice > 0 {
		return item.UnitPrice
	}
	if item.Quantity > 0 {
		return item.TotalPrice / item.Quantity
	}
	return item.TotalPrice
}

func clonePantryItem(item *domain.PantryItem) *domain.PantryItem {
	if item == nil {
		return nil
	}
	copy := *item
	if item.ExpirationDate != nil {
		value := *item.ExpirationDate
		copy.ExpirationDate = &value
	}
	if item.Barcode != nil {
		value := *item.Barcode
		copy.Barcode = &value
	}
	return &copy
}

func (s *Service) rollbackReceiptInventory(applied []receiptInventoryMutation) error {
	if len(applied) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var rollbackErrors []error
	for index := len(applied) - 1; index >= 0; index-- {
		change := applied[index]
		if change.before != nil {
			if err := s.repo.UpdatePantryItem(ctx, change.before); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore pantry item %q: %w", change.before.Name, err))
			}
			continue
		}
		if change.createdID != "" {
			if err := s.repo.DeletePantryItem(ctx, change.createdID); err != nil && !errors.Is(err, domain.ErrItemNotFound) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove newly added pantry item %q: %w", change.createdID, err))
			}
		}
	}
	if len(rollbackErrors) == 0 {
		return nil
	}
	return errors.Join(rollbackErrors...)
}

// Helpers

func (s *Service) publishLowStockEvent(item *domain.PantryItem) {
	if s.bus != nil {
		_ = s.bus.Publish(context.Background(), events.Event{
			ID:         uuid.NewString(),
			Type:       events.PantryLowStock,
			OccurredAt: time.Now().UTC(),
			Payload:    item,
			Source:     "pantry_service",
		})
	}

	// Add to shopping list if shopping integration is present
	if s.shoppingApp != nil {
		qtyStr := fmt.Sprintf("%.0f %s", math.Max(1.0, item.MinThreshold*2-item.Quantity), item.Unit)
		_ = s.shoppingApp.AddItem(context.Background(), item.Name, qtyStr, string(item.Category), "Dispensa Automatica")
	}
}
