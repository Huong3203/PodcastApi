package config

import "os"

type MoMoConfig struct {
	PartnerCode    string
	AccessKey      string
	SecretKey      string
	CreateEndpoint string
	RedirectURL    string
	IPNUrl         string
	IsSandbox      bool
}

func GetMoMoConfig() MoMoConfig {
	// Lấy giá trị environment hoặc mặc định
	createEndpoint := getEnvOrDefault("MOMO_CREATE_ENDPOINT", "https://test-payment.momo.vn/v2/gateway/api/create")
	isSandbox := getEnvOrDefault("MOMO_SANDBOX", "true") == "true"

	return MoMoConfig{
		PartnerCode:    getEnvOrDefault("MOMO_PARTNER_CODE", "MOMO"),
		AccessKey:      getEnvOrDefault("MOMO_ACCESS_KEY", "F8BBA842ECF85"),
		SecretKey:      getEnvOrDefault("MOMO_SECRET_KEY", "K951B6PE1waDMi640xX08PD3vg6EkVlz"),
		CreateEndpoint: createEndpoint,
		RedirectURL:    getEnvOrDefault("MOMO_REDIRECT_URL", "sonifyapp://payment-result"),
		IPNUrl:         getEnvOrDefault("MOMO_IPN_URL", "https://podcastapi.onrender.com/api/payment/momo/ipn"),
		IsSandbox:      isSandbox,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
