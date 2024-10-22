package handler

type FormattedCallLog struct {
	PhoneNumber        string `json:"phone_number"`
	DisplayName        string `json:"display_name"`
	CalledPhoneNumber  string `json:"called_phone_number"`
	FormattedTimestamp string `json:"call_date"` // Formatted timestamp
}
