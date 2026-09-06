package model

type PathologyTestDTO struct {
	ID             string `json:"id"`
	ClinicID       string `json:"clinicId"`
	TestName       string `json:"testName"`
	TestCode       string `json:"testCode"`
	Category       string `json:"category"`
	SampleType     string `json:"sampleType"`
	Price          int    `json:"price"`
	Tax            string `json:"tax"`
	ParameterCount int    `json:"parameterCount"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type CreatePathologyTestRequest struct {
	ClinicID   string `json:"clinicId"`
	TestName   string `json:"testName"`
	TestCode   string `json:"testCode"`
	Category   string `json:"category"`
	SampleType string `json:"sampleType"`
	Price      int    `json:"price"`
	Tax        string `json:"tax"`
}

type PathologyListFilter struct {
	ClinicID string
	Page     int
	PageSize int
	Search   string
	Category string
}

type PatientGroupRange struct {
	PatientCategory string `json:"patientCategory"`
	Low             string `json:"low"`
	High            string `json:"high"`
}

type PathologyParameterDTO struct {
	ID             string              `json:"id"`
	TestID         string              `json:"testId"`
	ParameterName  string              `json:"parameterName"`
	Unit           string              `json:"unit"`
	NormalRangeMin string              `json:"normalRangeMin"`
	NormalRangeMax string              `json:"normalRangeMax"`
	Method         string              `json:"method"`
	PatientGroups  []PatientGroupRange `json:"patientGroups"`
	CreatedAt      string              `json:"createdAt"`
	UpdatedAt      string              `json:"updatedAt"`
}

type CreatePathologyParamRequest struct {
	TestID         string              `json:"testId"`
	ClinicID       string              `json:"clinicId"`
	ParameterName  string              `json:"parameterName"`
	Unit           string              `json:"unit"`
	NormalRangeMin string              `json:"normalRangeMin"`
	NormalRangeMax string              `json:"normalRangeMax"`
	Method         string              `json:"method"`
	PatientGroups  []PatientGroupRange `json:"patientGroups"`
}

type SyncTestParamsRequest struct {
	ParamIDs []string `json:"paramIds"`
}

type ParamGroupDTO struct {
	ID         string   `json:"id"`
	ClinicID   string   `json:"clinicId"`
	GroupName  string   `json:"groupName"`
	ParamIDs   []string `json:"paramIds"`
	ParamNames []string `json:"paramNames"`
	CreatedAt  string   `json:"createdAt"`
}

type SaveGroupRequest struct {
	ClinicID  string   `json:"clinicId"`
	GroupName string   `json:"groupName"`
	ParamIDs  []string `json:"paramIds"`
}
