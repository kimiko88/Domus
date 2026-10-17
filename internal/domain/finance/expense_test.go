package finance_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/finance"
)

func TestExpense_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		expense finance.Expense
		wantErr error
	}{
		{
			name: "valid grocery expense",
			expense: finance.Expense{
				Title:       "Spesa Esselunga",
				Amount:      84.50,
				Category:    finance.CategoryFood,
				Subcategory: finance.SubcategoryGroceries,
				ExpenseDate: now,
			},
			wantErr: nil,
		},
		{
			name: "valid utility bill with kwh",
			expense: finance.Expense{
				Title:       "Bolletta Enel Energia",
				Amount:      135.20,
				Category:    finance.CategoryUtilities,
				Subcategory: finance.SubcategoryElectricity,
				ExpenseDate: now,
				EnergyKWh:   450.0,
			},
			wantErr: nil,
		},
		{
			name: "empty title error",
			expense: finance.Expense{
				Title:       "   ",
				Amount:      50.0,
				Category:    finance.CategoryTransport,
				ExpenseDate: now,
			},
			wantErr: finance.ErrInvalidTitle,
		},
		{
			name: "zero or negative amount error",
			expense: finance.Expense{
				Title:       "Rifornimento",
				Amount:      0,
				Category:    finance.CategoryTransport,
				ExpenseDate: now,
			},
			wantErr: finance.ErrInvalidAmount,
		},
		{
			name: "invalid category error",
			expense: finance.Expense{
				Title:       "Spesa misteriosa",
				Amount:      25.0,
				Category:    "criptovalute",
				ExpenseDate: now,
			},
			wantErr: finance.ErrInvalidCategory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.expense.Validate()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.expense.Currency != "EUR" {
					t.Errorf("expected default currency EUR, got %s", tt.expense.Currency)
				}
			}
		})
	}
}
