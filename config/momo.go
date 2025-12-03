package config

import "os"

type MoMoConfig struct {
	PartnerCode    string
	AccessKey      string
	SecretKey      string
	CreateEndpoint string
	RedirectURL    string
	IPNUrl         string
}

func GetMoMoConfig() MoMoConfig {
	return MoMoConfig{
		PartnerCode:    getEnvOrDefault("MOMO_PARTNER_CODE", "MOMO"),
		AccessKey:      getEnvOrDefault("MOMO_ACCESS_KEY", "F8BBA842ECF85"),
		SecretKey:      getEnvOrDefault("MOMO_SECRET_KEY", "K951B6PE1waDMi640xX08PD3vg6EkVlz"),
		CreateEndpoint: getEnvOrDefault("MOMO_CREATE_ENDPOINT", "https://test-payment.momo.vn/v2/gateway/api/create"),
		RedirectURL:    getEnvOrDefault("MOMO_REDIRECT_URL", "http://localhost:8080/payment/momo/callback"),
		IPNUrl:         getEnvOrDefault("MOMO_IPN_URL", "http://localhost:8080/payment/momo/ipn"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
