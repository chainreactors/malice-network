package configs

// LicenseRegistrationData 许可证注册数据结构
type LicenseRegistrationData struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Type       string `json:"type"`
	MaxBuilds  int    `json:"max_builds"`
	BuildCount int    `json:"build_count"`
}

// LicenseResponse SaaS API响应结构
type LicenseResponse struct {
	Success bool `json:"success"`
	License struct {
		Token string `json:"Token"`
	} `json:"license"`
}
