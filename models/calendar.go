package models

import "time"

// Occurrence represents a bill or income event on a specific date
type Occurrence struct {
	Name   string
	Amount int
	Date   time.Time
	Type   string // "bill" or "income"
}

// CalendarDay represents a single day in the calendar view
type CalendarDay struct {
	Date        time.Time
	Entries     []Occurrence
	NetImpact   int
	IsToday     bool
	OtherMonth  bool
}

// ExpandOccurrences generates all dates an event occurs within [start, end]
func ExpandOccurrences(anchorDate time.Time, recurrence string, start, end time.Time) []time.Time {
	var dates []time.Time

	switch recurrence {
	case "once":
		if !anchorDate.Before(start) && !anchorDate.After(end) {
			dates = append(dates, anchorDate)
		}

	case "weekly":
		dates = expandByDays(anchorDate, 7, start, end)

	case "biweekly":
		dates = expandByDays(anchorDate, 14, start, end)

	case "monthly":
		dates = expandMonthly(anchorDate, start, end)
	}

	return dates
}

func expandByDays(anchor time.Time, interval int, start, end time.Time) []time.Time {
	var dates []time.Time

	// Find the first occurrence on or after start
	current := anchor
	if current.After(end) {
		return nil
	}

	if current.Before(start) {
		// Jump forward to near start
		daysBehind := int(start.Sub(current).Hours() / 24)
		jumps := daysBehind / interval
		current = current.AddDate(0, 0, jumps*interval)
		// Ensure we didn't overshoot before start
		for current.Before(start) {
			current = current.AddDate(0, 0, interval)
		}
	}

	for !current.After(end) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, interval)
	}

	return dates
}

func expandMonthly(anchor time.Time, start, end time.Time) []time.Time {
	var dates []time.Time
	day := anchor.Day()

	// Start from the anchor's month or earlier if needed
	current := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, time.Local)

	// Jump to start month if anchor is before start
	if current.Before(time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.Local)) {
		current = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	endMonth := time.Date(end.Year(), end.Month()+1, 0, 0, 0, 0, 0, time.Local)

	for !current.After(endMonth) {
		// Clamp day to last day of month
		lastDay := time.Date(current.Year(), current.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()
		d := day
		if d > lastDay {
			d = lastDay
		}

		date := time.Date(current.Year(), current.Month(), d, 0, 0, 0, 0, time.Local)
		if !date.Before(start) && !date.After(end) && !date.Before(anchor) {
			dates = append(dates, date)
		}

		current = current.AddDate(0, 1, 0)
	}

	return dates
}

// MonthlyIncomeTotal returns the sum of all active income expanded over a given month
func MonthlyIncomeTotal(year int, month time.Month) (int, error) {
	items, err := ListActiveIncome()
	if err != nil {
		return 0, err
	}

	start := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, -1)

	total := 0
	for _, item := range items {
		anchor, err := time.Parse("2006-01-02", item.DepositDate)
		if err != nil {
			continue
		}
		occurrences := ExpandOccurrences(anchor, item.Recurrence, start, end)
		total += item.Amount * len(occurrences)
	}
	return total, nil
}

// MonthlyBillsTotal returns the sum of all active bills expanded over a given month
func MonthlyBillsTotal(year int, month time.Month) (int, error) {
	bills, err := ListActiveBills()
	if err != nil {
		return 0, err
	}

	start := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, -1)

	total := 0
	for _, bill := range bills {
		anchor, err := time.Parse("2006-01-02", bill.DueDate)
		if err != nil {
			continue
		}
		occurrences := ExpandOccurrences(anchor, bill.Recurrence, start, end)
		total += bill.Amount * len(occurrences)
	}
	return total, nil
}

// UpcomingBillOccurrences returns the next N bill occurrences from today
func UpcomingBillOccurrences(limit int) ([]Occurrence, error) {
	bills, err := ListActiveBills()
	if err != nil {
		return nil, err
	}

	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 3, 0) // Look 3 months ahead

	var all []Occurrence
	for _, bill := range bills {
		anchor, err := time.Parse("2006-01-02", bill.DueDate)
		if err != nil {
			continue
		}
		for _, d := range ExpandOccurrences(anchor, bill.Recurrence, start, end) {
			all = append(all, Occurrence{
				Name:   bill.Name,
				Amount: bill.Amount,
				Date:   d,
				Type:   "bill",
			})
		}
	}

	// Sort by date
	sortOccurrences(all)

	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// UpcomingIncomeOccurrences returns the next N income occurrences from today
func UpcomingIncomeOccurrences(limit int) ([]Occurrence, error) {
	items, err := ListActiveIncome()
	if err != nil {
		return nil, err
	}

	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 3, 0)

	var all []Occurrence
	for _, item := range items {
		anchor, err := time.Parse("2006-01-02", item.DepositDate)
		if err != nil {
			continue
		}
		for _, d := range ExpandOccurrences(anchor, item.Recurrence, start, end) {
			all = append(all, Occurrence{
				Name:   item.Name,
				Amount: item.Amount,
				Date:   d,
				Type:   "income",
			})
		}
	}

	sortOccurrences(all)

	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// BuildCalendarMonth generates calendar data for a given month
func BuildCalendarMonth(year int, month time.Month) ([]CalendarDay, error) {
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	// Calendar grid starts on Sunday of the week containing the 1st
	gridStart := firstOfMonth
	for gridStart.Weekday() != time.Sunday {
		gridStart = gridStart.AddDate(0, 0, -1)
	}

	// Grid ends on Saturday of the week containing the last day
	gridEnd := lastOfMonth
	for gridEnd.Weekday() != time.Saturday {
		gridEnd = gridEnd.AddDate(0, 0, 1)
	}

	// Gather all occurrences for the grid range
	bills, err := ListActiveBills()
	if err != nil {
		return nil, err
	}
	incomeItems, err := ListActiveIncome()
	if err != nil {
		return nil, err
	}

	// Map date -> occurrences
	dayMap := make(map[string][]Occurrence)

	for _, bill := range bills {
		anchor, err := time.Parse("2006-01-02", bill.DueDate)
		if err != nil {
			continue
		}
		for _, d := range ExpandOccurrences(anchor, bill.Recurrence, gridStart, gridEnd) {
			key := d.Format("2006-01-02")
			dayMap[key] = append(dayMap[key], Occurrence{
				Name:   bill.Name,
				Amount: bill.Amount,
				Date:   d,
				Type:   "bill",
			})
		}
	}

	for _, item := range incomeItems {
		anchor, err := time.Parse("2006-01-02", item.DepositDate)
		if err != nil {
			continue
		}
		for _, d := range ExpandOccurrences(anchor, item.Recurrence, gridStart, gridEnd) {
			key := d.Format("2006-01-02")
			dayMap[key] = append(dayMap[key], Occurrence{
				Name:   item.Name,
				Amount: item.Amount,
				Date:   d,
				Type:   "income",
			})
		}
	}

	today := time.Now()
	todayStr := today.Format("2006-01-02")

	var days []CalendarDay
	current := gridStart
	for !current.After(gridEnd) {
		key := current.Format("2006-01-02")
		entries := dayMap[key]

		net := 0
		for _, e := range entries {
			if e.Type == "income" {
				net += e.Amount
			} else {
				net -= e.Amount
			}
		}

		days = append(days, CalendarDay{
			Date:       current,
			Entries:    entries,
			NetImpact:  net,
			IsToday:    key == todayStr,
			OtherMonth: current.Month() != month,
		})

		current = current.AddDate(0, 0, 1)
	}

	return days, nil
}

func sortOccurrences(occ []Occurrence) {
	// Simple insertion sort — small data sets
	for i := 1; i < len(occ); i++ {
		key := occ[i]
		j := i - 1
		for j >= 0 && occ[j].Date.After(key.Date) {
			occ[j+1] = occ[j]
			j--
		}
		occ[j+1] = key
	}
}
