package util

var validNetworkIDs = []string{"214001", "222010", "234015", "262002", "262009", "286002"}
var validServiceIDs = []string{"antiSpam", "callForking", "callProtect", "volteRoaming", "verifiedBusiness"}

func CustomErrorResponse(status int, title string, detail string) (map[string]interface{}, int) {
	return map[string]interface{}{
		"status": status,
		"title":  title,
		"detail": detail,
	}, status
}

func Custom502Error() (map[string]interface{}, int) {
	return CustomErrorResponse(502, "Generic Error", "Unexpected status code 502")
}

func IsValidNetworkAndService(networkID string, serviceID string) bool {
	return StringInSlice(networkID, validNetworkIDs) && StringInSlice(serviceID, validServiceIDs)
}

func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
