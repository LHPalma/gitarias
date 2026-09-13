package cmd

type weekdayCountsDocument struct {
	Weekdays []weekdayCountRecord `json:"weekdays"`
}
