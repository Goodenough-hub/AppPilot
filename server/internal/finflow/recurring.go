package finflow

import (
	"fmt"
	"time"
)

// computeNextDate 计算下一次触发日期，参考 PWA web/src/services/recurring.ts
func computeNextDate(current string, frequency string, interval int, dayOfMonth, dayOfWeek *int) (string, error) {
	cur, err := time.Parse("2006-01-02", current)
	if err != nil {
		return "", fmt.Errorf("invalid date: %w", err)
	}
	if interval <= 0 {
		interval = 1
	}
	switch frequency {
	case "daily":
		return cur.AddDate(0, 0, interval).Format("2006-01-02"), nil
	case "weekly":
		return cur.AddDate(0, 0, 7*interval).Format("2006-01-02"), nil
	case "monthly":
		return cur.AddDate(0, interval, 0).Format("2006-01-02"), nil
	case "yearly":
		return cur.AddDate(interval, 0, 0).Format("2006-01-02"), nil
	default:
		return "", fmt.Errorf("unknown frequency: %s", frequency)
	}
}

// ProcessDue 处理所有到期 recurring：生成 transaction + 推进 next_date
func (r *Repository) ProcessDue(userID int64) (int, error) {
	today := time.Now().Format("2006-01-02")
	due, err := r.ListDueRecurring(userID, today)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, rec := range due {
		tx := Transaction{
			Amount:     rec.Amount,
			Type:       rec.Type,
			Note:       rec.Note,
			Date:       rec.NextDate,
			CategoryID: rec.CategoryID,
			AccountID:  rec.AccountID,
			ToAccount:  rec.ToAccount,
			SourceID:   strPtr(fmt.Sprintf("recurring:%s", rec.ID)),
			SourceType: strPtr("recurring"),
		}
		if _, err := r.CreateTransaction(userID, tx); err != nil {
			return count, err
		}
		next, err := computeNextDate(rec.NextDate, rec.Frequency, rec.Interval, rec.DayOfMonth, rec.DayOfWeek)
		if err != nil {
			return count, err
		}
		if rec.EndDate != nil && next > *rec.EndDate {
			// 到结束日期，停用
			_ = r.AdvanceRecurringNextDate(parseID(rec.ID), next)
		} else {
			if err := r.AdvanceRecurringNextDate(parseID(rec.ID), next); err != nil {
				return count, err
			}
		}
		count++
	}
	return count, nil
}

func strPtr(s string) *string { return &s }

func parseID(s string) int64 {
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}
