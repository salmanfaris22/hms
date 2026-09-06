package model

type Overview struct {
	TotalPatients    int     `json:"totalPatients"`
	TotalStaff       int     `json:"totalStaff"`
	ActiveToday      int     `json:"activeToday"`
	TotalClinics     int     `json:"totalClinics"`
	TotalRevenue     float64 `json:"totalRevenue"`
	TotalExpenses    float64 `json:"totalExpenses"`
	NetProfit        float64 `json:"netProfit"`
	OutstandingAR    float64 `json:"outstandingAr"`
	CashInHand       float64 `json:"cashInHand"`
	StorageUsedGB    float64 `json:"storageUsedGb"`
	StorageQuotaGB   float64 `json:"storageQuotaGb"`
	AppointmentsToday int    `json:"appointmentsToday"`
	AppointmentsWeek  int    `json:"appointmentsWeek"`
	LowStockCount    int     `json:"lowStockCount"`
	ExpiringSoonCount int    `json:"expiringSoonCount"`
	Rating           float64 `json:"rating"`
	ReviewsCount     int     `json:"reviewsCount"`
	Period           string  `json:"period"`

	RevenueTrend     []TrendPoint    `json:"revenueTrend"`
	ExpenseTrend     []TrendPoint    `json:"expenseTrend"`
	IncomeByCategory []CategorySlice `json:"incomeByCategory"`
	TopDepartments   []DeptRow       `json:"topDepartments"`
	RecentActivity   []ActivityRow   `json:"recentActivity"`
}

type TrendPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type CategorySlice struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color string  `json:"color"`
}

type DeptRow struct {
	Name        string `json:"name"`
	StaffCount  int    `json:"staffCount"`
	PatientCount int   `json:"patientCount"`
}

type ActivityRow struct {
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"createdAt"`
}
