package model

import "time"

type PaymentSetting struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BankAccount struct {
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
	Icon          string `json:"icon,omitempty"`
}

type PaymentSettingsConfig struct {
	PakasirEnabled        bool          `json:"pakasir_enabled"`
	PakasirProjectSlug    string        `json:"pakasir_project_slug"`
	PakasirAPIKey         string        `json:"pakasir_api_key"`
	PakasirQRISOnly       bool          `json:"pakasir_qris_only"`
	MayarEnabled          bool          `json:"mayar_enabled"`
	MayarAPIKey           string        `json:"mayar_api_key"`
	MayarIsProduction     bool          `json:"mayar_is_production"`
	MayarWebhookToken     string        `json:"mayar_webhook_token"`
	IpaymuEnabled         bool          `json:"ipaymu_enabled"`
	IpaymuVA              string        `json:"ipaymu_va"`
	IpaymuAPIKey          string        `json:"ipaymu_api_key"`
	IpaymuIsProduction    bool          `json:"ipaymu_is_production"`
	ManualTransferEnabled bool          `json:"manual_transfer_enabled"`
	ManualBankAccounts    []BankAccount `json:"manual_bank_accounts"`
	ManualInstructions    string        `json:"manual_instructions"`
	ManualWhatsApp        string        `json:"manual_whatsapp"`
	DefaultGateway        string        `json:"default_gateway"` // "pakasir" | "mayar" | "ipaymu" | "manual" | "customer_choice"
}

type PublicPaymentMethod struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Badge        string        `json:"badge"`
	Type         string        `json:"type"` // "gateway" | "manual"
	IsEnabled    bool          `json:"is_enabled"`
	BankAccounts []BankAccount `json:"bank_accounts,omitempty"`
	Instructions string        `json:"instructions,omitempty"`
	WhatsApp     string        `json:"whatsapp,omitempty"`
}
